package conf

import (
	"fmt"
	"os"

	base "github.com/pion/ion-sfu/pkg/conf"
	"github.com/spf13/viper"
)

const (
	portRangeLimit = 100
)

var (
	cfg     = config{}
	Global  = base.Global
	Plugins = base.Plugins
	WebRTC  = base.WebRTC
	Rtp     = base.Rtp
	Log     = base.Log
	Router  = base.Router
	Etcd    = &cfg.Etcd
	Pprof   = &cfg.Pprof
)

func init() {
	if !cfg.load() {
		os.Exit(-1)
	}
}

type pprof struct {
	Port string `mapstructure:"port"`
}

type etcd struct {
	Addrs []string `mapstructure:"addrs"`
}

type config struct {
	Pprof pprof `mapstructure:"pprof"`
	Etcd  etcd  `mapstructure:"etcd"`
}

func (c *config) load() bool {
	path := *base.CfgFile
	_, err := os.Stat(path)
	if err != nil {
		return false
	}

	viper.SetConfigFile(path)
	viper.SetConfigType("toml")

	err = viper.ReadInConfig()
	if err != nil {
		fmt.Printf("config file %s read failed. %v\n", path, err)
		return false
	}

	err = viper.GetViper().Unmarshal(c)
	if err != nil {
		fmt.Printf("config file %s loaded failed. %v\n", path, err)
		return false
	}

	fmt.Printf("elements config %s load ok!\n", path)
	return true
}
