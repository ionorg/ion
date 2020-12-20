package server

import (
	"github.com/pion/ion/pkg/node/biz"
)

type signalConf struct {
	GRPC grpcConf `mapstructure:"grpc"`
}

// Config for server
type Config struct {
	biz.Config
	Signal signalConf `mapstructure:"signal"`
}

// signalConf represents signal server configuration
type grpcConf struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	Cert            string `mapstructure:"cert"`
	Key             string `mapstructure:"key"`
	AllowAllOrigins bool   `mapstructure:"allow_all_origins"`
}
