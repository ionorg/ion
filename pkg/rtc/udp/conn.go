package udp

import (
	"bufio"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/pion/ion/pkg/log"
)

const receiveMTU = 8192

var errClosedListener = errors.New("udp: listener closed")

type UDPConn struct {
	r    *bufio.Reader
	conn *net.UDPConn
}

func newUDPConn(c *net.UDPConn) *UDPConn {
	return &UDPConn{bufio.NewReader(c), c}
}

func (u *UDPConn) Peek(n int) ([]byte, error) {
	return u.r.Peek(n)
}

func (u *UDPConn) Read(p []byte) (int, error) {
	return u.r.Read(p)
}

func (u *UDPConn) Close() error {
	return u.conn.Close()
}

func (u *UDPConn) LocalAddr() net.Addr {
	return u.conn.LocalAddr()
}

func (u *UDPConn) ReadFrom(b []byte) (int, net.Addr, error) {
	return u.conn.ReadFrom(b)
}

func (u *UDPConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	return u.conn.WriteTo(p, addr)
}

// Listener augments a connection-oriented Listener over a UDP PacketConn
type Listener struct {
	// pConn *net.UDPConn
	pConn *UDPConn

	lock      sync.RWMutex
	accepting bool
	acceptCh  chan *Conn
	doneCh    chan struct{}
	doneOnce  sync.Once

	conns map[string]*Conn
}

// Accept waits for and returns the next connection to the listener.
// You have to either close or read on all connection that are created.
func (l *Listener) Accept() (*Conn, error) {
	select {
	case c := <-l.acceptCh:
		return c, nil

	case <-l.doneCh:
		log.Infof("doneCh!!!!!!!!!!!!!!!")
		return nil, errClosedListener
	}
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (l *Listener) Close() error {
	l.lock.Lock()
	defer l.lock.Unlock()

	var err error
	l.doneOnce.Do(func() {
		l.accepting = false
		close(l.doneCh)
		err = l.cleanup()
	})

	return err
}

// cleanup closes the packet conn if it is no longer used
// The caller should hold the read lock.
func (l *Listener) cleanup() error {
	if !l.accepting && len(l.conns) == 0 {
		return l.pConn.Close()
	}
	return nil
}

// Addr returns the listener's network address.
func (l *Listener) Addr() net.Addr {
	return l.pConn.LocalAddr()
}

// Listen creates a new listener
func Listen(network string, laddr *net.UDPAddr) (*Listener, error) {
	conn, err := net.ListenUDP(network, laddr)
	// conn, err := net.Listen(network, laddr)
	if err != nil {
		return nil, err
	}

	l := &Listener{
		pConn:     newUDPConn(conn),
		acceptCh:  make(chan *Conn),
		conns:     make(map[string]*Conn),
		accepting: true,
		doneCh:    make(chan struct{}),
	}

	go l.readLoop()

	return l, nil
}

// readLoop has two tasks:
// 1. Dispatching incoming packets to the correct Conn.
//    It can therefore not be ended until all Conns are closed.
// 2. Creating a new Conn when receiving from a new remote.
func (l *Listener) readLoop() {
	buf := make([]byte, receiveMTU)

readLoop:

	for {
		n, raddr, err := l.pConn.ReadFrom(buf)
		if err != nil {
			log.Errorf("Listener.readLoop ReadFrom err=%v", err)
			return
		}
		conn, err := l.getConn(raddr)
		if err != nil {
			continue
		}
		select {
		case cBuf := <-conn.readCh:
			//copy buf to cBuf
			log.Debugf("Listener.readLoop copy")
			n = copy(cBuf, buf[:n])
			log.Debugf("Listener.readLoop copy n=%d", n)
			conn.sizeCh <- n
		case <-conn.doneCh:
			continue readLoop
		}
	}
}

func (l *Listener) getConn(raddr net.Addr) (*Conn, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	conn, ok := l.conns[raddr.String()]
	if !ok {
		if !l.accepting {
			return nil, errClosedListener
		}
		conn = l.newConn(raddr)
		l.conns[raddr.String()] = conn
		l.acceptCh <- conn
	}
	return conn, nil
}

// Conn augments a connection-oriented connection over a UDP PacketConn
type Conn struct {
	listener *Listener

	rAddr net.Addr

	readCh chan []byte
	sizeCh chan int

	lock     sync.RWMutex
	doneCh   chan struct{}
	doneOnce sync.Once
	// peek     []byte
}

func (l *Listener) newConn(rAddr net.Addr) *Conn {
	return &Conn{
		listener: l,
		rAddr:    rAddr,
		readCh:   make(chan []byte),
		sizeCh:   make(chan int),
		doneCh:   make(chan struct{}),
	}
}

// Read
func (c *Conn) Read(p []byte) (int, error) {
	select {
	case c.readCh <- p:
		// c.peek = p
		n := <-c.sizeCh
		return n, nil
	case <-c.doneCh:
		return 0, io.EOF
	}
}

// Peek
// func (c *Conn) Peek() []byte {
// return c.peek
// }

// Write writes len(p) bytes from p to the DTLS connection
func (c *Conn) Write(p []byte) (n int, err error) {
	c.lock.Lock()
	l := c.listener
	c.lock.Unlock()

	if l == nil {
		return 0, io.EOF
	}

	return l.pConn.WriteTo(p, c.rAddr)
}

// Close closes the conn and releases any Read calls
func (c *Conn) Close() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	var err error
	c.doneOnce.Do(func() {
		close(c.doneCh)
		c.listener.lock.Lock()
		delete(c.listener.conns, c.rAddr.String())
		err = c.listener.cleanup()
		c.listener.lock.Unlock()
		c.listener = nil
	})

	return err
}

// LocalAddr is a stub
func (c *Conn) LocalAddr() net.Addr {
	c.lock.Lock()
	l := c.listener
	c.lock.Unlock()

	if l == nil {
		return nil
	}

	return l.pConn.LocalAddr()
}

// RemoteAddr is a stub
func (c *Conn) RemoteAddr() net.Addr {
	return c.rAddr
}

// SetDeadline is a stub
func (c *Conn) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline is a stub
func (c *Conn) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline is a stub
func (c *Conn) SetWriteDeadline(t time.Time) error {
	return nil
}
