// Package ice implements the Interactive Connectivity Establishment (ICE)
// protocol defined in rfc5245.
package ice

import (
	"context"
	"math/rand"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sssgun/ion/logging"
	"github.com/sssgun/ion/mdns"
	"github.com/sssgun/ion/stun"
	"github.com/sssgun/ion/transport/packetio"
	"github.com/sssgun/ion/transport/vnet"
)

const (
	// taskLoopInterval is the interval at which the agent performs checks
	defaultTaskLoopInterval = 2 * time.Second

	// keepaliveInterval used to keep candidates alive
	defaultKeepaliveInterval = 10 * time.Second

	// defaultConnectionTimeout used to declare a connection dead
	defaultConnectionTimeout = 30 * time.Second

	// timeout for candidate selection, after this time, the best candidate is used
	defaultCandidateSelectionTimeout = 10 * time.Second

	// wait time before nominating a host candidate
	defaultHostAcceptanceMinWait = 0

	// wait time before nominating a srflx candidate
	defaultSrflxAcceptanceMinWait = 500 * time.Millisecond

	// wait time before nominating a prflx candidate
	defaultPrflxAcceptanceMinWait = 1000 * time.Millisecond

	// wait time before nominating a relay candidate
	defaultRelayAcceptanceMinWait = 2000 * time.Millisecond

	// max binding request before considering a pair failed
	defaultMaxBindingRequests = 7

	// the number of bytes that can be buffered before we start to error
	maxBufferSize = 1000 * 1000 // 1MB

	// wait time before binding requests can be deleted
	maxBindingRequestTimeout = 500 * time.Millisecond
)

var (
	defaultCandidateTypes = []CandidateType{CandidateTypeHost, CandidateTypeServerReflexive, CandidateTypeRelay}
)

type bindingRequest struct {
	timestamp      time.Time
	transactionID  [stun.TransactionIDSize]byte
	destination    net.Addr
	isUseCandidate bool
}

// Agent represents the ICE agent
type Agent struct {
	// Lock for transactional operations on Agent. Unlike a mutex
	// all queued lock attempts are canceled when .Close() is called
	muChan chan struct{}

	onConnectionStateChangeHdlr       atomic.Value // func(ConnectionState)
	onSelectedCandidatePairChangeHdlr atomic.Value // func(Candidate, Candidate)
	onCandidateHdlr                   atomic.Value // func(Candidate)

	// Used to block double Dial/Accept
	opened bool

	// State owned by the taskLoop
	onConnected     chan struct{}
	onConnectedOnce sync.Once

	connectivityTicker *time.Ticker
	// force candidate to be contacted immediately (instead of waiting for connectivityTicker)
	forceCandidateContact chan bool

	trickle    bool
	tieBreaker uint64
	lite       bool

	connectionState ConnectionState
	gatheringState  GatheringState

	mDNSMode MulticastDNSMode
	mDNSName string
	mDNSConn *mdns.Conn

	haveStarted   atomic.Value
	isControlling bool

	maxBindingRequests uint16

	candidateSelectionTimeout time.Duration
	hostAcceptanceMinWait     time.Duration
	srflxAcceptanceMinWait    time.Duration
	prflxAcceptanceMinWait    time.Duration
	relayAcceptanceMinWait    time.Duration

	portmin uint16
	portmax uint16

	candidateTypes []CandidateType

	// How long should a pair stay quiet before we declare it dead?
	// 0 means never timeout
	connectionTimeout time.Duration

	// How often should we send keepalive packets?
	// 0 means never
	keepaliveInterval time.Duration

	// How after should we run our internal taskLoop
	taskLoopInterval time.Duration

	localUfrag      string
	localPwd        string
	localCandidates map[NetworkType][]Candidate

	remoteUfrag      string
	remotePwd        string
	remoteCandidates map[NetworkType][]Candidate

	checklist []*candidatePair
	selector  pairCandidateSelector

	selectedPair atomic.Value // *candidatePair

	urls         []*URL
	networkTypes []NetworkType

	buffer *packetio.Buffer

	// LRU of outbound Binding request Transaction IDs
	pendingBindingRequests []bindingRequest

	// 1:1 D-NAT IP address mapping
	extIPMapper *externalIPMapper

	// State for closing
	done chan struct{}
	err  atomicError

	chanCandidate chan Candidate
	chanState     chan ConnectionState

	loggerFactory logging.LoggerFactory
	log           logging.LeveledLogger

	net *vnet.Net

	interfaceFilter func(string) bool

	insecureSkipVerify bool
}

func (a *Agent) ok() error {
	select {
	case <-a.done:
		return a.getErr()
	default:
	}
	return nil
}

func (a *Agent) getErr() error {
	if err := a.err.Load(); err != nil {
		return err
	}
	return ErrClosed
}

// Run an operation with the the lock taken
// If the agent is closed return an error
func (a *Agent) run(t func(*Agent)) error {
	if err := a.ok(); err != nil {
		return err
	}

	select {
	case <-a.done:
		return a.getErr()
	case a.muChan <- struct{}{}:
		var err error
		select {
		case <-a.done:
			// Ensure the agent is not closed
			err = a.getErr()
		default:
			t(a)
		}
		<-a.muChan
		return err
	}
}

