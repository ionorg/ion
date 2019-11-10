package conf

import (
	"flag"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

var (
	cfg    = config{}
	Global = &cfg.Global
	Log    = &cfg.Log
	Etcd   = &cfg.Etcd
	Amqp   = &cfg.Amqp
	Redis  = &cfg.Redis
)

func init() {
	if !cfg.parse() {
		showHelp()
		os.Exit(-1)
	}
	fmt.Println("conf.init() ok")
}

type global struct {
	Pprof string `mapstructure:"pprof"`
	// TestIP []string `mapstructure:"testip"`
}

type log struct {
	Level string `mapstructure:"level"`
}

type etcd struct {
	Addrs []string `mapstructure:"addrs"`
}

type amqp struct {
	Url string `mapstructure:"url"`
}

type redis struct {
	Addrs []string `mapstructure:"addrs"`
	Pwd   string   `mapstructure:"password"`
	DB    int      `mapstructure:"db"`
}

type config struct {
	Global  global `mapstructure:"global"`
	Log     log    `mapstructure:"log"`
	Etcd    etcd   `mapstructure:"etcd"`
	Amqp    amqp   `mapstructure:"amqp"`
	Redis   redis  `mapstructure:"redis"`
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
