package conf

import (
	"flag"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

var (
	Cfg = &Config{}
)

func init() {
	if !Cfg.Parse() {
		panic("config init error!")
	}
}

type Mode struct {
	Single bool   `mapstructure:"single"`
	Pprof  string `mapstructure:"pprof"`
}

type Log struct {
	Level string `mapstructure:"level"`
}

type Etcd struct {
	Servers []string `mapstructure:"servers"`
}

type Signal struct {
	Host    string `mapstructure:"host"`
	Port    string `mapstructure:"port"`
	CertPem string `mapstructure:"certpem"`
	KeyPem  string `mapstructure:"keypem"`
}

type SFU struct {
	Ices []string `mapstructure:"ices"`
}

type Config struct {
	Mode    Mode   `mapstructure:"mode"`
	Log     Log    `mapstructure:"log"`
	Etcd    Etcd   `mapstructure:"etcds"`
	Signal  Signal `mapstructure:"signal"`
	Sfu     SFU    `mapstructure:"sfu"`
	CfgFile string
	err     error
}

func ShowHelp() {
	fmt.Printf("Usage:%s {params}\n", os.Args[0])
	fmt.Println("      -c {config file}")
	fmt.Println("      -h (show help info)")
}

func (c *Config) Load() bool {

	_, c.err = os.Stat(c.CfgFile)
	if c.err != nil {
		panic(c.CfgFile + " didn't exist!")
		return false
	}

	viper.SetConfigFile(c.CfgFile)
	viper.SetConfigType("toml")

	c.err = viper.ReadInConfig()
	if c.err != nil {
		fmt.Printf("config file %s read failed. %v\n", c.CfgFile, c.err)
		return false
	}
	c.err = viper.GetViper().UnmarshalExact(c)
	if c.err != nil {
		fmt.Printf("config file %s loaded failed. %v\n", c.CfgFile, c.err)
		return false
	}

	fmt.Printf("config %s load ok!\n", c.CfgFile)
	return true
}

func (c *Config) Parse() bool {
	flag.StringVar(&c.CfgFile, "c", "conf/conf.toml", "config file")
	help := flag.Bool("h", false, "help info")
	flag.Parse()
	if !c.Load() {
		ShowHelp()
		return false
	}

	if *help {
		ShowHelp()
		return false
	}
	return true
}
