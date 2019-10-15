package controller

import (
	"github.com/gwos/tng/transit"
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

// TNG Control Plane interfaces
type Services interface {
	Start() (StatusEnum, error)
	Stop() (StatusEnum, error)
	Status() (StatusEnum, error)
	Stats() (*transit.AgentStats, error)
	// LoadConfig() (StatusEnum, error)  // TODO: define configs to be passed in
	// ListConfig() (StatusEnum, error)  // TODO: define configs to be returned
}

type Controller struct {
	State StatusEnum
}

func NewController() *Controller {
	return &Controller{State: Pending}
}

func (controller *Controller) Start() (StatusEnum, error) {
	controller.State = Running
	return controller.State, nil
}

func (controller *Controller) Stop() (StatusEnum, error) {
	controller.State = Stopped
	return controller.State, nil
}

func (controller *Controller) Status() (StatusEnum, error) {
	return controller.State, nil
}

func (controller *Controller) Stats() (*transit.AgentStats, error) {
	return &transit.AgentStatistics, nil
}
