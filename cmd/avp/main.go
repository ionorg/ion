package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path"

	iavp "github.com/pion/ion-avp/pkg"
	"github.com/pion/ion-avp/pkg/elements"
	log "github.com/pion/ion-log"
	conf "github.com/pion/ion/pkg/conf/avp"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/node/avp"
)

func init() {
	fixByFile := []string{"asm_amd64.s", "proc.go", "icegatherer.go"}
	fixByFunc := []string{}
	log.Init(conf.Avp.Log.Level, fixByFile, fixByFunc)

	elems := make(map[string]iavp.ElementFun)
	if conf.Element.Webmsaver.On {
		if _, err := os.Stat(conf.Element.Webmsaver.Path); os.IsNotExist(err) {
			if err = os.MkdirAll(conf.Element.Webmsaver.Path, 0755); err != nil {
				log.Errorf("make dir error: %v", err)
			}
		}
		elems["webmsaver"] = func(rid, pid, tid string, config []byte) iavp.Element {
			filewriter := elements.NewFileWriter(path.Join(conf.Element.Webmsaver.Path, fmt.Sprintf("%s-%s.webm", rid, pid)))
			webm := elements.NewWebmSaver()
			webm.Attach(filewriter)
			return webm
		}
	}

	avp.InitAVP(conf.Avp, elems)
}

func main() {
	log.Infof("--- Starting AVP Node ---")

	if conf.Global.Pprof != "" {
		go func() {
			log.Infof("Start pprof on %s", conf.Global.Pprof)
			err := http.ListenAndServe(conf.Global.Pprof, nil)
			if err != nil {
				log.Errorf("http.ListenAndServe err=%v", err)
			}
		}()
	}

	serviceNode := discovery.NewServiceNode(conf.Etcd.Addrs, conf.Global.Dc)
	serviceNode.RegisterNode("avp", "node-avp", "avp-channel-id")
	avp.Init(conf.Global.Dc, serviceNode.NodeInfo().Info["id"], conf.Nats.URL)

	select {}
}
