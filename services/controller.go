package services

import (
	"context"
	"fmt"
	"github.com/gin-gonic/contrib/cors"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gwos/tng/cache"
	"github.com/gwos/tng/clients"
	"github.com/gwos/tng/setup"
	"github.com/gwos/tng/log"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"
	"net/http"
	"net/http/pprof"
	"strings"
	"sync"
	"time"
)

// Controller implements AgentServices, Controllers interface
type Controller struct {
	*AgentService
	srv                *http.Server
	listMetricsHandler GetBytesHandlerType
	authClient         *clients.GWClient
}

const shutdownTimeout = 5 * time.Second

var onceController sync.Once
var controller *Controller

// GetController implements Singleton pattern
func GetController() *Controller {
	onceController.Do(func() {
		controller = &Controller{
			GetAgentService(),
			nil,
			nil,
			nil,
		}
	})
	return controller
}

// ListMetrics implements Controllers.ListMetrics interface
func (controller *Controller) ListMetrics() ([]byte, error) {
	if controller.listMetricsHandler != nil {
		return controller.listMetricsHandler()
	}
	return nil, fmt.Errorf("listMetricsHandler unavailable")
}

// RegisterListMetricsHandler implements Controllers.RegisterListMetricsHandler interface
func (controller *Controller) RegisterListMetricsHandler(fn GetBytesHandlerType) {
	controller.listMetricsHandler = fn
}

// RemoveListMetricsHandler implements Controllers.RemoveListMetricsHandler interface
func (controller *Controller) RemoveListMetricsHandler() {
	controller.listMetricsHandler = nil
}

// StartController implements AgentServices.StartController interface
// overrides AgentService implementation
// starts the http server
func (controller *Controller) StartController() error {
	if controller.srv != nil {
		return fmt.Errorf("StartController: already started")
	}
	if len(setup.GetConfig().GWConfigs) == 0 {
		return fmt.Errorf("StartController: %v", "empty GWConfigs")
	}

	controller.authClient = &clients.GWClient{GWConfig: setup.GetConfig().GWConfigs[0]}
	var addr string
	if strings.HasPrefix(controller.AgentConfig.ControllerAddr, ":") {
		addr = "localhost" + controller.AgentConfig.ControllerAddr
	} else {
		addr = controller.AgentConfig.ControllerAddr
	}
	certFile := controller.AgentConfig.ControllerCertFile
	keyFile := controller.AgentConfig.ControllerKeyFile

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowedHeaders = []string{"GWOS-APP-NAME", "GWOS-API-TOKEN", "Content-Type"}
	router.Use(cors.New(corsConfig))
	router.Use(sessions.Sessions("tng-session", sessions.NewCookieStore([]byte("secret"))))
	controller.registerAPI1(router, addr)

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
			log.Info("controller: start listen TLS", addr)
			if err = controller.srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
				log.Error("controller: start error:", err)
			}
		} else {
			log.Info("controller: start listen", addr)
			if err = controller.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Error("controller: start error:", err)
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

// StopController implements AgentServices.StopController interface
// overrides AgentService implementation
// gracefully shutdowns the http server
func (controller *Controller) StopController() error {
	// NOTE: the controller.agentStatus.Controller will be updated by controller.StartServer itself
	log.Info("Controller: shutdown ...")
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := controller.srv.Shutdown(ctx); err != nil {
		log.Error("Controller: shutdown error:", err)
	}
	// catching ctx.Done() timeout
	select {
	case <-ctx.Done():
		log.Warn("controller: shutdown: timeout")
	}
	log.Warn("controller: exiting")
	controller.srv = nil
	return nil
}

//
// @Description The following API endpoint can be used to get list of metrics from the server.
// @Tags Metrics
// @Accept  json
// @Produce  json
// @Success 200 {object} services.AgentStatus
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal server error"
// @Router /listMetrics [get]
// @Param   GWOS-APP-NAME    header    string     true        "Auth header"
// @Param   GWOS-API-TOKEN    header    string     true        "Auth header"
func (controller *Controller) listMetrics(c *gin.Context) {
	metrics, err := controller.ListMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
	}

	c.Data(http.StatusOK, gin.MIMEJSON, metrics)
}

//
// @Description The following API endpoint can be used to start NATS streaming server.
// @Tags NATS
// @Accept  json
// @Produce  json
// @Success 200 {object} services.AgentStatus
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal server error"
// @Router /nats [post]
// @Param   GWOS-APP-NAME    header    string     true        "Auth header"
// @Param   GWOS-API-TOKEN    header    string     true        "Auth header"
func (controller *Controller) startNats(c *gin.Context) {
	err := controller.StartNats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
	}

	c.JSON(http.StatusOK, controller.Status())
}

