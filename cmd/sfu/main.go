package main

import (
	"github.com/pion/ion/pkg/log"
)

func main() {
	log.Init("debug")
	log.Infof("--- Starting SFU Node ---")
	select {}
}
