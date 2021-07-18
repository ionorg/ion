package main

import (
	"flag"
	"fmt"
	"os"

	log "github.com/pion/ion-log"
	biz "github.com/pion/ion/apps/biz/server"
)

func showHelp() {
	fmt.Printf("Usage:%s {params}\n", os.Args[0])
	fmt.Println("      -c {config file}")
	fmt.Println("      -h (show help info)")
}

func main() {
	var file string
	flag.StringVar(&file, "c", "configs/app-biz.toml", "config file")
	help := flag.Bool("h", false, "help info")
	flag.Parse()
	conf := biz.Config{}
	err := conf.Load(file)
	if err != nil {
		log.Errorf("config load error: %v", err)
		return
	}

	if *help {
		showHelp()
		return
	}

	log.Init(conf.Log.Level)
	log.Infof("--- Starting Biz Node ---")
	node := biz.New(conf)
	if err := node.Start(); err != nil {
		log.Errorf("biz init start: %v", err)
		os.Exit(-1)
	}
	defer node.Close()
	select {}
}
