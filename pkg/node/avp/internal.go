package avp

import (
	"errors"

	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
)

func handleRequest(rpcID string) (*nats.Subscription, error) {
	log.Infof("handleRequest: rpcID => [%s]", rpcID)
	return nrpc.Subscribe(rpcID, func(msg interface{}) (interface{}, error) {
		log.Infof("handleRequest: %T, %+v", msg, msg)

		switch v := msg.(type) {
		case *proto.ToAvpProcessMsg:
			if err := s.Process(v.Addr, v.PID, v.RID, v.TID, v.EID, v.Config); err != nil {
				return nil, err
			}
		default:
			return nil, errors.New("unkonw message")
		}

		return nil, nil
	})
}