//
// @Description The following API endpoint can be used to stop NATS streaming server.
// @Tags NATS
// @Accept  json
// @Produce  json
// @Success 200 {object} services.AgentStatus
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal server error"
// @Router /nats [delete]
// @Param   GWOS-APP-NAME    header    string     true        "Auth header"
// @Param   GWOS-API-TOKEN    header    string     true        "Auth header"
func (controller *Controller) stopNats(c *gin.Context) {
	err := controller.StopNats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
	}

	c.JSON(http.StatusOK, controller.Status())
}

//
// @Description The following API endpoint can be used to start NATS transport(this means that messages will begin to be sent).
// @Tags NATS
// @Accept  json
// @Produce  json
// @Success 200 {object} services.AgentStatus
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal server error"
// @Router /nats/transport [post]
// @Param   GWOS-APP-NAME    header    string     true        "Auth header"
// @Param   GWOS-API-TOKEN    header    string     true        "Auth header"
func (controller *Controller) startTransport(c *gin.Context) {
	err := controller.StartTransport()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
	}

	c.JSON(http.StatusOK, controller.Status())
}

//
// @Description The following API endpoint can be used to stop NATS transport.
// @Tags NATS
// @Accept  json
// @Produce  json
// @Success 200 {object} services.AgentStatus
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal server error"
// @Router /nats/transport [delete]
// @Param   GWOS-APP-NAME    header    string     true        "Auth header"
// @Param   GWOS-API-TOKEN    header    string     true        "Auth header"
func (controller *Controller) stopTransport(c *gin.Context) {
	err := controller.StopTransport()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
	}

	c.JSON(http.StatusOK, controller.Status())
}

//
// @Description The following API endpoint can be used to get TNG statistics.
// @Tags Agent
// @Accept  json
// @Produce  json
// @Success 200 {object} services.AgentStats
// @Failure 401 {string} string "Unauthorized"
// @Router /stats [get]
// @Param   gwos-app-name    header    string     true        "Auth header"
// @Param   gwos-api-token    header    string     true        "Auth header"
func (controller *Controller) stats(c *gin.Context) {
	c.JSON(http.StatusOK, controller.Stats())
}

//
// @Description The following API endpoint can be used to get TNG status.
// @Tags Server
// @Accept  json
// @Produce  json
// @Success 200 {object} services.AgentStatus
// @Failure 401 {string} string "Unauthorized"
// @Router /status [get]
// @Param   GWOS-APP-NAME    header    string     true        "Auth header"
// @Param   GWOS-API-TOKEN    header    string     true        "Auth header"
func (controller *Controller) status(c *gin.Context) {
	c.JSON(http.StatusOK, controller.Status())
}

func (controller *Controller) validateToken(c *gin.Context) {
	credentials := cache.Credentials{
		GwosAppName:  c.Request.Header.Get("GWOS-APP-NAME"),
		GwosAPIToken: c.Request.Header.Get("GWOS-API-TOKEN"),
	}

	if credentials.GwosAppName == "" || credentials.GwosAPIToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": `Invalid GWOS-APP-NAME or GWOS-API-TOKEN`})
		c.Abort()
		return
	}

	key := fmt.Sprintf("%s:%s", credentials.GwosAppName, credentials.GwosAPIToken)

	_, isCached := cache.AuthCache.Get(key)
	if !isCached {
		err := controller.authClient.ValidateToken(credentials.GwosAppName, credentials.GwosAPIToken)
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

func (controller *Controller) registerAPI1(router *gin.Engine, addr string) {
	swaggerUrl := ginSwagger.URL("http://" + addr + "/swagger/doc.json")
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, swaggerUrl))

	apiV1Group := router.Group("/api/v1")
	apiV1Group.Use(controller.validateToken)

	apiV1Group.GET("/listMetrics", controller.listMetrics)
	apiV1Group.GET("/stats", controller.stats)
	apiV1Group.GET("/status", controller.status)
	apiV1Group.POST("/nats", controller.startNats)
	apiV1Group.DELETE("/nats", controller.stopNats)
	apiV1Group.POST("/nats/transport", controller.startTransport)
	apiV1Group.DELETE("/nats/transport", controller.stopTransport)

	pprofGroup := apiV1Group.Group("/debug/pprof")
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
