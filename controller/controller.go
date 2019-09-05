package controller

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

// Agent possible status
type StatusEnum int
const (
	Running StatusEnum = iota
	Stopped
	Unknown
	Pending
)

type AgentStats struct {
	AgentId string
	AppType string
	BytesSent int
	MetricsSent int
	MessagesSent int
	LastInventoryRun time.Time
	LastMetricsRun time.Time
	ExecutionTimeInventory time.Duration
	ExecutionTimeMetrics time.Duration
	UpSince time.Duration
	LastError string
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

//---------------------------------------------

func Start(c *gin.Context) {
	var controllerServices = CreateController()
	_, _ = controllerServices.Start()

	c.JSON(http.StatusOK, controllerServices)
}

func Stop(c *gin.Context) {
	var controllerServices = CreateController()
	_, _ = controllerServices.Stop()

	c.JSON(http.StatusOK, controllerServices)
}

func Status(c *gin.Context) {
	var controllerServices = CreateController()

	c.JSON(http.StatusOK, controllerServices)
}

func Stats(c *gin.Context) {
	var controllerServices = CreateController()
	stats, _ := controllerServices.Stats()

	c.JSON(http.StatusOK, stats)
}

//---------------------------------------------

type Controller struct {
	State StatusEnum
}

func CreateController() *Controller {
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

func (controller *Controller) Stats() (*AgentStats, error) {
	return &AgentStats{
		AgentId:                "agent 007",
		AppType:				"nagios",
		BytesSent:              8192,
		MetricsSent:            1024,
		MessagesSent:           512,
		LastInventoryRun:       time.Time{},
		LastMetricsRun:         time.Time{},
		ExecutionTimeInventory: 3949,
		ExecutionTimeMetrics:   21934,
		UpSince:                9393993,
		LastError:              "",
	}, nil
}