// AgentConfig collects the arguments to ice.Agent construction into
// a single structure, for future-proofness of the interface
type AgentConfig struct {
	Urls []*URL

	// PortMin and PortMax are optional. Leave them 0 for the default UDP port allocation strategy.
	PortMin uint16
	PortMax uint16

	// LocalUfrag and LocalPwd values used to perform connectivity
	// checks.  The values MUST be unguessable, with at least 128 bits of
	// random number generator output used to generate the password, and
	// at least 24 bits of output to generate the username fragment.
	LocalUfrag string
	LocalPwd   string

	// Trickle specifies whether or not ice agent should trickle candidates or
	// work perform synchronous gathering.
	Trickle bool

	// MulticastDNSMode controls mDNS behavior for the ICE agent
	MulticastDNSMode MulticastDNSMode

	// MulticastDNSHostName controls the hostname for this agent. If none is specified a random one will be generated
	MulticastDNSHostName string

	// ConnectionTimeout defaults to 30 seconds when this property is nil.
	// If the duration is 0, we will never timeout this connection.
	ConnectionTimeout *time.Duration
	// KeepaliveInterval determines how often should we send ICE
	// keepalives (should be less then connectiontimeout above)
	// when this is nil, it defaults to 10 seconds.
	// A keepalive interval of 0 means we never send keepalive packets
	KeepaliveInterval *time.Duration

	// NetworkTypes is an optional configuration for disabling or enabling
	// support for specific network types.
	NetworkTypes []NetworkType

	// CandidateTypes is an optional configuration for disabling or enabling
	// support for specific candidate types.
	CandidateTypes []CandidateType

	LoggerFactory logging.LoggerFactory

	// taskLoopInterval controls how often our internal task loop runs, this
	// task loop handles things like sending keepAlives. This is only value for testing
	// keepAlive behavior should be modified with KeepaliveInterval and ConnectionTimeout
	taskLoopInterval time.Duration

	// MaxBindingRequests is the max amount of binding requests the agent will send
	// over a candidate pair for validation or nomination, if after MaxBindingRequests
	// the candidate is yet to answer a binding request or a nomination we set the pair as failed
	MaxBindingRequests *uint16

	// CandidatesSelectionTimeout specify a timeout for selecting candidates, if no nomination has happen
	// before this timeout, once hit we will nominate the best valid candidate available,
	// or mark the connection as failed if no valid candidate is available
	CandidateSelectionTimeout *time.Duration

	// Lite agents do not perform connectivity check and only provide host candidates.
	Lite bool

	// NAT1To1IPCandidateType is used along with NAT1To1IPs to specify which candidate type
	// the 1:1 NAT IP addresses should be mapped to.
	// If unspecified or CandidateTypeHost, NAT1To1IPs are used to replace host candidate IPs.
	// If CandidateTypeServerReflexive, it will insert a srflx candidate (as if it was dervied
	// from a STUN server) with its port number being the one for the actual host candidate.
	// Other values will result in an error.
	NAT1To1IPCandidateType CandidateType

	// NAT1To1IPs contains a list of public IP addresses that are to be used as a host
	// candidate or srflx candidate. This is used typically for servers that are behind
	// 1:1 D-NAT (e.g. AWS EC2 instances) and to eliminate the need of server reflexisive
	// candidate gathering.
	NAT1To1IPs []string

	// HostAcceptanceMinWait specify a minimum wait time before selecting host candidates
	HostAcceptanceMinWait *time.Duration
	// HostAcceptanceMinWait specify a minimum wait time before selecting srflx candidates
	SrflxAcceptanceMinWait *time.Duration
	// HostAcceptanceMinWait specify a minimum wait time before selecting prflx candidates
	PrflxAcceptanceMinWait *time.Duration
	// HostAcceptanceMinWait specify a minimum wait time before selecting relay candidates
	RelayAcceptanceMinWait *time.Duration

	// Net is the our abstracted network interface for internal development purpose only
	// (see github.com/sssgun/ion/transport/vnet)
	Net *vnet.Net

	// InterfaceFilter is a function that you can use in order to  whitelist or blacklist
	// the interfaces which are used to gather ICE candidates.
	InterfaceFilter func(string) bool

	// InsecureSkipVerify controls if self-signed certificates are accepted when connecting
	// to TURN servers via TLS or DTLS
	InsecureSkipVerify bool
}

