package main

import (
	"flag"

	log "github.com/pion/ion-log"
	biz "github.com/pion/ion/apps/biz/server"
	"github.com/pion/ion/pkg/node/sfu"

	"github.com/pion/ion/pkg/runner"
)

func main() {
	var bizConfFile, sfuConfFile, addr, loglevel string
	flag.StringVar(&bizConfFile, "bc", "", "biz config file")
	flag.StringVar(&sfuConfFile, "sc", "", "sfu config file")
	flag.StringVar(&addr, "addr", ":8443", "grpc listening addr")
	flag.StringVar(&loglevel, "l", "info", "log level")
	flag.Parse()
	if bizConfFile == "" && sfuConfFile == "" {
		flag.PrintDefaults()
		return
	}
	bizConf, sfuConf := biz.Config{}, sfu.Config{}
	err := bizConf.Load(bizConfFile)
	if err != nil {
		log.Errorf("config load error: %v", err)
		return
	}

	err = sfuConf.Load(sfuConfFile)
	if err != nil {
		log.Errorf("config load error: %v", err)
		return
	}

	log.Init(bizConf.Log.Level)
	log.Infof("--- Starting Biz Node ---")
	// defer node.Close()
	biz := biz.New(bizConf)
	defer biz.Close()
	sfu := sfu.New(sfuConf)
	err = runner.New().AddService(addr,
		runner.ServiceUnit{
			Service:    biz,
			ConfigFile: bizConfFile,
		},
		runner.ServiceUnit{
			Service:    sfu,
			ConfigFile: sfuConfFile,
		})
	if err != nil {
		log.Errorf("runner AddService error: %v", err)
		return
	}
	select {}
}
