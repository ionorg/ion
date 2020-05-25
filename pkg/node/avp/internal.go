package avp

import (
	"encoding/json"
	"fmt"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/node/avp/elements"
	"github.com/pion/ion/pkg/node/avp/process"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
)

func handleRequest(rpcID string) {
	log.Infof("handleRequest: rpcID => [%v]", rpcID)
	protoo.OnRequest(rpcID, func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
		method := request["method"].(string)
		data := json.RawMessage(request["data"].([]byte))
		log.Debugf("handleRequest: method => %s, data => %v", method, data)

		var proc proto.ElementInfo
		if err := json.Unmarshal(data, &proc); err != nil {
			reject(400, "Marshal error")
			return
		}

		var result map[string]interface{}
		err := util.NewNpError(400, fmt.Sprintf("Unkown method [%s]", method))

		switch method {
		case proto.AVPProcess:
			result, err = startProcess(proc)
		case proto.AVPUnprocess:
			result, err = endProcess(proc)
		}

		if err != nil {
			reject(err.Code, err.Reason)
		} else {
			accept(result)
		}
	})
}

func startProcess(msg proto.ElementInfo) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("process msg=%v", msg)
	pipeline := process.GetPipeline(msg.MID)
	if pipeline == nil {
		return nil, util.NewNpError(404, "process: pipeline not found")
	}
	element, err := elements.GetElement(msg)
	if err != nil {
		return nil, util.NewNpError(404, "process: element not found")
	}
	pipeline.AddElement(msg.Type, element)
	return util.Map(), nil
}

func endProcess(msg proto.ElementInfo) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("publish unprocess=%v", msg)
	return util.Map(), nil
}