// NewAgent creates a new Agent
func NewAgent(config *AgentConfig) (*Agent, error) {
	var err error
	if config.PortMax < config.PortMin {
		return nil, ErrPort
	}

	// local username fragment and password
	localUfrag := randSeq(16)
	localPwd := randSeq(32)

	if config.LocalUfrag != "" {
		if len([]rune(config.LocalUfrag))*8 < 24 {
			return nil, ErrLocalUfragInsufficientBits
		}

		localUfrag = config.LocalUfrag
	}

	if config.LocalPwd != "" {
		if len([]rune(config.LocalPwd))*8 < 128 {
			return nil, ErrLocalPwdInsufficientBits
		}

		localPwd = config.LocalPwd
	}

	mDNSName := config.MulticastDNSHostName
	if mDNSName == "" {
		if mDNSName, err = generateMulticastDNSName(); err != nil {
			return nil, err
		}
	}

	if !strings.HasSuffix(mDNSName, ".local") || len(strings.Split(mDNSName, ".")) != 2 {
		return nil, ErrInvalidMulticastDNSHostName
	}

	mDNSMode := config.MulticastDNSMode
	if mDNSMode == 0 {
		mDNSMode = MulticastDNSModeQueryOnly
	}

	loggerFactory := config.LoggerFactory
	if loggerFactory == nil {
		loggerFactory = logging.NewDefaultLoggerFactory()
	}
	log := loggerFactory.NewLogger("ice")

	var mDNSConn *mdns.Conn
	mDNSConn, mDNSMode, err = createMulticastDNS(mDNSMode, mDNSName, log)
	// Opportunistic mDNS: If we can't open the connection, that's ok: we
	// can continue without it.
	if err != nil {
		log.Warnf("Failed to initialize mDNS %s: %v", mDNSName, err)
	}
	closeMDNSConn := func() {
		if mDNSConn != nil {
			if mdnsCloseErr := mDNSConn.Close(); mdnsCloseErr != nil {
				log.Warnf("Failed to close mDNS: %v", mdnsCloseErr)
			}
		}
	}

	a := &Agent{
		tieBreaker:             rand.New(rand.NewSource(time.Now().UnixNano())).Uint64(),
		lite:                   config.Lite,
		gatheringState:         GatheringStateNew,
		connectionState:        ConnectionStateNew,
		localCandidates:        make(map[NetworkType][]Candidate),
		remoteCandidates:       make(map[NetworkType][]Candidate),
		pendingBindingRequests: make([]bindingRequest, 0),
		checklist:              make([]*candidatePair, 0),
		urls:                   config.Urls,
		networkTypes:           config.NetworkTypes,
		localUfrag:             localUfrag,
		localPwd:               localPwd,
		onConnected:            make(chan struct{}),
		buffer:                 packetio.NewBuffer(),
		done:                   make(chan struct{}),
		chanState:              make(chan ConnectionState, 1),
		portmin:                config.PortMin,
		portmax:                config.PortMax,
		trickle:                config.Trickle,
		loggerFactory:          loggerFactory,
		log:                    log,
		net:                    config.Net,
		muChan:                 make(chan struct{}, 1),

		mDNSMode: mDNSMode,
		mDNSName: mDNSName,
		mDNSConn: mDNSConn,

		forceCandidateContact: make(chan bool, 1),

		interfaceFilter: config.InterfaceFilter,

		insecureSkipVerify: config.InsecureSkipVerify,
	}
	a.haveStarted.Store(false)

	if a.net == nil {
		a.net = vnet.NewNet(nil)
	} else if a.net.IsVirtual() {
		a.log.Warn("vnet is enabled")
		if a.mDNSMode != MulticastDNSModeDisabled {
			a.log.Warn("vnet does not support mDNS yet")
		}
	}

	a.initWithDefaults(config)

	// Make sure the buffer doesn't grow indefinitely.
	// NOTE: We actually won't get anywhere close to this limit.
	// SRTP will constantly read from the endpoint and drop packets if it's full.
	a.buffer.SetLimitSize(maxBufferSize)

	if a.lite && (len(a.candidateTypes) != 1 || a.candidateTypes[0] != CandidateTypeHost) {
		closeMDNSConn()
		return nil, ErrLiteUsingNonHostCandidates
	}

	if config.Urls != nil && len(config.Urls) > 0 && !containsCandidateType(CandidateTypeServerReflexive, a.candidateTypes) && !containsCandidateType(CandidateTypeRelay, a.candidateTypes) {
		closeMDNSConn()
		return nil, ErrUselessUrlsProvided
	}

	if err = a.initExtIPMapping(config); err != nil {
		closeMDNSConn()
		return nil, err
	}

	go func() {
		for s := range a.chanState {
			hdlr, ok := a.onConnectionStateChangeHdlr.Load().(func(ConnectionState))
			if ok {
				hdlr(s)
			}
		}
	}()

	// Initialize local candidates
	if !a.trickle {
		<-a.gatherCandidates()
	}
	return a, nil
}

