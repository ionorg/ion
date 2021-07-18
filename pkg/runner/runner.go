package runner

import (
	"net"

	log "github.com/pion/ion-log"
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
func New() *ServiceRunner {
	return &ServiceRunner{}
}

// ServiceUnit contain a service and it's config file path
type ServiceUnit struct {
	Service    Service
	ConfigFile string
}

// ServiceRunner allow you run grpc service all-in-one
type ServiceRunner struct {
}

// AddService start a grpc server on addr, and registe all serviceUints
func (s *ServiceRunner) AddService(addr string, serviceUnits ...ServiceUnit) error {
	log.Infof("ServiceRunner.AddService addr=%v serviceUnits=%+v", addr, serviceUnits)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Panicf("failed to listen: %v", err)
	}

	log.Infof("--- Listening at %s ---", addr)
	grpcServer := grpc.NewServer()

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
		err = serviceUnit.Service.StartGRPC(grpcServer)
		if err != nil {
			log.Errorf("Init service error: %v", err)
			return err
		}
	}

	// Run grpc server
	if err := grpcServer.Serve(lis); err != nil {
		log.Panicf("failed to serve: %v", err)
	}
	return nil
}
