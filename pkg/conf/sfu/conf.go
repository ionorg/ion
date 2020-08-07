package conf

import (
	"flag"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

const (
	portRangeLimit = 100
)

var (
	cfg     = config{}
	Global  = &cfg.Global
	Plugins = &cfg.Plugins
	WebRTC  = &cfg.WebRTC
	Rtp     = &cfg.Rtp
	Log     = &cfg.Log
	Etcd    = &cfg.Etcd
	Nats    = &cfg.Nats
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

type JitterBuffer struct {
	On            bool `mapstructure:"on"`
	TCCOn         bool `mapstructure:"tccon"`
	REMBCycle     int  `mapstructure:"rembcycle"`
	PLICycle      int  `mapstructure:"plicycle"`
	MaxBandwidth  int  `mapstructure:"maxbandwidth"`
	MaxBufferTime int  `mapstructure:"maxbuffertime"`
}

type RTPForwarder struct {
	On      bool   `mapstructure:"on"`
	Addr    string `mapstructure:"addr"`
	KcpKey  string `mapstructure:"kcpkey"`
	KcpSalt string `mapstructure:"kcpsalt"`
}

type plugins struct {
	On           bool         `mapstructure:"on"`
	JitterBuffer JitterBuffer `mapstructure:"jitterbuffer"`
	RTPForwarder RTPForwarder `mapstructure:"rtpforwarder"`
}

type log struct {
	Level string `mapstructure:"level"`
}

type etcd struct {
	Addrs []string `mapstructure:"addrs"`
}

type nats struct {
	URL string `mapstructure:"url"`
}

type iceserver struct {
	URLs       []string `mapstructure:"urls"`
	Username   string   `mapstructure:"username"`
	Credential string   `mapstructure:"credential"`
}

type webrtc struct {
	ICEPortRange []uint16    `mapstructure:"portrange"`
	ICEServers   []iceserver `mapstructure:"iceserver"`
}

type rtp struct {
	Port    int    `mapstructure:"port"`
	KcpKey  string `mapstructure:"kcpkey"`
	KcpSalt string `mapstructure:"kcpsalt"`
}

type config struct {
	Global  global  `mapstructure:"global"`
	Plugins plugins `mapstructure:"plugins"`
	WebRTC  webrtc  `mapstructure:"webrtc"`
	Rtp     rtp     `mapstructure:"rtp"`
	Log     log     `mapstructure:"log"`
	Etcd    etcd    `mapstructure:"etcd"`
	Nats    nats    `mapstructure:"nats"`
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
