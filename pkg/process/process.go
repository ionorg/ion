package process

import (
	"sync"
	"time"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/rtc/rtpengine"
	"github.com/pion/ion/pkg/rtc/transport"
)

const (
	statCycle = 3 * time.Second
)

var (
	pipelines    = make(map[string]*Pipeline)
	pipelineLock sync.RWMutex

	stop bool
)

// InitRTP rtp port
func InitRTP(port int, kcpKey, kcpSalt string) error {
	// show stat about all pipelines
	go check()

	var connCh chan *transport.RTPTransport
	var err error
	// accept relay rtptransport
	if kcpKey != "" && kcpSalt != "" {
		connCh, err = rtpengine.ServeWithKCP(port, kcpKey, kcpSalt)
	} else {
		connCh, err = rtpengine.Serve(port)
	}
	if err != nil {
		log.Errorf("process.InitRPC err=%v", err)
		return err
	}
	go func() {
		for {
			if stop {
				return
			}
			for rtpTransport := range connCh {
				go func(rtpTransport *transport.RTPTransport) {
					id := <-rtpTransport.IDChan

					if id == "" {
						log.Errorf("invalid id from incoming rtp transport")
						return
					}

					log.Infof("accept new rtp id=%s conn=%s", id, rtpTransport.RemoteAddr().String())
					addPipeline(id, rtpTransport)
				}(rtpTransport)
			}
		}
	}()
	return nil
}

func addPipeline(id string, pub transport.Transport) *Pipeline {
	log.Infof("process.addPipeline id=%s", id)
	pipelineLock.Lock()
	defer pipelineLock.Unlock()
	pipelines[id] = NewPipeline(id, pub)
	return pipelines[id]
}

// GetPipeline get pipeline from map
func GetPipeline(id string) *Pipeline {
	log.Infof("process.GetPipeline id=%s", id)
	pipelineLock.RLock()
	defer pipelineLock.RUnlock()
	return pipelines[id]
}

// DelPipeline delete pub
func DelPipeline(id string) {
	log.Infof("DelPipeline id=%s", id)
	pipeline := GetPipeline(id)
	if pipeline == nil {
		return
	}
	pipeline.Close()
	pipelineLock.Lock()
	defer pipelineLock.Unlock()
	delete(pipelines, id)
}

// Close close all pipelines
func Close() {
	if stop {
		return
	}
	stop = true
	pipelineLock.Lock()
	defer pipelineLock.Unlock()
	for id, pipeline := range pipelines {
		if pipeline != nil {
			pipeline.Close()
			delete(pipelines, id)
		}
	}
}

// check show all pipelines' stat
func check() {
	t := time.NewTicker(statCycle)
	for range t.C {
		info := "\n----------------process-----------------\n"
		pipelineLock.Lock()
		if len(pipelines) == 0 {
			pipelineLock.Unlock()
			continue
		}

		for id, pipeline := range pipelines {
			if !pipeline.Alive() {
				pipeline.Close()
				delete(pipelines, id)
				log.Infof("Stat delete %v", id)
			}
			info += "pipeline: " + id + "\n"
		}
		pipelineLock.Unlock()
		log.Infof(info)
	}
}