// a sSeparate init routine called by NewAgent() to overcome gocyclo error with golangci-lint
func (a *Agent) initWithDefaults(config *AgentConfig) {
	if config.MaxBindingRequests == nil {
		a.maxBindingRequests = defaultMaxBindingRequests
	} else {
		a.maxBindingRequests = *config.MaxBindingRequests
	}

	if config.CandidateSelectionTimeout == nil {
		a.candidateSelectionTimeout = defaultCandidateSelectionTimeout
	} else {
		a.candidateSelectionTimeout = *config.CandidateSelectionTimeout
	}

	if config.HostAcceptanceMinWait == nil {
		a.hostAcceptanceMinWait = defaultHostAcceptanceMinWait
	} else {
		a.hostAcceptanceMinWait = *config.HostAcceptanceMinWait
	}

	if config.SrflxAcceptanceMinWait == nil {
		a.srflxAcceptanceMinWait = defaultSrflxAcceptanceMinWait
	} else {
		a.srflxAcceptanceMinWait = *config.SrflxAcceptanceMinWait
	}

	if config.PrflxAcceptanceMinWait == nil {
		a.prflxAcceptanceMinWait = defaultPrflxAcceptanceMinWait
	} else {
		a.prflxAcceptanceMinWait = *config.PrflxAcceptanceMinWait
	}

	if config.RelayAcceptanceMinWait == nil {
		a.relayAcceptanceMinWait = defaultRelayAcceptanceMinWait
	} else {
		a.relayAcceptanceMinWait = *config.RelayAcceptanceMinWait
	}

	// connectionTimeout used to declare a connection dead
	if config.ConnectionTimeout == nil {
		a.connectionTimeout = defaultConnectionTimeout
	} else {
		a.connectionTimeout = *config.ConnectionTimeout
	}

	if config.KeepaliveInterval == nil {
		a.keepaliveInterval = defaultKeepaliveInterval
	} else {
		a.keepaliveInterval = *config.KeepaliveInterval
	}

	if config.taskLoopInterval == 0 {
		a.taskLoopInterval = defaultTaskLoopInterval
	} else {
		a.taskLoopInterval = config.taskLoopInterval
	}

	if config.CandidateTypes == nil || len(config.CandidateTypes) == 0 {
		a.candidateTypes = defaultCandidateTypes
	} else {
		a.candidateTypes = config.CandidateTypes
	}
}

func (a *Agent) initExtIPMapping(config *AgentConfig) error {
	var err error
	a.extIPMapper, err = newExternalIPMapper(config.NAT1To1IPCandidateType, config.NAT1To1IPs)
	if err != nil {
		return err
	}
	if a.extIPMapper == nil {
		return nil // this may happen when config.NAT1To1IPs is an empty array
	}
	if a.extIPMapper.candidateType == CandidateTypeHost {
		if a.mDNSMode == MulticastDNSModeQueryAndGather {
			return ErrMulticastDNSWithNAT1To1IPMapping
		}
		candiHostEnabled := false
		for _, candiType := range a.candidateTypes {
			if candiType == CandidateTypeHost {
				candiHostEnabled = true
				break
			}
		}
		if !candiHostEnabled {
			return ErrIneffectiveNAT1To1IPMappingHost
		}
	} else if a.extIPMapper.candidateType == CandidateTypeServerReflexive {
		candiSrflxEnabled := false
		for _, candiType := range a.candidateTypes {
			if candiType == CandidateTypeServerReflexive {
				candiSrflxEnabled = true
				break
			}
		}
		if !candiSrflxEnabled {
			return ErrIneffectiveNAT1To1IPMappingSrflx
		}
	}
	return nil
}

// OnConnectionStateChange sets a handler that is fired when the connection state changes
func (a *Agent) OnConnectionStateChange(f func(ConnectionState)) error {
	a.onConnectionStateChangeHdlr.Store(f)
	return nil
}

// OnSelectedCandidatePairChange sets a handler that is fired when the final candidate
// pair is selected
func (a *Agent) OnSelectedCandidatePairChange(f func(Candidate, Candidate)) error {
	a.onSelectedCandidatePairChangeHdlr.Store(f)
	return nil
}

// OnCandidate sets a handler that is fired when new candidates gathered. When
// the gathering process complete the last candidate is nil.
func (a *Agent) OnCandidate(f func(Candidate)) error {
	a.onCandidateHdlr.Store(f)
	return nil
}

func (a *Agent) onSelectedCandidatePairChange(p *candidatePair) {
	if p != nil {
		if h, ok := a.onSelectedCandidatePairChangeHdlr.Load().(func(Candidate, Candidate)); ok {
			h(p.local, p.remote)
		}
	}
}

func (a *Agent) startConnectivityChecks(isControlling bool, remoteUfrag, remotePwd string) error {
	switch {
	case a.haveStarted.Load():
		return ErrMultipleStart
	case remoteUfrag == "":
		return ErrRemoteUfragEmpty
	case remotePwd == "":
		return ErrRemotePwdEmpty
	}

	a.haveStarted.Store(true)
	a.log.Debugf("Started agent: isControlling? %t, remoteUfrag: %q, remotePwd: %q", isControlling, remoteUfrag, remotePwd)

	return a.run(func(agent *Agent) {
		agent.isControlling = isControlling
		agent.remoteUfrag = remoteUfrag
		agent.remotePwd = remotePwd

		if isControlling {
			a.selector = &controllingSelector{agent: a, log: a.log}
		} else {
			a.selector = &controlledSelector{agent: a, log: a.log}
		}

		if a.lite {
			a.selector = &liteSelector{pairCandidateSelector: a.selector}
		}

		a.selector.Start()

		agent.updateConnectionState(ConnectionStateChecking)

		// TODO this should be dynamic, and grow when the connection is stable
		a.requestConnectivityCheck()
		agent.connectivityTicker = time.NewTicker(a.taskLoopInterval)

		go func() {
			contact := func() {
				if err := a.run(func(a *Agent) {
					a.selector.ContactCandidates()
				}); err != nil {
					a.log.Warnf("taskLoop failed: %v", err)
				}
			}

			for {
				select {
				case <-a.forceCandidateContact:
					contact()
				case <-a.connectivityTicker.C:
					contact()
				case <-a.done:
					return
				}
			}
		}()
	})
}

