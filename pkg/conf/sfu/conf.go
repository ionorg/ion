package conf

import (
	"flag"
	"fmt"
	"os"

	log "github.com/pion/ion-log"
	sfu "github.com/pion/ion-sfu/pkg"
	"github.com/spf13/viper"
)

const (
	portRangeLimit = 100
)

var (
	cfg    = config{}
	Global = &cfg.Global
	WebRTC = &cfg.WebRTC
	Router = &cfg.Router
	Log    = &cfg.Log
	Etcd   = &cfg.Etcd
	Nats   = &cfg.Nats
)

func init() {
	if !cfg.parse() {
		showHelp()
		os.Exit(-1)
	}
}

type global struct {
	Addr  string `mapstructure:"addr"`
	Pprof string `mapstructure:"pprof"`
	Dc    string `mapstructure:"dc"`
	// TestIP []string `mapstructure:"testip"`
}

type etcd struct {
	Addrs []string `mapstructure:"addrs"`
}

type nats struct {
	URL string `mapstructure:"url"`
}

type config struct {
	Global  global           `mapstructure:"global"`
	WebRTC  sfu.WebRTCConfig `mapstructure:"webrtc"`
	Router  sfu.RouterConfig `mapstructure:"router"`
	Log     log.Config       `mapstructure:"log"`
	Etcd    etcd             `mapstructure:"etcd"`
	Nats    nats             `mapstructure:"nats"`
	CfgFile string
}

func showHelp() {
	fmt.Printf("Usage:%s {params}\n", os.Args[0])
	fmt.Println("      -c {config file}")
	fmt.Println("      -h (show help info)")
}

func (c *config) load() bool {
	_, err := os.Stat(c.CfgFile)
	if err != nil {
		return false
	}

	viper.SetConfigFile(c.CfgFile)
	viper.SetConfigType("toml")

	err = viper.ReadInConfig()
	if err != nil {
		fmt.Printf("config file %s read failed. %v\n", c.CfgFile, err)
		return false
	}
	err = viper.GetViper().UnmarshalExact(c)
	if err != nil {
		fmt.Printf("config file %s loaded failed. %v\n", c.CfgFile, err)
		return false
	}

	if len(c.WebRTC.ICEPortRange) > 2 {
		fmt.Printf("config file %s loaded failed. range port must be [min,max]\n", c.CfgFile)
		return false
	}

	if len(c.WebRTC.ICEPortRange) != 0 && c.WebRTC.ICEPortRange[1]-c.WebRTC.ICEPortRange[0] <= portRangeLimit {
		fmt.Printf("config file %s loaded failed. range port must be [min, max] and max - min >= %d\n", c.CfgFile, portRangeLimit)
		return false
	}

	fmt.Printf("config %s load ok!\n", c.CfgFile)
	return true
}

func (c *config) parse() bool {
	flag.StringVar(&c.CfgFile, "c", "conf/conf.toml", "config file")
	help := flag.Bool("h", false, "help info")
	flag.Parse()
	if !c.load() {
		return false
	}

	if *help {
		showHelp()
		return false
	}
	return true
}
