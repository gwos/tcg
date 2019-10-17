package controller

import (
	"fmt"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

const userKey string = "user"

var TransitController = NewController()

func StartServer(tls bool, port int) error {
	router := gin.Default()

	router.Use(sessions.Sessions("mysession", sessions.NewCookieStore([]byte("secret"))))

	router.POST("api/v1/login", login)

	basicAuth := router.Group("/api/v1")
	basicAuth.Use(authenticationRequired)
	{
		basicAuth.GET("/stats", stats)
		basicAuth.GET("/status", status)
		basicAuth.POST("/nats/start", startNATS)
		basicAuth.DELETE("/nats/stop", stopNATS)
		basicAuth.POST("/nats/transport/start", startTransport)
		basicAuth.DELETE("/nats/transport/stop", stopTransport)
	}

	if tls {
		if err := router.RunTLS(fmt.Sprintf(":%d", port), "controller/server.pem", "controller/server.key"); err != nil {
			return err
		}
	} else {
		if err := router.Run(fmt.Sprintf(":%d", port)); err != nil {
			return err
		}
	}

	return nil
}

func startNATS(c *gin.Context) {
	err := TransitController.StartNATS()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
	}

	c.JSON(http.StatusOK, TransitController)
}

func stopNATS(c *gin.Context) {
	err := TransitController.StopNATS()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
	}

	c.JSON(http.StatusOK, TransitController)
}

func startTransport(c *gin.Context) {
	err := TransitController.StartTransport()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
	}

	c.JSON(http.StatusOK, TransitController)
}

func stopTransport(c *gin.Context) {
	err := TransitController.StopTransport()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
	}

	c.JSON(http.StatusOK, TransitController)
}

func status(c *gin.Context) {
	c.JSON(http.StatusOK, TransitController)
}

func stats(c *gin.Context) {
	stats, err := TransitController.Stats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
	}
	c.JSON(http.StatusOK, stats)
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
