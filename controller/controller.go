package controller

import "github.com/gwos/tng/transit"

// Agent possible status
type StatusEnum int
const (
	Running StatusEnum = iota
	Stopped
	Unknown
	Pending
)

type AgentStats struct {
	agentId string
	appType string
	bytesSent int
	metricsSent int
	messagesSent int
	lastInventoryRun MillisecondTimestamp
	lastMetricsRun MillisecondTimestamp
	executionTimeInventory time.Duration
	executionTimeMetrics time.Duration
	upSince MillisecondTimestamp
	lastError string
}

// TNG Control Plane interfaces
type ControllerServices interface {
	Start() (StatusEnum, error)
	Stop() (StatusEnum, error)
	Status() (StatusEnum, error)
	Stats() (*AgentStats, error)
	// LoadConfig() (StatusEnum, error)  // TODO: define configs to be passed in
	// ListConfig() (StatusEnum, error)  // TODO: define configs to be returned
}

type Controller struct {
	state StatusEnum
}

func CreateController() *Controller {
	return &Controller{state: Pending}
}

func (controller *Controller) Start() (StatusEnum, error) {
	controller.state = Running
	return controller.state, nil
}

func (controller *Controller) Stop() (StatusEnum, error) {
	controller.state = Stopped
	return controller.state, nil
}

func (controller *Controller) Status() (StatusEnum, error) {
	return controller.state, nil
}

func (controller *Controller) Stats() (*AgentStats, error) {
	return &AgentStats{
		agentId:                "agent 007",
		appType:				"nagios",
		bytesSent:              8192,
		metricsSent:            1024,
		messagesSent:           512,
		lastInventoryRun:       time.Time{},
		lastMetricsRun:         time.Time{},
		executionTimeInventory: 3949,
		executionTimeMetrics:   21934,
		upSince:                9393993,
		lastError:              "",
	}, nil
}
