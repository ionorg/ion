package conf

import (
	"flag"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

var (
	cfg      = Config{}
	GRPC     = &cfg.GRPC
	Pipeline = &cfg.Pipeline
	Rtp      = &cfg.Rtp
	Log      = &cfg.Log
	CfgFile  = &cfg.CfgFile
)

func init() {
	if !cfg.parse() {
		showHelp()
		os.Exit(-1)
	}
}

type grpc struct {
	Port string `mapstructure:"port"`
}

type samplebuilder struct {
	AudioMaxLate uint16 `mapstructure:"audiomaxlate"`
	VideoMaxLate uint16 `mapstructure:"videomaxlate"`
}

type pipeline struct {
	SampleBuilder samplebuilder `mapstructure:"samplebuilder"`
	WebmSaver     webmsaver     `mapstructure:"webmsaver"`
}

type webmsaver struct {
	Enabled   bool   `mapstructure:"enabled"`
	Togglable bool   `mapstructure:"togglable"`
	DefaultOn bool   `mapstructure:"defaulton"`
	Path      string `mapstructure:"path"`
}

type log struct {
	Level string `mapstructure:"level"`
}

type rtp struct {
	Port    int    `mapstructure:"port"`
	KcpKey  string `mapstructure:"kcpkey"`
	KcpSalt string `mapstructure:"kcpsalt"`
}

// Config for base AVP
type Config struct {
	GRPC     grpc     `mapstructure:"grpc"`
	Pipeline pipeline `mapstructure:"pipeline"`
	Rtp      rtp      `mapstructure:"rtp"`
	Log      log      `mapstructure:"log"`
	CfgFile  string
}

func showHelp() {
	fmt.Printf("Usage:%s {params}\n", os.Args[0])
	fmt.Println("      -c {config file}")
	fmt.Println("      -h (show help info)")
}

func (c *Config) load() bool {
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
	err = viper.GetViper().Unmarshal(c)
	if err != nil {
		fmt.Printf("config file %s loaded failed. %v\n", c.CfgFile, err)
		return false
	}

	fmt.Printf("config %s load ok!\n", c.CfgFile)
	return true
}

func (c *Config) parse() bool {

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
