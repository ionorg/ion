package server

import (
	"github.com/pion/ion/pkg/node/biz"
)

// Config for server
type Config struct {
	biz.Config
	Signal signalConf `mapstructure:"signal"`
}

// Server represents server
type Server struct {
	conf   Config
	biz    *biz.BIZ
	signal *Signal
}

// NewServer create a server instance
func NewServer(conf Config) *Server {
	s := &Server{
		biz:    biz.NewBIZ(conf.Config),
		signal: newSignal(conf.Signal),
	}

	return s
}

// Start server
func (s *Server) Start() error {
	bs, err := s.biz.Start()
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
