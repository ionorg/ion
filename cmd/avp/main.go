package main

import (
	"errors"
	"fmt"
	_ "net/http/pprof"
	"path"

	conf "github.com/pion/ion-avp/pkg/conf"
	"github.com/pion/ion-avp/pkg/elements"
	"github.com/pion/ion-avp/pkg/log"
	avp "github.com/pion/ion-avp/pkg/node"
	"github.com/pion/ion-avp/pkg/process"
	"github.com/pion/ion-avp/pkg/process/samples"
	pb "github.com/pion/ion-avp/pkg/proto/avp"
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

func getTogglableElement(e *pb.Element) (process.Element, error) {
	switch e.Type {
	case elements.TypeWebmSaver:
		filewriter := elements.NewFileWriter(elements.FileWriterConfig{
			ID:   e.Mid,
			Path: path.Join(conf.Pipeline.WebmSaver.Path, fmt.Sprintf("%s.webm", e.Mid)),
		})
		webm := elements.NewWebmSaver(elements.WebmSaverConfig{
			ID: e.Mid,
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
	avp.Init(conf.GRPC.Port)
	select {}
}
