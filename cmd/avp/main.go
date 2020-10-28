package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"path"

	iavp "github.com/pion/ion-avp/pkg"
	"github.com/pion/ion-avp/pkg/elements"
	conf "github.com/pion/ion/pkg/conf/avp"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/node/avp"
)

func init() {
	log.Init(conf.Avp.Log.Level)

	elems := make(map[string]iavp.ElementFun)
	if conf.Element.Webmsaver.On {
		elems["webmsaver"] = func(sid, pid, tid string, config []byte) iavp.Element {
			filewriter := elements.NewFileWriter(path.Join(conf.Element.Webmsaver.Path, fmt.Sprintf("%s-%s.webm", sid, pid)))
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

	rpcID := serviceNode.GetRPCChannel()
	eventID := serviceNode.GetEventChannel()
	avp.Init(conf.Global.Dc, serviceNode.NodeInfo().Info["id"], rpcID, eventID, conf.Nats.URL)

	select {}
}
