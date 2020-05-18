package conf

import (
	"flag"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

var (
	cfg     = config{}
	Global  = &cfg.Global
	Plugins = &cfg.Plugins
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
}

type samplebuilder struct {
	On           bool   `mapstructure:"on"`
	AudioMaxLate uint16 `mapstructure:"audiomaxlate"`
	VideoMaxLate uint16 `mapstructure:"videomaxlate"`
}

type webmsaver struct {
	On   bool   `mapstructure:"on"`
	Path string `mapstructure:"path"`
}

type plugins struct {
	On            bool          `mapstructure:"on"`
	SampleBuilder samplebuilder `mapstructure:"samplebuilder"`
	WebmSaver     webmsaver     `mapstructure:"webmsaver"`
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

type rtp struct {
	Port    int    `mapstructure:"port"`
	KcpKey  string `mapstructure:"kcpkey"`
	KcpSalt string `mapstructure:"kcpsalt"`
}

type config struct {
	Global  global  `mapstructure:"global"`
	Plugins plugins `mapstructure:"plugins"`
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
