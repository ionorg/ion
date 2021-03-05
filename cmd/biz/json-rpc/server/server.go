package server

import (
	"github.com/pion/ion-avp/pkg"
	"github.com/pion/ion/pkg/node/biz"
)

type signalConf struct {
	JsonRPC jsonRPCConf `mapstructure:"jsonrpc"`
}

// Config for server
type Config struct {
	biz.Config
	Signal signalConf `mapstructure:"signal"`
}

// Server represents server
type Server struct {
	biz    *biz.BIZ
	signal *Signal
	conf   Config
}

// NewServer create a server instance
func NewServer(nid string, conf Config) *Server {
	s := &Server{
		conf:   conf,
		biz:    biz.NewBIZ(nid),
		signal: newSignal(conf.Signal.JsonRPC),
	}
	return s
}

// Start server
func (s *Server) Start() error {
	bs, err := s.biz.Start(conf.Config)
	if err != nil {
		return err
	}
	s.signal.Start(bs)
	return nil
}

// Close server
func (s *Server) Close() {
	s.signal.close()
	s.biz.Close()
}
