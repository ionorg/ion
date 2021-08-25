package runner

import (
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/util"
	"google.golang.org/grpc"
)

// ConfigBase is a interface used by Service
type ConfigBase interface {
	Load(string) error
}

// Service is a interface
type Service interface {
	New() Service
	ConfigBase() ConfigBase
	StartGRPC(registrar grpc.ServiceRegistrar) error
	Close()
}

// New create a ServiceRunner
func New(options util.WrapperedServerOptions) *ServiceRunner {
	return &ServiceRunner{
		options:    options,
		grpcServer: grpc.NewServer(),
	}
}

// ServiceUnit contain a service and it's config file path
type ServiceUnit struct {
	Service    Service
	ConfigFile string
}

// ServiceRunner allow you run grpc service all-in-one
type ServiceRunner struct {
	options      util.WrapperedServerOptions
	grpcServer   *grpc.Server
	serviceUnits []ServiceUnit
}

// AddService start a grpc server on addr, and registe all serviceUints
func (s *ServiceRunner) AddService(serviceUnits ...ServiceUnit) error {
	log.Infof("ServiceRunner.AddService serviceUnits=%+v", serviceUnits)

	// Init all services
	for _, serviceUnit := range serviceUnits {
		// Init basic config
		conf := serviceUnit.Service.ConfigBase()
		err := conf.Load(serviceUnit.ConfigFile)
		if err != nil {
			log.Errorf("config load error: %v", err)
			return err
		}

		// Registe service to grpc server
		err = serviceUnit.Service.StartGRPC(s.grpcServer)
		if err != nil {
			log.Errorf("Init service error: %v", err)
			return err
		}
		s.serviceUnits = append(s.serviceUnits, serviceUnit)
	}

	wrapperedSrv := util.NewWrapperedGRPCWebServer(s.options, s.grpcServer)
	if err := wrapperedSrv.Serve(); err != nil {
		log.Panicf("failed to serve: %v", err)
		return err
	}
	return nil
}

// Close close all services
func (s *ServiceRunner) Close() {
	for _, u := range s.serviceUnits {
		u.Service.Close()
	}
}
