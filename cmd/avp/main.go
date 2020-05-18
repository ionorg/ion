package main

import (
	"net/http"
	_ "net/http/pprof"

	conf "github.com/pion/ion/pkg/conf/avp"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/node/avp"
	"github.com/pion/ion/pkg/rtc"
	"github.com/pion/ion/pkg/rtc/plugins"
)

func init() {
	log.Init(conf.Log.Level)
	if err := rtc.InitRTP(conf.Rtp.Port, conf.Rtp.KcpKey, conf.Rtp.KcpSalt); err != nil {
		panic(err)
	}

	pluginConfig := plugins.Config{
		On: conf.Plugins.On,
		SampleBuilder: plugins.SampleBuilderConfig{
			On:           conf.Plugins.SampleBuilder.On,
			AudioMaxLate: conf.Plugins.SampleBuilder.AudioMaxLate,
			VideoMaxLate: conf.Plugins.SampleBuilder.VideoMaxLate,
		},
		WebmSaver: plugins.WebmSaverConfig{
			On:   conf.Plugins.WebmSaver.On,
			Path: conf.Plugins.WebmSaver.Path,
		},
	}

	if err := rtc.CheckPlugins(pluginConfig); err != nil {
		panic(err)
	}
	rtc.InitPlugins(pluginConfig)
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
