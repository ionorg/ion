package avp

import (
	"encoding/json"
	"fmt"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/process"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
)

func handleRequest(rpcID string) {
	log.Infof("handleRequest: rpcID => [%v]", rpcID)
	protoo.OnRequest(rpcID, func(request nprotoo.Request, accept nprotoo.RespondFunc, reject nprotoo.RejectFunc) {
		method := request.Method
		data := request.Data
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

func startProcess(einfo proto.ElementInfo) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("process einfo=%v", einfo)
	pipeline := process.GetPipeline(einfo.MID)
	if pipeline == nil {
		return nil, util.NewNpError(404, "process: pipeline not found")
	}
	pipeline.AddElement(einfo)
	return util.Map(), nil
}

func endProcess(msg proto.ElementInfo) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("publish unprocess=%v", msg)
	return util.Map(), nil
}
