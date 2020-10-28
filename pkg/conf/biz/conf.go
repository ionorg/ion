package conf

import (
	"flag"
	"fmt"
	"os"

	"github.com/dgrijalva/jwt-go"
	"github.com/spf13/viper"
)

var (
	cfg    = config{}
	Global = &cfg.Global
	Log    = &cfg.Log
	Etcd   = &cfg.Etcd
	Nats   = &cfg.Nats
	Signal = &cfg.Signal
	Avp    = &cfg.Avp
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

type log struct {
	Level string `mapstructure:"level"`
}

type etcd struct {
	Addrs []string `mapstructure:"addrs"`
}

type signal struct {
	Host              string `mapstructure:"host"`
	Port              int    `mapstructure:"port"`
	Cert              string `mapstructure:"cert"`
	Key               string `mapstructure:"key"`
	WebSocketPath     string `mapstructure:"path"`
	AllowDisconnected bool   `mapstructure:"allow_disconnected"`

	AuthConnection AuthConfig `mapstructure:"auth_connection"`
	AuthRoom       AuthConfig `mapstructure:"auth_room"`
}

type AuthConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Key     string `mapstructure:"key"`
	KeyType string `mapstructure:"key_type"`
}

func (a AuthConfig) KeyFunc(t *jwt.Token) (interface{}, error) {
	switch a.KeyType {
	//TODO: add more support for keytypes here
	default:
		return []byte(a.Key), nil
	}
}

type nats struct {
	URL string `mapstructure:"url"`
}

type avp struct {
	Elements []string `mapstructure:"elements"`
}

type config struct {
	Global  global `mapstructure:"global"`
	Log     log    `mapstructure:"log"`
	Etcd    etcd   `mapstructure:"etcd"`
	Nats    nats   `mapstructure:"nats"`
	Signal  signal `mapstructure:"signal"`
	Avp     avp    `mapstructure:"avp"`
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
