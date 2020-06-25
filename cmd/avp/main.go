package main

import (
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"path"

	elements "github.com/pion/ion-elements"
	conf "github.com/pion/ion/pkg/conf/avp"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/node/avp"
	"github.com/pion/ion/pkg/process"
	"github.com/pion/ion/pkg/process/samples"
	"github.com/pion/ion/pkg/proto"
)

func getDefaultElements(id string) map[string]process.Element {
	de := make(map[string]process.Element)
	if conf.Pipeline.WebmSaver.Enabled && conf.Pipeline.WebmSaver.DefaultOn {
		filewriter := elements.NewFileWriter(elements.FileWriterConfig{
			ID:   id,
			Path: path.Join(conf.Pipeline.WebmSaver.Path, fmt.Sprintf("%s.webm", id)),
		})
		webm := elements.NewWebmSaver(elements.WebmSaverConfig{
			ID: id,
		})
		err := webm.Attach(filewriter)
		if err != nil {
			log.Errorf("error attaching filewriter to webm %s", err)
		} else {
			de[elements.TypeWebmSaver] = webm
		}
	}
	return de
}

func getTogglableElement(msg proto.ElementInfo) (process.Element, error) {
	switch msg.Type {
	case elements.TypeWebmSaver:
		filewriter := elements.NewFileWriter(elements.FileWriterConfig{
			ID:   msg.MID,
			Path: path.Join(conf.Pipeline.WebmSaver.Path, fmt.Sprintf("%s.webm", msg.MID)),
		})
		webm := elements.NewWebmSaver(elements.WebmSaverConfig{
			ID: msg.MID,
		})
		err := webm.Attach(filewriter)
		if err != nil {
			log.Errorf("error attaching filewriter to webm %s", err)
			return nil, err
		}
		return webm, nil
	}

	return nil, errors.New("element not found")
}

func init() {
	log.Init(conf.Log.Level)
	if err := process.InitRTP(conf.Rtp.Port, conf.Rtp.KcpKey, conf.Rtp.KcpSalt); err != nil {
		panic(err)
	}

	process.InitPipeline(process.Config{
		SampleBuilder: samples.BuilderConfig{
			AudioMaxLate: conf.Pipeline.SampleBuilder.AudioMaxLate,
			VideoMaxLate: conf.Pipeline.SampleBuilder.VideoMaxLate,
		},
		GetDefaultElements:  getDefaultElements,
		GetTogglableElement: getTogglableElement,
	})
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
	serviceNode.RegisterNode("avp", "node-avp", "avp-channel-id", "")

	rpcID := serviceNode.GetRPCChannel()
	eventID := serviceNode.GetEventChannel()
	avp.Init(conf.Global.Dc, serviceNode.NodeInfo().Info["id"], rpcID, eventID, conf.Nats.URL)
	select {}
}