func (a *Agent) updateConnectionState(newState ConnectionState) {
	if a.connectionState != newState {
		a.log.Infof("Setting new connection state: %s", newState)
		a.connectionState = newState

		// Call handler in different routine since we may be holding the agent lock
		// and the handler may also require it
		a.chanState <- newState
	}
}

func (a *Agent) setSelectedPair(p *candidatePair) {
	a.log.Tracef("Set selected candidate pair: %s", p)
	// Notify when the selected pair changes
	a.onSelectedCandidatePairChange(p)

	if p != nil {
		p.nominated = true
		a.selectedPair.Store(p)
	} else {
		var nilPair *candidatePair
		a.selectedPair.Store(nilPair)
	}

	a.updateConnectionState(ConnectionStateConnected)

	// Close mDNS Conn. We don't need to do anymore querying
	// and no reason to respond to others traffic
	a.closeMulticastConn()

	// Signal connected
	a.onConnectedOnce.Do(func() { close(a.onConnected) })
}

func (a *Agent) pingAllCandidates() {
	a.log.Trace("pinging all candidates")

	if len(a.checklist) == 0 {
		a.log.Warn("pingAllCandidates called with no candidate pairs. Connection is not possible yet.")
	}

	for _, p := range a.checklist {
		if p.state == CandidatePairStateWaiting {
			p.state = CandidatePairStateInProgress
		} else if p.state != CandidatePairStateInProgress {
			continue
		}

		if p.bindingRequestCount > a.maxBindingRequests {
			a.log.Tracef("max requests reached for pair %s, marking it as failed\n", p)
			p.state = CandidatePairStateFailed
		} else {
			a.selector.PingCandidate(p.local, p.remote)
			p.bindingRequestCount++
		}
	}
}

func (a *Agent) getBestAvailableCandidatePair() *candidatePair {
	var best *candidatePair
	for _, p := range a.checklist {
		if p.state == CandidatePairStateFailed {
			continue
		}

		if best == nil {
			best = p
		} else if best.Priority() < p.Priority() {
			best = p
		}
	}
	return best
}

func (a *Agent) getBestValidCandidatePair() *candidatePair {
	var best *candidatePair
	for _, p := range a.checklist {
		if p.state != CandidatePairStateSucceeded {
			continue
		}

		if best == nil {
			best = p
		} else if best.Priority() < p.Priority() {
			best = p
		}
	}
	return best
}

func (a *Agent) addPair(local, remote Candidate) *candidatePair {
	p := newCandidatePair(local, remote, a.isControlling)
	a.checklist = append(a.checklist, p)
	return p
}

func (a *Agent) findPair(local, remote Candidate) *candidatePair {
	for _, p := range a.checklist {
		if p.local.Equal(local) && p.remote.Equal(remote) {
			return p
		}
	}
	return nil
}

// validateSelectedPair checks if the selected pair is (still) valid
// Note: the caller should hold the agent lock.
func (a *Agent) validateSelectedPair() bool {
	selectedPair := a.getSelectedPair()
	if selectedPair == nil {
		return false
	}

	if (a.connectionTimeout != 0) &&
		(time.Since(selectedPair.remote.LastReceived()) > a.connectionTimeout) {
		a.setSelectedPair(nil)
		a.updateConnectionState(ConnectionStateDisconnected)
		return false
	}

	return true
}

// checkKeepalive sends STUN Binding Indications to the selected pair
// if no packet has been sent on that pair in the last keepaliveInterval
// Note: the caller should hold the agent lock.
func (a *Agent) checkKeepalive() {
	selectedPair := a.getSelectedPair()
	if selectedPair == nil {
		return
	}

	if (a.keepaliveInterval != 0) &&
		(time.Since(selectedPair.local.LastSent()) > a.keepaliveInterval) {
		// we use binding request instead of indication to support refresh consent schemas
		// see https://tools.ietf.org/html/rfc7675
		a.selector.PingCandidate(selectedPair.local, selectedPair.remote)
	}
}

// AddRemoteCandidate adds a new remote candidate
func (a *Agent) AddRemoteCandidate(c Candidate) error {
	// If we have a mDNS Candidate lets fully resolve it before adding it locally
	if c.Type() == CandidateTypeHost && strings.HasSuffix(c.Address(), ".local") {
		if a.mDNSMode == MulticastDNSModeDisabled {
			a.log.Warnf("remote mDNS candidate added, but mDNS is disabled: (%s)", c.Address())
			return nil
		}

		hostCandidate, ok := c.(*CandidateHost)
		if !ok {
			return ErrAddressParseFailed
		}

		go a.resolveAndAddMulticastCandidate(hostCandidate)
		return nil
	}

	go func() {
		if err := a.run(func(agent *Agent) {
			agent.addRemoteCandidate(c)
		}); err != nil {
			a.log.Warnf("Failed to add remote candidate %s: %v", c.Address(), err)
			return
		}
	}()
	return nil
}

