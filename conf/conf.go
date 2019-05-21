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
	Cfg.Parse()
}

type Mode struct {
	Standalone bool   `mapstructure:"standalone"`
	Signal     string `mapstructure:"signal"`
}

type Log struct {
	Level string `mapstructure:"level"`
}

type Etcd struct {
	Servers []string `mapstructure:"servers"`
}

type Centrifugo struct {
	Url     string `mapstructure:"url"`
	Key     string `mapstructure:"key"`
	CertPem string `mapstructure:"certpem"`
	KeyPem  string `mapstructure:"keypem"`
	Expire  int64  `mapstructure:"expire"`
}

type Protoo struct {
	Host    string `mapstructure:"host"`
	Port    string `mapstructure:"port"`
	CertPem string `mapstructure:"certpem"`
	KeyPem  string `mapstructure:"keypem"`
}

type SFU struct {
	Ices []string `mapstructure:"ices"`
}

type Config struct {
	Mode       Mode       `mapstructure:"mode"`
	Log        Log        `mapstructure:"log"`
	Etcd       Etcd       `mapstructure:"etcds"`
	Centrifugo Centrifugo `mapstructure:"centrifugo"`
	Protoo     Protoo     `mapstructure:"protoo"`
	Sfu        SFU        `mapstructure:"sfu"`
	CfgFile    string
	err        error
}

func ShowHelp() {
	fmt.Sprintf("Usage:%s {params}\n", os.Args[0])
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
		fmt.Errorf("config file %s read failed. %v", c.CfgFile, c.err)
		return false
	}
	c.err = viper.GetViper().UnmarshalExact(c)
	if c.err != nil {
		fmt.Errorf("config file %s loaded failed. %v", c.CfgFile, c.err)
		return false
	}

	fmt.Printf("config %s load ok!\n", c.CfgFile)
	return true
}

func (c *Config) Parse() bool {
	flag.StringVar(&c.CfgFile, "c", "conf/conf.toml", "config file")
	help := flag.Bool("h", false, "help info")
	flag.Parse()

	if !c.Load() || *help {
		ShowHelp()
		return false
	}
	return true
}
