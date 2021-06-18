package util

import (
	"context"

	nrpc "github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/cloudwebrtc/nats-grpc/pkg/rpc/reflection"
	"github.com/jhump/protoreflect/grpcreflect"
	log "github.com/pion/ion-log"
	rpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
)

// Get service information through reflection, this feature can be used to create json-rpc, restful API
func GetServiceInfo(nc nrpc.NatsConn, nid string, selfnid string) (map[string][]*reflection.MethodDescriptor, error) {
	ncli := nrpc.NewClient(nc, nid, selfnid)
	ctx, cancel := context.WithCancel(context.Background())
	rc := grpcreflect.NewClient(ctx, rpb.NewServerReflectionClient(ncli))

	defer func() {
		cancel()
		ncli.Close()
	}()

	reflector := reflection.NewReflector(rc)
	list, err := reflector.ListServices()
	if err != nil {
		log.Errorf("ListServices: error %v\n", err)
		return nil, err
	}

	log.Debugf("ListServices: %v", list)

	info := make(map[string][]*reflection.MethodDescriptor)
	for _, svc := range list {
		if svc == "grpc.reflection.v1alpha.ServerReflection" {
			continue
		}
		log.Debugf("Service => %v", svc)
		mds, err := reflector.DescribeService(svc)
		if err != nil {
			return nil, err
		}
		for _, md := range mds {
			log.Debugf("Method => %v", md.GetName())
		}
		info[svc] = mds
	}

	return info, nil
}