func (a *Agent) resolveAndAddMulticastCandidate(c *CandidateHost) {
	if a.mDNSConn == nil {
		return
	}
	_, src, err := a.mDNSConn.Query(context.TODO(), c.Address())
	if err != nil {
		a.log.Warnf("Failed to discover mDNS candidate %s: %v", c.Address(), err)
		return
	}

	ip, _, _, _ := parseAddr(src)
	if ip == nil {
		a.log.Warnf("Failed to discover mDNS candidate %s: failed to parse IP", c.Address())
		return
	}

	if err = c.setIP(ip); err != nil {
		a.log.Warnf("Failed to discover mDNS candidate %s: %v", c.Address(), err)
		return
	}

	if err = a.run(func(agent *Agent) {
		agent.addRemoteCandidate(c)
	}); err != nil {
		a.log.Warnf("Failed to add mDNS candidate %s: %v", c.Address(), err)
		return
	}
}

func (a *Agent) requestConnectivityCheck() {
	select {
	case a.forceCandidateContact <- true:
	default:
	}
}

// addRemoteCandidate assumes you are holding the lock (must be execute using a.run)
func (a *Agent) addRemoteCandidate(c Candidate) {
	set := a.remoteCandidates[c.NetworkType()]

	for _, candidate := range set {
		if candidate.Equal(c) {
			return
		}
	}

	set = append(set, c)
	a.remoteCandidates[c.NetworkType()] = set

	if localCandidates, ok := a.localCandidates[c.NetworkType()]; ok {
		for _, localCandidate := range localCandidates {
			a.addPair(localCandidate, c)
		}
	}

	a.requestConnectivityCheck()
}

func (a *Agent) addCandidate(c Candidate, candidateConn net.PacketConn) error {
	return a.run(func(agent *Agent) {
		c.start(a, candidateConn)

		set := a.localCandidates[c.NetworkType()]
		for _, candidate := range set {
			if candidate.Equal(c) {
				if err := c.close(); err != nil {
					a.log.Warnf("Failed to close duplicate candidate: %v", err)
				}
				return
			}
		}

		set = append(set, c)
		a.localCandidates[c.NetworkType()] = set

		if remoteCandidates, ok := a.remoteCandidates[c.NetworkType()]; ok {
			for _, remoteCandidate := range remoteCandidates {
				a.addPair(c, remoteCandidate)
			}
		}

		a.requestConnectivityCheck()

		a.chanCandidate <- c
	})
}

// GetLocalCandidates returns the local candidates
func (a *Agent) GetLocalCandidates() ([]Candidate, error) {
	res := make(chan []Candidate, 1)

	err := a.run(func(agent *Agent) {
		var candidates []Candidate
		for _, set := range agent.localCandidates {
			candidates = append(candidates, set...)
		}
		res <- candidates
	})
	if err != nil {
		return nil, err
	}

	return <-res, nil
}

// GetLocalUserCredentials returns the local user credentials
func (a *Agent) GetLocalUserCredentials() (frag string, pwd string) {
	return a.localUfrag, a.localPwd
}

// Close cleans up the Agent
func (a *Agent) Close() error {
	done := make(chan struct{})
	err := a.run(func(agent *Agent) {
		defer func() {
			close(done)
			close(agent.chanState)
		}()
		agent.err.Store(ErrClosed)
		close(agent.done)

		// Cleanup all candidates
		for net, cs := range agent.localCandidates {
			for _, c := range cs {
				err := c.close()
				if err != nil {
					a.log.Warnf("Failed to close candidate %s: %v", c, err)
				}
			}
			delete(agent.localCandidates, net)
		}
		for net, cs := range agent.remoteCandidates {
			for _, c := range cs {
				err := c.close()
				if err != nil {
					a.log.Warnf("Failed to close candidate %s: %v", c, err)
				}
			}
			delete(agent.remoteCandidates, net)
		}
		if err := a.buffer.Close(); err != nil {
			a.log.Warnf("failed to close buffer: %v", err)
		}

		if a.connectivityTicker != nil {
			a.connectivityTicker.Stop()
		}

		a.closeMulticastConn()
		a.updateConnectionState(ConnectionStateClosed)
	})
	if err != nil {
		return err
	}

	<-done
	return nil
}

func (a *Agent) findRemoteCandidate(networkType NetworkType, addr net.Addr) Candidate {
	ip, port, err := addrIPAndPort(addr)
	if err != nil {
		a.log.Warn(err.Error())
		return nil
	}

	set := a.remoteCandidates[networkType]
	for _, c := range set {
		if c.Address() == ip.String() && c.Port() == port {
			return c
		}
	}
	return nil
}

func (a *Agent) sendBindingRequest(m *stun.Message, local, remote Candidate) {
	a.log.Tracef("ping STUN from %s to %s\n", local.String(), remote.String())

	a.invalidatePendingBindingRequests(time.Now())
	a.pendingBindingRequests = append(a.pendingBindingRequests, bindingRequest{
		timestamp:      time.Now(),
		transactionID:  m.TransactionID,
		destination:    remote.addr(),
		isUseCandidate: m.Contains(stun.AttrUseCandidate),
	})

	a.sendSTUN(m, local, remote)
}

