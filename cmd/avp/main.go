package main

import (
	"net/http"
	_ "net/http/pprof"

	conf "github.com/pion/ion/pkg/conf/avp"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/node/avp"
	"github.com/pion/ion/pkg/node/avp/elements"
	"github.com/pion/ion/pkg/node/avp/process"
	"github.com/pion/ion/pkg/node/avp/process/samplebuilder"
)

func init() {
	log.Init(conf.Log.Level)
	if err := process.InitRTP(conf.Rtp.Port, conf.Rtp.KcpKey, conf.Rtp.KcpSalt); err != nil {
		panic(err)
	}

	pipelineConfig := process.Config{
		SampleBuilder: samplebuilder.Config{
			AudioMaxLate: conf.Pipeline.SampleBuilder.AudioMaxLate,
			VideoMaxLate: conf.Pipeline.SampleBuilder.VideoMaxLate,
		},
		WebmSaver: elements.WebmSaverConfig{
			Togglable: conf.Pipeline.WebmSaver.Togglable,
			DefaultOn: conf.Pipeline.WebmSaver.DefaultOn,
			Path:      conf.Pipeline.WebmSaver.Path,
		},
	}

	process.InitPipeline(pipelineConfig)
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
