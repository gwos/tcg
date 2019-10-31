package services

import (
	"context"
	"fmt"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gwos/tng/cache"
	"github.com/gwos/tng/nats"
	stan "github.com/nats-io/go-nats-streaming"
	"log"
	"net/http"
	"net/http/pprof"
	"sync"
	"time"
)

// Controller implements Controllers interface
type Controller struct {
	*TransitService
	srv *http.Server
}

const shutdownTimeout = 5 * time.Second

var onceController sync.Once
var controller *Controller

// GetController implements Singleton pattern
func GetController() *Controller {
	onceController.Do(func() {
		controller = &Controller{GetTransitService(), nil}
	})
	return controller
}

// ListMetrics implements Controllers.ListMetrics interface
func (controller *Controller) ListMetrics() ([]byte, error) {
	ch := make(chan []byte)
	defer close(ch)

	go func(c chan []byte) {
		done := make(chan bool)
		defer close(done)
		natsConn, _ := nats.Connect("tng-controller")
		natsSub, _ := natsConn.Subscribe(SubjListMetricsResponse, func(msg *stan.Msg) {
			c <- msg.Data
			done <- true
		})
		<-done
		natsSub.Close()
		natsConn.Close()
	}(ch)

	err := nats.Publish(SubjListMetricsRequest, []byte("REQUEST"))
	if err != nil {
		return nil, err
	}

	return <-ch, nil
}

// StartServer starts the http server
func (controller *Controller) StartServer(addr, certFile, keyFile string) error {
	if controller.srv != nil {
		return fmt.Errorf("controller: already started")
	}
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(sessions.Sessions("tng-session", sessions.NewCookieStore([]byte("secret"))))
	controller.registerAPI1(router)

	controller.srv = &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		controller.agentStatus.Lock()
		controller.agentStatus.Controller = Running
		controller.agentStatus.Unlock()

		var err error
		if certFile != "" && keyFile != "" {
			log.Println("controller: start listen TLS", addr)
			if err = controller.srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
				log.Println("controller: start error:", err)
			}
		} else {
			log.Println("controller: start listen", addr)
			if err = controller.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Println("controller: start error:", err)
			}
		}

		controller.agentStatus.Lock()
		controller.agentStatus.Controller = Stopped
		controller.agentStatus.Unlock()
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

// StopServer gracefully shutdowns the http server
func (controller *Controller) StopServer() error {
	// NOTE: the controller.agentStatus.Controller will be updated by controller.StartServer itself
	log.Println("controller: shutdown ...")
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := controller.srv.Shutdown(ctx); err != nil {
		log.Println("controller: shutdown error:", err)
	}
	// catching ctx.Done() timeout
	select {
	case <-ctx.Done():
		log.Println("controller: shutdown: timeout")
	}
	log.Println("controller: exiting")
	controller.srv = nil
	return nil
}

func (controller *Controller) listMetrics(c *gin.Context) {
	metrics, err := controller.ListMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
	}

	c.JSON(http.StatusOK, string(metrics))
}

func (controller *Controller) startNATS(c *gin.Context) {
	err := controller.StartNATS()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
	}

	c.JSON(http.StatusOK, controller)
}

func (controller *Controller) stopNATS(c *gin.Context) {
	err := controller.StopNATS()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
	}

	c.JSON(http.StatusOK, controller)
}

func (controller *Controller) startTransport(c *gin.Context) {
	err := controller.StartTransport()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
	}

	c.JSON(http.StatusOK, controller)
}

func (controller *Controller) stopTransport(c *gin.Context) {
	err := controller.StopTransport()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
	}

	c.JSON(http.StatusOK, controller)
}

func (controller *Controller) stats(c *gin.Context) {
	c.JSON(http.StatusOK, controller.Stats())
}

func (controller *Controller) status(c *gin.Context) {
	c.JSON(http.StatusOK, controller.Status())
}

func (controller *Controller) validateToken(c *gin.Context) {
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
		err := controller.Transit.ValidateToken(credentials.GwosAppName, credentials.GwosApiToken)
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

func (controller *Controller) registerAPI1(router *gin.Engine) {
	apiV1Group := router.Group("/api/v1")
	apiV1Group.Use(controller.validateToken)

	apiV1Group.GET("/listMetrics", controller.listMetrics)
	apiV1Group.GET("/stats", controller.stats)
	apiV1Group.GET("/status", controller.status)
	apiV1Group.POST("/nats/start", controller.startNATS)
	apiV1Group.DELETE("/nats/stop", controller.stopNATS)
	apiV1Group.POST("/nats/transport/start", controller.startTransport)
	apiV1Group.DELETE("/nats/transport/stop", controller.stopTransport)

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