func (a *Agent) sendBindingSuccess(m *stun.Message, local, remote Candidate) {
	base := remote
	if out, err := stun.Build(m, stun.BindingSuccess,
		&stun.XORMappedAddress{
			IP:   base.addr().IP,
			Port: base.addr().Port,
		},
		stun.NewShortTermIntegrity(a.localPwd),
		stun.Fingerprint,
	); err != nil {
		a.log.Warnf("Failed to handle inbound ICE from: %s to: %s error: %s", local, remote, err)
	} else {
		a.sendSTUN(out, local, remote)
	}
}

/* Removes pending binding requests that are over maxBindingRequestTimeout old

   Let HTO be the transaction timeout, which SHOULD be 2*RTT if
   RTT is known or 500 ms otherwise.
   https://tools.ietf.org/html/rfc8445#appendix-B.1
*/
func (a *Agent) invalidatePendingBindingRequests(filterTime time.Time) {
	initialSize := len(a.pendingBindingRequests)

	temp := a.pendingBindingRequests[:0]
	for _, bindingRequest := range a.pendingBindingRequests {
		if filterTime.Sub(bindingRequest.timestamp) < maxBindingRequestTimeout {
			temp = append(temp, bindingRequest)
		}
	}

	a.pendingBindingRequests = temp
	if bindRequestsRemoved := initialSize - len(a.pendingBindingRequests); bindRequestsRemoved > 0 {
		a.log.Tracef("Discarded %d binding requests because they expired", bindRequestsRemoved)
	}
}

// Assert that the passed TransactionID is in our pendingBindingRequests and returns the destination
// If the bindingRequest was valid remove it from our pending cache
func (a *Agent) handleInboundBindingSuccess(id [stun.TransactionIDSize]byte) (bool, *bindingRequest) {
	a.invalidatePendingBindingRequests(time.Now())
	for i := range a.pendingBindingRequests {
		if a.pendingBindingRequests[i].transactionID == id {
			validBindingRequest := a.pendingBindingRequests[i]
			a.pendingBindingRequests = append(a.pendingBindingRequests[:i], a.pendingBindingRequests[i+1:]...)
			return true, &validBindingRequest
		}
	}
	return false, nil
}

// handleInbound processes STUN traffic from a remote candidate
func (a *Agent) handleInbound(m *stun.Message, local Candidate, remote net.Addr) {
	var err error
	if m == nil || local == nil {
		return
	}

	if m.Type.Method != stun.MethodBinding ||
		!(m.Type.Class == stun.ClassSuccessResponse ||
			m.Type.Class == stun.ClassRequest ||
			m.Type.Class == stun.ClassIndication) {
		a.log.Tracef("unhandled STUN from %s to %s class(%s) method(%s)", remote, local, m.Type.Class, m.Type.Method)
		return
	}

	if a.isControlling {
		if m.Contains(stun.AttrICEControlling) {
			a.log.Debug("inbound isControlling && a.isControlling == true")
			return
		} else if m.Contains(stun.AttrUseCandidate) {
			a.log.Debug("useCandidate && a.isControlling == true")
			return
		}
	} else {
		if m.Contains(stun.AttrICEControlled) {
			a.log.Debug("inbound isControlled && a.isControlling == false")
			return
		}
	}

	remoteCandidate := a.findRemoteCandidate(local.NetworkType(), remote)
	if m.Type.Class == stun.ClassSuccessResponse {
		if err = assertInboundMessageIntegrity(m, []byte(a.remotePwd)); err != nil {
			a.log.Warnf("discard message from (%s), %v", remote, err)
			return
		}

		if remoteCandidate == nil {
			a.log.Warnf("discard success message from (%s), no such remote", remote)
			return
		}

		a.selector.HandleSuccessResponse(m, local, remoteCandidate, remote)
	} else if m.Type.Class == stun.ClassRequest {
		if err = assertInboundUsername(m, a.localUfrag+":"+a.remoteUfrag); err != nil {
			a.log.Warnf("discard message from (%s), %v", remote, err)
			return
		} else if err = assertInboundMessageIntegrity(m, []byte(a.localPwd)); err != nil {
			a.log.Warnf("discard message from (%s), %v", remote, err)
			return
		}

		if remoteCandidate == nil {
			ip, port, networkType, ok := parseAddr(remote)
			if !ok {
				a.log.Errorf("Failed to create parse remote net.Addr when creating remote prflx candidate")
				return
			}

			prflxCandidateConfig := CandidatePeerReflexiveConfig{
				Network:   networkType.String(),
				Address:   ip.String(),
				Port:      port,
				Component: local.Component(),
				RelAddr:   "",
				RelPort:   0,
			}

			prflxCandidate, err := NewCandidatePeerReflexive(&prflxCandidateConfig)
			if err != nil {
				a.log.Errorf("Failed to create new remote prflx candidate (%s)", err)
				return
			}
			remoteCandidate = prflxCandidate

			a.log.Debugf("adding a new peer-reflexive candidate: %s ", remote)
			a.addRemoteCandidate(remoteCandidate)
		}

		a.log.Tracef("inbound STUN (Request) from %s to %s", remote.String(), local.String())

		a.selector.HandleBindingRequest(m, local, remoteCandidate)
	}

	if remoteCandidate != nil {
		remoteCandidate.seen(false)
	}
}

