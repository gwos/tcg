package controller

import (
	"github.com/gwos/tng/services"
	"github.com/gwos/tng/transit"
	"log"
)

// Agent possible status
type StatusEnum string

const (
	Running StatusEnum = "Running"
	Stopped            = "Stopped"
	Unknown            = "Unknown"
	Pending            = "Pending"
	userKey string     = "user"
)

func init() {
	err := StartServer(transit.Config.AgentConfig.SSL, transit.Config.AgentConfig.Port)
	if err != nil {
		log.Fatal(err)
	}
}

// TNG Control Plane interfaces
type Services interface {
	StartNATS() error
	StopNATS() error
	StartTransport() error
	StopTransport() error
	Stats() (*transit.AgentStats, error)
	// LoadConfig() (StatusEnum, error)  // TODO: define configs to be passed in
	// ListConfig() (StatusEnum, error)  // TODO: define configs to be returned
}

type Controller struct {
	NATSState      StatusEnum
	TransportState StatusEnum
}

var service services.Service

func NewController() *Controller {
	return &Controller{NATSState: Pending}
}

func (controller *Controller) StartNATS() error {
	err := service.StartNATS()
	if err != nil {
		return err
	}
	controller.NATSState = Running
	return nil
}

func (controller *Controller) StopNATS() error {
	service.StopNATS()
	controller.NATSState = Stopped
	return nil
}

func (controller *Controller) StartTransport() error {
	err := service.StartTransport()
	if err != nil {
		return err
	}
	controller.TransportState = Running
	return nil
}

func (controller *Controller) StopTransport() error {
	err := service.StopTransport()
	if err != nil {
		return err
	}
	controller.TransportState = Stopped
	return nil
}

func (controller Controller) Stats() (*transit.AgentStats, error) {
	return &transit.AgentStatistics, nil
}
