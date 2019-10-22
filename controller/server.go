package controller

import (
	"fmt"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gwos/tng/cache"
	"net/http"
	"time"
)

const userKey string = "user"

var TransitController = NewController()

func StartServer(tls bool, port int) error {
	router := gin.Default()

	router.Use(sessions.Sessions("mysession", sessions.NewCookieStore([]byte("secret"))))

	basicAuth := router.Group("/api/v1")

	basicAuth.Use(authorizationValidation)
	{
		basicAuth.GET("/stats", stats)
		basicAuth.GET("/status", status)
		basicAuth.POST("/nats/start", startNATS)
		basicAuth.DELETE("/nats/stop", stopNATS)
		basicAuth.POST("/nats/transport/start", startTransport)
		basicAuth.DELETE("/nats/transport/stop", stopTransport)

		basicAuth.GET("/test", test)
	}

	if tls {
		if err := router.RunTLS(fmt.Sprintf(":%d", port), "../controller/server.pem", "../controller/server.key"); err != nil {
			return err
		}
	} else {
		if err := router.Run(fmt.Sprintf(":%d", port)); err != nil {
			return err
		}
	}

	return nil
}

func test(context *gin.Context) {
	context.JSON(http.StatusOK, "WORKS!")
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

func authorizationValidation(c *gin.Context) {
	credentials := cache.Credentials{
		GwosAppName:  c.Request.Header.Get("GWOS-APP-NAME"),
		GwosApiToken: c.Request.Header.Get("GWOS-API-TOKEN"),
	}

	if credentials.GwosAppName == "" || credentials.GwosApiToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid \"GWOS-APP-NAME\" or \"GWOS-API-TOKEN\""})
		c.Abort()
		return
	}

	key := fmt.Sprintf("%s:%s", credentials.GwosAppName, credentials.GwosApiToken)

	_, isCached := cache.AuthCache.Get(key)
	if !isCached {
		err := TransitController.Identity(credentials.GwosAppName, credentials.GwosApiToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		err = cache.AuthCache.Add(key, credentials, 8*time.Hour)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			c.Abort()
			return
		}
	}
}
