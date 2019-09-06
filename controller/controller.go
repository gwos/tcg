package controller

import (
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"time"
)

// Agent possible status
type StatusEnum int
const (
	Running StatusEnum = iota
	Stopped
	Unknown
	Pending
	userKey string = "user"
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
	start() (StatusEnum, error)
	stop() (StatusEnum, error)
	status() (StatusEnum, error)
	stats() (*AgentStats, error)
	// LoadConfig() (StatusEnum, error)  // TODO: define configs to be passed in
	// ListConfig() (StatusEnum, error)  // TODO: define configs to be returned
}

var controllerServices = CreateController()

func StartServer(){
	router := gin.Default()

	router.Use(sessions.Sessions("mysession", sessions.NewCookieStore([]byte("secret"))))

	router.POST("api/v1/login", login)

	basicAuth := router.Group("/api/v1")
	basicAuth.Use(authenticationRequired)
	{
		basicAuth.GET("/stats", stats)
		basicAuth.GET("/status", status)
		basicAuth.POST("/start", start)
		basicAuth.DELETE("/stop", stop)
	}

	_ = router.RunTLS(":8080", "controller/server.pem", "controller/server.key")
}

func authenticationRequired(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user needs to be signed in to access this service"})
		c.Abort()
		return
	}
}

func login(c *gin.Context) {
	session := sessions.Default(c)
	username := c.PostForm("username")
	password := c.PostForm("password")

	// Validate form input
	if strings.Trim(username, " ") == "" || strings.Trim(password, " ") == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parameters can't be empty"})
		return
	}

	if username != "vlad" || password != "gwos" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
		return
	}

	session.Set(userKey, username) // In real world usage you'd set this to the users ID
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully authenticated user"})
}


func start(c *gin.Context) {
	_, _ = controllerServices.Start()

	c.JSON(http.StatusOK, controllerServices)
}

func stop(c *gin.Context) {
	_, _ = controllerServices.Stop()

	c.JSON(http.StatusOK, controllerServices)
}

func status(c *gin.Context) {
	c.JSON(http.StatusOK, controllerServices)
}

func stats(c *gin.Context) {
	stats, _ := controllerServices.Stats()

	c.JSON(http.StatusOK, stats)
}

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
