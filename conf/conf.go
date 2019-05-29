package conf

import (
	"flag"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

var (
	cfg    = config{}
	SFU    = &cfg.SFU
	Log    = &cfg.Log
	Etcd   = &cfg.Etcd
	Signal = &cfg.Signal
)

func init() {
	if !cfg.parse() {
		panic("config init error!")
	}
}

type log struct {
	Level string `mapstructure:"level"`
}

type etcd struct {
	Servers []string `mapstructure:"servers"`
}

type signal struct {
	Host    string `mapstructure:"host"`
	Port    string `mapstructure:"port"`
	CertPem string `mapstructure:"certpem"`
	KeyPem  string `mapstructure:"keypem"`
}

type sfu struct {
	Ices   []string `mapstructure:"ices"`
	Single bool     `mapstructure:"single"`
	Pprof  string   `mapstructure:"pprof"`
}

type config struct {
	SFU     sfu    `mapstructure:"sfu"`
	Log     log    `mapstructure:"log"`
	Etcd    etcd   `mapstructure:"etcds"`
	Signal  signal `mapstructure:"signal"`
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
		panic(c.CfgFile + " didn't exist!")
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
		showHelp()
		return false
	}

	if *help {
		showHelp()
		return false
	}
	return true
}
