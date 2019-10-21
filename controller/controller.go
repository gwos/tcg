package controller

import (
	"github.com/gwos/tng/services"
	"log"
)

// StatusEnum defines Agent Controller status
type StatusEnum string

// Agent Controller status
const (
	Running StatusEnum = "Running"
	Stopped            = "Stopped"
	Unknown            = "Unknown"
	Pending            = "Pending"
)

var service services.Service

func init() {
	err := StartServer(service.AgentConfig.SSL, service.AgentConfig.Port)
	if err != nil {
		log.Fatal(err)
	}
}

// Services defines TNG Control Plane interfaces
type Services interface {
	Identity(appName, apiToken string) error
	StartNATS() error
	StopNATS() error
	StartTransport() error
	StopTransport() error
	Stats() (*services.AgentStats, error)
	// LoadConfig() (StatusEnum, error)  // TODO: define configs to be passed in
	// ListConfig() (StatusEnum, error)  // TODO: define configs to be returned
}

// Controller implements Services interface
type Controller struct {
	NATSState      StatusEnum
	TransportState StatusEnum
}

// NewController creates instance
func NewController() *Controller {
	return &Controller{NATSState: Pending}
}

// StartNATS implements Services.StartNATS
func (controller *Controller) StartNATS() error {
	err := service.StartNATS()
	if err != nil {
		return err
	}
	controller.NATSState = Running
	return nil
}

// StopNATS implements Services.StopNATS
func (controller *Controller) StopNATS() error {
	service.StopNATS()
	controller.NATSState = Stopped
	return nil
}

// StartTransport implements Services.StartTransport
func (controller *Controller) StartTransport() error {
	err := service.StartTransport()
	if err != nil {
		return err
	}
	controller.TransportState = Running
	return nil
}

// StopTransport implements Services.StopTransport
func (controller *Controller) StopTransport() error {
	err := service.StopTransport()
	if err != nil {
		return err
	}
	controller.TransportState = Stopped
	return nil
}

// Stats implements Services.Stats
func (controller Controller) Stats() (*services.AgentStats, error) {
	return &service.AgentStats, nil
}

// Identity implements Services.Identity
func (controller Controller) Identity(appName, apiToken string) error {
	return service.Identity(appName, apiToken)
}
