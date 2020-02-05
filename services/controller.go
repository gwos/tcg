package services

import (
	"context"
	"fmt"
	"github.com/gin-gonic/contrib/cors"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gwos/tng/cache"
	"github.com/gwos/tng/log"
	"github.com/gwos/tng/nats"
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

// SendEvent implements Controllers.SendEvent interface
func (controller *Controller) SendEvent(bytes []byte) error {
	return nats.Publish(SubjSendEvent, bytes)
}

// starts the http server
// overrides AgentService implementation
func (controller *Controller) startController() error {
	if controller.srv != nil {
		return fmt.Errorf("StartController: already started")
	}

	var addr string
	if strings.HasPrefix(controller.Connector.ControllerAddr, ":") {
		addr = "localhost" + controller.Connector.ControllerAddr
	} else {
		addr = controller.Connector.ControllerAddr
	}
	certFile := controller.Connector.ControllerCertFile
	keyFile := controller.Connector.ControllerKeyFile

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
		controller.agentStatus.Controller = Running

		var err error
		if certFile != "" && keyFile != "" {
			log.Info("Controller: start listen TLS", addr)
			if err = controller.srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
				log.Error("Controller: start error:", err)
			}
		} else {
			log.Info("Controller: start listen", addr)
			if err = controller.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Error("Controller: start error:", err)
			}
		}

		controller.agentStatus.Controller = Stopped
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

// gracefully shutdowns the http server
// overrides AgentService implementation
func (controller *Controller) stopController() error {
	// NOTE: the controller.agentStatus.Controller will be updated by controller.StartController itself
	log.Info("Controller: shutdown ...")
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := controller.srv.Shutdown(ctx); err != nil {
		log.Error("Controller: shutdown error:", err)
	}
	// catching ctx.Done() timeout
	select {
	case <-ctx.Done():
		log.Warn("Controller: shutdown: timeout")
	}
	log.Warn("Controller: exiting")
	controller.srv = nil
	return nil
}

//
// @Description The following API endpoint can be used to send Alerts to Foundation.
// @Tags Alerts
// @Accept  json
// @Produce  json
// @Success 200
// @Failure 401 {string} string "Unauthorized"
// @Router /events [post]
// @Param   GWOS-APP-NAME    header    string     true        "Auth header"
// @Param   GWOS-API-TOKEN   header    string     true        "Auth header"
func (controller *Controller) events(c *gin.Context) {
	value, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	err = controller.SendEvent(value)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, nil)
}

//
// @Description The following API endpoint can be used to get list of metrics from the server.
// @Tags Metrics
// @Accept  json
// @Produce  json
// @Success 200 {object} services.AgentStatus
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal server error"
// @Router /metrics [get]
// @Param   GWOS-APP-NAME    header    string     true        "Auth header"
// @Param   GWOS-API-TOKEN   header    string     true        "Auth header"
func (controller *Controller) listMetrics(c *gin.Context) {
	metrics, err := controller.ListMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.Data(http.StatusOK, gin.MIMEJSON, metrics)
}

//
// @Description The following API endpoint can be used to reload the configuration.
// @Tags Server
// @Accept  json
// @Produce  json
// @Success 200
// @Failure 401 {string} string "Unauthorized"
// @Router /reload [post]
// @Param   GWOS-APP-NAME    header    string     true        "Auth header"
// @Param   GWOS-API-TOKEN   header    string     true        "Auth header"
func (controller *Controller) reload(c *gin.Context) {
	ctrl, err := controller.ReloadAsync(nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, ConnectorStatusDTO{Processing, ctrl.Idx})
}

//
// @Description The following API endpoint can be used to start NATS dispatcher.
// @Tags Server
// @Accept  json
// @Produce  json
// @Success 200 {object} services.ConnectorStatusDTO
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal server error"
// @Router /start [post]
// @Param   GWOS-APP-NAME    header    string     true        "Auth header"
// @Param   GWOS-API-TOKEN   header    string     true        "Auth header"
func (controller *Controller) start(c *gin.Context) {
	status := controller.Status()
	if status.Transport == Running && status.Ctrl == nil {
		c.JSON(http.StatusOK, ConnectorStatusDTO{Running, 0})
		return
	}
	ctrl, err := controller.StartTransportAsync(nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, ConnectorStatusDTO{Processing, ctrl.Idx})
}

//
// @Description The following API endpoint can be used to stop NATS dispatcher.
// @Tags Server
// @Accept  json
// @Produce  json
// @Success 200 {object} services.ConnectorStatusDTO
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal server error"
// @Router /stop [post]
// @Param   GWOS-APP-NAME    header    string     true        "Auth header"
// @Param   GWOS-API-TOKEN   header    string     true        "Auth header"
func (controller *Controller) stop(c *gin.Context) {
	status := controller.Status()
	if status.Transport == Stopped && status.Ctrl == nil {
		c.JSON(http.StatusOK, ConnectorStatusDTO{Stopped, 0})
		return
	}
	ctrl, err := controller.StopTransportAsync(nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, ConnectorStatusDTO{Processing, ctrl.Idx})
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
// @Param   GWOS-API-TOKEN   header    string     true        "Auth header"
func (controller *Controller) stats(c *gin.Context) {
	c.JSON(http.StatusOK, controller.Stats())
}

//
// @Description The following API endpoint can be used to get TNG status.
// @Tags Server
// @Accept  json
// @Produce  json
// @Success 200 {object} services.ConnectorStatusDTO
// @Failure 401 {string} string "Unauthorized"
// @Router /status [get]
// @Param   GWOS-APP-NAME    header    string     true        "Auth header"
// @Param   GWOS-API-TOKEN   header    string     true        "Auth header"
func (controller *Controller) status(c *gin.Context) {
	status := controller.Status()
	statusDTO := ConnectorStatusDTO{status.Transport, 0}
	if status.Ctrl != nil {
		statusDTO = ConnectorStatusDTO{Processing, status.Ctrl.Idx}
	}
	c.JSON(http.StatusOK, statusDTO)
}

func (controller *Controller) validateToken(c *gin.Context) {
	// check local pin
	pin := controller.Connector.ControllerPin
	if len(pin) > 0 && pin == c.Request.Header.Get("X-PIN") {
		return
	}

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
		err := controller.dsClient.ValidateToken(credentials.GwosAppName, credentials.GwosAPIToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		err = cache.AuthCache.Add(key, credentials, 8*time.Hour)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
}

func (controller *Controller) registerAPI1(router *gin.Engine, addr string) {
	swaggerURL := ginSwagger.URL("http://" + addr + "/swagger/doc.json")
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, swaggerURL))

	apiV1Group := router.Group("/api/v1")
	apiV1Group.Use(controller.validateToken)

	apiV1Group.POST("/events", controller.events)
	apiV1Group.GET("/metrics", controller.listMetrics)
	apiV1Group.POST("/reload", controller.reload)
	apiV1Group.POST("/start", controller.start)
	apiV1Group.POST("/stop", controller.stop)
	apiV1Group.GET("/stats", controller.stats)
	apiV1Group.GET("/status", controller.status)

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
