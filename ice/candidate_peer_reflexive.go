// Package ice ...
//nolint:dupl
package ice

import "net"

// CandidatePeerReflexive ...
type CandidatePeerReflexive struct {
	candidateBase
}

// CandidatePeerReflexiveConfig is the config required to create a new CandidatePeerReflexive
type CandidatePeerReflexiveConfig struct {
	CandidateID string
	Network     string
	Address     string
	Port        int
	Component   uint16
	RelAddr     string
	RelPort     int
}

// NewCandidatePeerReflexive creates a new peer reflective candidate
func NewCandidatePeerReflexive(config *CandidatePeerReflexiveConfig) (*CandidatePeerReflexive, error) {
	ip := net.ParseIP(config.Address)
	if ip == nil {
		return nil, ErrAddressParseFailed
	}

	networkType, err := determineNetworkType(config.Network, ip)
	if err != nil {
		return nil, err
	}

	candidateID := config.CandidateID
	if candidateID == "" {
		var err error
		candidateID, err = generateCandidateID()
		if err != nil {
			return nil, err
		}
	}

	return &CandidatePeerReflexive{
		candidateBase: candidateBase{
			id:            candidateID,
			networkType:   networkType,
			candidateType: CandidateTypePeerReflexive,
			address:       config.Address,
			port:          config.Port,
			resolvedAddr:  &net.UDPAddr{IP: ip, Port: config.Port},
			component:     config.Component,
			relatedAddress: &CandidateRelatedAddress{
				Address: config.RelAddr,
				Port:    config.RelPort,
			},
		},
	}, nil
}