// validateNonSTUNTraffic processes non STUN traffic from a remote candidate,
// and returns true if it is an actual remote candidate
func (a *Agent) validateNonSTUNTraffic(local Candidate, remote net.Addr) bool {
	var isValidCandidate uint64
	if err := a.run(func(agent *Agent) {
		remoteCandidate := a.findRemoteCandidate(local.NetworkType(), remote)
		if remoteCandidate != nil {
			remoteCandidate.seen(false)
			atomic.AddUint64(&isValidCandidate, 1)
		}
	}); err != nil {
		a.log.Warnf("failed to validate remote candidate: %v", err)
	}

	return atomic.LoadUint64(&isValidCandidate) == 1
}

func (a *Agent) getSelectedPair() *candidatePair {
	selectedPair := a.selectedPair.Load()
	if selectedPair == nil {
		return nil
	}

	return selectedPair.(*candidatePair)
}

func (a *Agent) closeMulticastConn() {
	if a.mDNSConn != nil {
		if err := a.mDNSConn.Close(); err != nil {
			a.log.Warnf("failed to close mDNS Conn: %v", err)
		}
	}
}

// GetCandidatePairsStats returns a list of candidate pair stats
func (a *Agent) GetCandidatePairsStats() []CandidatePairStats {
	resultChan := make(chan []CandidatePairStats, 1)
	err := a.run(func(agent *Agent) {
		result := make([]CandidatePairStats, 0, len(agent.checklist))
		for _, cp := range agent.checklist {
			stat := CandidatePairStats{
				Timestamp:         time.Now(),
				LocalCandidateID:  cp.local.ID(),
				RemoteCandidateID: cp.remote.ID(),
				State:             cp.state,
				Nominated:         cp.nominated,
				// PacketsSent uint32
				// PacketsReceived uint32
				// BytesSent uint64
				// BytesReceived uint64
				// LastPacketSentTimestamp time.Time
				// LastPacketReceivedTimestamp time.Time
				// FirstRequestTimestamp time.Time
				// LastRequestTimestamp time.Time
				// LastResponseTimestamp time.Time
				// TotalRoundTripTime float64
				// CurrentRoundTripTime float64
				// AvailableOutgoingBitrate float64
				// AvailableIncomingBitrate float64
				// CircuitBreakerTriggerCount uint32
				// RequestsReceived uint64
				// RequestsSent uint64
				// ResponsesReceived uint64
				// ResponsesSent uint64
				// RetransmissionsReceived uint64
				// RetransmissionsSent uint64
				// ConsentRequestsSent uint64
				// ConsentExpiredTimestamp time.Time
			}
			result = append(result, stat)
		}
		resultChan <- result
	})
	if err != nil {
		a.log.Errorf("error getting candidate pairs stats %v", err)
		return []CandidatePairStats{}
	}
	return <-resultChan
}

// GetLocalCandidatesStats returns a list of local candidates stats
func (a *Agent) GetLocalCandidatesStats() []CandidateStats {
	resultChan := make(chan []CandidateStats, 1)
	err := a.run(func(agent *Agent) {
		result := make([]CandidateStats, 0, len(agent.localCandidates))
		for networkType, localCandidates := range agent.localCandidates {
			for _, c := range localCandidates {
				stat := CandidateStats{
					Timestamp:     time.Now(),
					ID:            c.ID(),
					NetworkType:   networkType,
					IP:            c.Address(),
					Port:          c.Port(),
					CandidateType: c.Type(),
					Priority:      c.Priority(),
					// URL string
					RelayProtocol: "udp",
					// Deleted bool
				}
				result = append(result, stat)
			}
		}
		resultChan <- result
	})
	if err != nil {
		a.log.Errorf("error getting candidate pairs stats %v", err)
		return []CandidateStats{}
	}
	return <-resultChan
}

// GetRemoteCandidatesStats returns a list of remote candidates stats
func (a *Agent) GetRemoteCandidatesStats() []CandidateStats {
	resultChan := make(chan []CandidateStats, 1)
	err := a.run(func(agent *Agent) {
		result := make([]CandidateStats, 0, len(agent.remoteCandidates))
		for networkType, localCandidates := range agent.remoteCandidates {
			for _, c := range localCandidates {
				stat := CandidateStats{
					Timestamp:     time.Now(),
					ID:            c.ID(),
					NetworkType:   networkType,
					IP:            c.Address(),
					Port:          c.Port(),
					CandidateType: c.Type(),
					Priority:      c.Priority(),
					// URL string
					RelayProtocol: "udp",
				}
				result = append(result, stat)
			}
		}
		resultChan <- result
	})
	if err != nil {
		a.log.Errorf("error getting candidate pairs stats %v", err)
		return []CandidateStats{}
	}
	return <-resultChan
}
