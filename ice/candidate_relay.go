package ice

import (
	"net"
)

// CandidateRelay ...
type CandidateRelay struct {
	candidateBase

	onClose func() error
}

// CandidateRelayConfig is the config required to create a new CandidateRelay
type CandidateRelayConfig struct {
	CandidateID string
	Network     string
	Address     string
	Port        int
	Component   uint16
	RelAddr     string
	RelPort     int
	OnClose     func() error
}

// NewCandidateRelay creates a new relay candidate
func NewCandidateRelay(config *CandidateRelayConfig) (*CandidateRelay, error) {
	candidateID := config.CandidateID

	if candidateID == "" {
		var err error
		candidateID, err = generateCandidateID()
		if err != nil {
			return nil, err
		}
	}

	ip := net.ParseIP(config.Address)
	if ip == nil {
		return nil, ErrAddressParseFailed
	}

	networkType, err := determineNetworkType(config.Network, ip)
	if err != nil {
		return nil, err
	}

	return &CandidateRelay{
		candidateBase: candidateBase{
			id:            candidateID,
			networkType:   networkType,
			candidateType: CandidateTypeRelay,
			address:       config.Address,
			port:          config.Port,
			resolvedAddr:  &net.UDPAddr{IP: ip, Port: config.Port},
			component:     config.Component,
			relatedAddress: &CandidateRelatedAddress{
				Address: config.RelAddr,
				Port:    config.RelPort,
			},
		},
		onClose: config.OnClose,
	}, nil
}

func (c *CandidateRelay) close() error {
	err := c.candidateBase.close()
	if c.onClose != nil {
		err = c.onClose()
		c.onClose = nil
	}
	return err
}
