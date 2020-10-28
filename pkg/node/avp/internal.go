package avp

import (
	"fmt"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion-avp/pkg/log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
)

func handleRequest(rpcID string) {
	log.Infof("handleRequest: rpcID => [%v]", rpcID)
	protoo.OnRequest(rpcID, func(request nprotoo.Request, accept nprotoo.RespondFunc, reject nprotoo.RejectFunc) {
		method := request.Method
		data := request.Data
		log.Infof("handleRequest: method => %s, data => %s", method, data)

		var result interface{}
		errResult := util.NewNpError(400, fmt.Sprintf("Unknown method [%s]", method))

		switch method {
		case proto.AvpProcess:
			var msg proto.ToAvpProcessMsg
			if errResult = data.Unmarshal(&msg); errResult != nil {
				break
			}
			if err := s.Process(msg.Addr, msg.PID, msg.SID, msg.TID, msg.EID, msg.Config); err != nil {
				errResult = util.NewNpError(500, err.Error())
			}
			errResult = nil
		}

		if errResult != nil {
			reject(errResult.Code, errResult.Reason)
		} else {
			accept(result)
		}
	})
}
