package turn

import "errors"

var (
	errRelayAddressInvalid        = errors.New("turn: RelayAddress must be valid IP to use RelayAddressGeneratorStatic")
	errNoAvailableConns           = errors.New("turn: PacketConnConfigs and ConnConfigs are empty, unable to proceed")
	errConnUnset                  = errors.New("turn: PacketConnConfig must have a non-nil Conn")
	errListenerUnset              = errors.New("turn: ListenerConfig must have a non-nil Listener")
	errListeningAddressInvalid    = errors.New("turn: RelayAddressGenerator has invalid ListeningAddress")
	errRelayAddressGeneratorUnset = errors.New("turn: RelayAddressGenerator in RelayConfig is unset")
)
