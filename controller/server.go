package controller

import (
	"context"
	"fmt"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gwos/tng/cache"
	"log"
	"net/http"
	"net/http/pprof"
	"time"
)

const shutdownTimeout = 5 * time.Second

var controller = NewController()
var srv *http.Server

// StopServer gracefully shutdowns the server
func StopServer() error {
	log.Println("controller: shutdown ...")
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Println("controller: shutdown error:", err)
	}
	// catching ctx.Done() timeout
	select {
	case <-ctx.Done():
		log.Println("controller: shutdown: timeout")
	}
	log.Println("controller: exiting")
	srv = nil
	return nil
}

// StartServer starts http server
func StartServer(addr, certFile, keyFile string) error {
	if srv != nil {
		return fmt.Errorf("controller: already started")
	}
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(sessions.Sessions("tng-session", sessions.NewCookieStore([]byte("secret"))))
	registerAPI1(router)

	srv = &http.Server{
		Addr:    addr,
		Handler: router,
	}
	go func() {
		if certFile != "" && keyFile != "" {
			log.Println("controller: start listen TLS", addr)
			if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
				log.Println("controller: start error:", err)
			}
		} else {
			log.Println("controller: start listen", addr)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Println("controller: start error:", err)
			}
		}
	}()
	// TODO: ensure signal processing in case of linked library
	// // Wait for interrupt signal to gracefully shutdown the server
	// quit := make(chan os.Signal)
	// // kill (no param) default send syscall.SIGTERM
	// // kill -2 is syscall.SIGINT
	// // kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
	// signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	// <-quit
	// StopServer()

	return nil
}

func startNATS(c *gin.Context) {
	err := controller.StartNATS()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
	}

	c.JSON(http.StatusOK, controller)
}

func stopNATS(c *gin.Context) {
	err := controller.StopNATS()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
	}

	c.JSON(http.StatusOK, controller)
}

func startTransport(c *gin.Context) {
	err := controller.StartTransport()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
	}

	c.JSON(http.StatusOK, controller)
}

func stopTransport(c *gin.Context) {
	err := controller.StopTransport()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
	}

	c.JSON(http.StatusOK, controller)
}

func status(c *gin.Context) {
	c.JSON(http.StatusOK, controller)
}

func stats(c *gin.Context) {
	stats, err := controller.Stats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
	}
	c.JSON(http.StatusOK, stats)
}

func listMetrics(c *gin.Context) {
	metrics, err := controller.ListMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
	}

	c.JSON(http.StatusOK, string(metrics))
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
		err := controller.ValidateToken(credentials.GwosAppName, credentials.GwosApiToken)
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

func registerAPI1(router *gin.Engine) {
	apiV1Group := router.Group("/api/v1")
	apiV1Group.Use(authorizationValidation)

	apiV1Group.GET("/stats", stats)
	apiV1Group.GET("/status", status)
	router.GET("/listMetrics", listMetrics)
	apiV1Group.POST("/nats/start", startNATS)
	apiV1Group.DELETE("/nats/stop", stopNATS)
	apiV1Group.POST("/nats/transport/start", startTransport)
	apiV1Group.DELETE("/nats/transport/stop", stopTransport)

	pprofGroup := apiV1Group.Group("/pprof")
	pprofGroup.GET("/", gin.WrapF(pprof.Index))
	pprofGroup.GET("/cmdline", gin.WrapF(pprof.Cmdline))
	pprofGroup.GET("/profile", gin.WrapF(pprof.Profile))
	pprofGroup.GET("/symbol", gin.WrapF(pprof.Symbol))
	pprofGroup.POST("/symbol", gin.WrapF(pprof.Symbol))
	pprofGroup.GET("/trace", gin.WrapF(pprof.Trace))
	pprofGroup.GET("/allocs", gin.WrapF(pprof.Handler("allocs").ServeHTTP))
	pprofGroup.GET("/block", gin.WrapF(pprof.Handler("block").ServeHTTP))
	pprofGroup.GET("/goroutine", gin.WrapF(pprof.Handler("goroutine").ServeHTTP))
	pprofGroup.GET("/heap", gin.WrapF(pprof.Handler("heap").ServeHTTP))
	pprofGroup.GET("/mutex", gin.WrapF(pprof.Handler("mutex").ServeHTTP))
	pprofGroup.GET("/threadcreate", gin.WrapF(pprof.Handler("threadcreate").ServeHTTP))
}
