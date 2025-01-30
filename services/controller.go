package services

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"expvar"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/pprof"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/gwos/tcg/config"
	tcgerr "github.com/gwos/tcg/sdk/errors"
	"github.com/gwos/tcg/tracing"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/sys/unix"
)

// Controller implements AgentServices, Controllers interface
type Controller struct {
	*TransitService
	muBASIC     sync.Mutex
	muGWOS      sync.Mutex
	authCache   *cache.Cache
	entrypoints []Entrypoint
	srv         *http.Server
}

// Credentials defines type of AuthCache items
type Credentials struct {
	GwosAppName  string
	GwosAPIToken string
}

// Entrypoint describes controller API
type Entrypoint struct {
	URL     string
	Method  string
	Handler func(c *gin.Context)
}

const startRetryDelay = time.Millisecond * 200

var onceController sync.Once
var controller *Controller

// GetController implements Singleton pattern
func GetController() *Controller {
	onceController.Do(func() {
		controller = &Controller{
			TransitService: GetTransitService(),
			authCache:      cache.New(8*time.Hour, time.Hour),
			entrypoints:    []Entrypoint{},
		}
	})
	return controller
}

// RegisterEntrypoints sets addition API
func (controller *Controller) RegisterEntrypoints(entrypoints []Entrypoint) {
	controller.entrypoints = entrypoints
}

// RemoveEntrypoints removes addition API
func (controller *Controller) RemoveEntrypoints() {
	controller.entrypoints = []Entrypoint{}
}

// starts the http server
// overrides AgentService implementation
func (controller *Controller) startController() error {
	if controller.srv != nil {
		log.Warn().Msg("controller already started")
		return nil
	}

	certFile := controller.Connector.ControllerCertFile
	keyFile := controller.Connector.ControllerKeyFile
	addr := controller.Connector.ControllerAddr
	if strings.HasPrefix(addr, ":") {
		addr = "0.0.0.0" + addr
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowHeaders = []string{"GWOS-APP-NAME", "GWOS-API-TOKEN", "Content-Type"}
	router.Use(cors.New(corsConfig))
	router.Use(sessions.Sessions("tcg-session", cookie.NewStore([]byte("secret"))))
	controller.registerAPI1(router, addr, controller.entrypoints)

	/* set a short timer to wait for http.Server starting */
	idleTimer := time.NewTimer(startRetryDelay * 2)
	go func() {
		t0 := time.Now()
		controller.agentStatus.Controller.Set(StatusRunning)
		controller.srv = &http.Server{
			Addr:         addr,
			Handler:      router,
			ReadTimeout:  controller.Connector.ControllerReadTimeout,
			WriteTimeout: controller.Connector.ControllerWriteTimeout,
		}
		lc := net.ListenConfig{
			Control: func(network, address string, c syscall.RawConn) error {
				var opErr error
				if err := c.Control(func(fd uintptr) {
					opErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
				}); err != nil {
					return err
				}
				return opErr
			},
		}

		for {
			var err error
			var l net.Listener
			if certFile != "" && keyFile != "" {
				log.Info().Msgf("controller starts listen TLS: %s", addr)
				// err = controller.srv.ListenAndServeTLS(certFile, keyFile)
				// using listener with configured socket options SO_REUSEPORT to prevent
				// "bind: address already in use" error on stop-start controller
				if l, err = lc.Listen(context.Background(), "tcp", addr); err == nil {
					err = controller.srv.ServeTLS(l, certFile, keyFile)
				}
			} else {
				log.Info().Msgf("controller starts listen: %s", addr)
				// err = controller.srv.ListenAndServe()
				// using listener with configured socket options SO_REUSEPORT to prevent
				// "bind: address already in use" error on stop-start controller
				if l, err = lc.Listen(context.Background(), "tcp", addr); err == nil {
					err = controller.srv.Serve(l)
				}
			}
			/* getting here after http.Server exit */
			/* catch the "bind: address already in use" error */
			if err != nil && tcgerr.IsErrorAddressInUse(err) &&
				time.Since(t0) < controller.Connector.ControllerStartTimeout-startRetryDelay {
				log.Warn().Err(err).Msg("controller retrying http.Server start")
				time.Sleep(startRetryDelay)
				idleTimer.Reset(startRetryDelay * 2)
				continue
			} else if err != nil && err != http.ErrServerClosed {
				log.Err(err).Msg("controller got http.Server error")
			}
			break
		}
		controller.agentStatus.Controller.Set(StatusStopped)
		controller.srv = nil
	}()
	/* wait for http.Server starting to prevent misbehavior on immediate shutdown */
	<-idleTimer.C
	return nil
}

// gracefully shutdowns the http server
// overrides AgentService implementation
func (controller *Controller) stopController() error {
	// NOTE: the controller.agentStatus.Controller will be updated by controller.StartController itself
	if controller.srv == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), controller.Connector.ControllerStopTimeout)
	go func() {
		log.Info().Msg("controller shutdown ...")
		if err := controller.srv.Shutdown(ctx); err != nil {
			log.Warn().Msgf("controller got %s", err)
		}
		cancel()
	}()
	/* wait for http.Server stopping to prevent misbehavior on immediate start */
	<-ctx.Done()
	return nil
}

// @Description The following API endpoint can be used to Agent configure.
// @Tags    agent, connector
// @Accept  json
// @Produce json
// @Success 200
// @Failure 401 {string} string "Unauthorized"
// @Router  /config [post]
// @Param   GWOS-APP-NAME    header    string     true        "Auth header"
// @Param   GWOS-API-TOKEN   header    string     true        "Auth header"
func (controller *Controller) config(c *gin.Context) {
	payload, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	/* validate payload */
	var dto config.ConnectorDTO
	if err := json.Unmarshal(payload, &dto); err != nil {
		log.Err(err).Msg("could not unmarshal connector dto")
		c.JSON(http.StatusBadRequest, "could not unmarshal connector dto")
		return
	}
	/* process payload */
	task, err := controller.taskQueue.PushAsync(taskConfig, payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, ConnectorStatusDTO{StatusProcessing, task.Idx})
}

// @Description The following API endpoint can be used to send Downtimes to Foundation.
// @Tags    downtimes
// @Accept  json
// @Produce json
// @Success 200
// @Failure 401 {string} string "Unauthorized"
// @Router  /downtime-clear [post]
// @Param   GWOS-APP-NAME    header    string     true        "Auth header"
// @Param   GWOS-API-TOKEN   header    string     true        "Auth header"
func (controller *Controller) clearInDowntime(c *gin.Context) {
	var (
		err     error
		payload []byte
	)
	ctx, span := tracing.StartTraceSpan(c.Request.Context(), "controller", string(TOpClearInDowntime))
	defer func() {
		tracing.EndTraceSpan(span,
			tracing.TraceAttrError(err),
			tracing.TraceAttrPayloadLen(payload),
			tracing.TraceAttrEntrypoint(c.FullPath()),
		)
	}()

	payload, err = c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	err = controller.ClearInDowntime(ctx, payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, nil)
}

// @Description The following API endpoint can be used to send Downtimes to Foundation.
// @Tags    downtimes
// @Accept  json
// @Produce json
// @Success 200
// @Failure 401 {string} string "Unauthorized"
// @Router  /downtime-set [post]
// @Param   GWOS-APP-NAME    header    string     true        "Auth header"
// @Param   GWOS-API-TOKEN   header    string     true        "Auth header"
func (controller *Controller) setInDowntime(c *gin.Context) {
	var (
		err     error
		payload []byte
	)
	ctx, span := tracing.StartTraceSpan(c.Request.Context(), "controller", string(TOpSetInDowntime))
	defer func() {
		tracing.EndTraceSpan(span,
			tracing.TraceAttrError(err),
			tracing.TraceAttrPayloadLen(payload),
			tracing.TraceAttrEntrypoint(c.FullPath()),
		)
	}()

	payload, err = c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	err = controller.SetInDowntime(ctx, payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, nil)
}

// @Description The following API endpoint can be used to send Alerts to Foundation.
// @Tags    alert, event
// @Accept  json
// @Produce json
// @Success 200
// @Failure 401 {string} string "Unauthorized"
// @Router  /events [post]
// @Param   GWOS-APP-NAME    header    string     true        "Auth header"
// @Param   GWOS-API-TOKEN   header    string     true        "Auth header"
func (controller *Controller) sendEvents(c *gin.Context) {
	var (
		err     error
		payload []byte
	)
	ctx, span := tracing.StartTraceSpan(c.Request.Context(), "controller", string(TOpSendEvents))
	defer func() {
		tracing.EndTraceSpan(span,
			tracing.TraceAttrError(err),
			tracing.TraceAttrPayloadLen(payload),
			tracing.TraceAttrEntrypoint(c.FullPath()),
		)
	}()

	payload, err = c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	err = controller.SendEvents(ctx, payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, nil)
}

// @Description The following API endpoint can be used to send Alerts to Foundation.
// @Tags    alert, event
// @Accept  json
// @Produce json
// @Success 200
// @Failure 401 {string} string "Unauthorized"
// @Router  /events-ack [post]
// @Param   GWOS-APP-NAME    header    string     true        "Auth header"
// @Param   GWOS-API-TOKEN   header    string     true        "Auth header"
func (controller *Controller) sendEventsAck(c *gin.Context) {
	var (
		err     error
		payload []byte
	)
	ctx, span := tracing.StartTraceSpan(c.Request.Context(), "controller", string(TOpSendEventsAck))
	defer func() {
		tracing.EndTraceSpan(span,
			tracing.TraceAttrError(err),
			tracing.TraceAttrPayloadLen(payload),
			tracing.TraceAttrEntrypoint(c.FullPath()),
		)
	}()

	payload, err = c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	err = controller.SendEventsAck(ctx, payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, nil)
}

// @Description The following API endpoint can be used to send Alerts to Foundation.
// @Tags    alert, event
// @Accept  json
// @Produce json
// @Success 200
// @Failure 401 {string} string "Unauthorized"
// @Router  /events-unack [post]
// @Param   GWOS-APP-NAME    header    string     true        "Auth header"
// @Param   GWOS-API-TOKEN   header    string     true        "Auth header"
func (controller *Controller) sendEventsUnack(c *gin.Context) {
	var (
		err     error
		payload []byte
	)
	ctx, span := tracing.StartTraceSpan(c.Request.Context(), "controller", string(TOpSendEventsUnack))
	defer func() {
		tracing.EndTraceSpan(span,
			tracing.TraceAttrError(err),
			tracing.TraceAttrPayloadLen(payload),
			tracing.TraceAttrEntrypoint(c.FullPath()),
		)
	}()

	payload, err = c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	err = controller.SendEventsUnack(ctx, payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, nil)
}

// @Description The following API endpoint can be used to send Inventory to Foundation.
// @Tags    inventory
// @Accept  json
// @Produce json
// @Success 200
// @Failure 401 {string} string "Unauthorized"
// @Router  /inventory [post]
// @Param   GWOS-APP-NAME    header    string     true        "Auth header"
// @Param   GWOS-API-TOKEN   header    string     true        "Auth header"
func (controller *Controller) syncInventory(c *gin.Context) {
	var (
		err     error
		payload []byte
	)
	ctx, span := tracing.StartTraceSpan(c.Request.Context(), "controller", string(TOpSyncInventory))
	defer func() {
		tracing.EndTraceSpan(span,
			tracing.TraceAttrError(err),
			tracing.TraceAttrPayloadLen(payload),
			tracing.TraceAttrEntrypoint(c.FullPath()),
		)
	}()

	payload, err = c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	err = controller.SynchronizeInventory(ctx, payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, nil)
}

// @Description The following API endpoint can be used to send Metrics to Foundation.
// @Tags    metrics
// @Accept  json
// @Produce json
// @Success 200
// @Failure 401 {string} string "Unauthorized"
// @Router  /metrics [post]
// @Param   GWOS-APP-NAME    header    string     true        "Auth header"
// @Param   GWOS-API-TOKEN   header    string     true        "Auth header"
func (controller *Controller) sendMetrics(c *gin.Context) {
	var (
		err     error
		payload []byte
	)
	ctx, span := tracing.StartTraceSpan(c.Request.Context(), "controller", string(TOpSendMetrics))
	defer func() {
		tracing.EndTraceSpan(span,
			tracing.TraceAttrError(err),
			tracing.TraceAttrPayloadLen(payload),
			tracing.TraceAttrEntrypoint(c.FullPath()),
		)
	}()

	payload, err = c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	err = controller.SendResourceWithMetrics(ctx, payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, nil)
}

// @Description The following API endpoint can be used to get list of metrics from the server.
// @Tags    metric
// @Accept  json
// @Produce json
// @Success 200 {object} services.AgentStatus
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal server error"
// @Router  /metrics [get]
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

// @Description The following API endpoint can be used to reset NATS queues.
// @Tags    agent, connector
// @Accept  json
// @Produce json
// @Success 200 {object} services.ConnectorStatusDTO
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal server error"
// @Router  /reset-nats [post]
// @Param   GWOS-APP-NAME    header    string     true        "Auth header"
// @Param   GWOS-API-TOKEN   header    string     true        "Auth header"
func (controller *Controller) resetNats(c *gin.Context) {
	task, err := controller.ResetNatsAsync()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, ConnectorStatusDTO{StatusProcessing, task.Idx})
}

// @Description The following API endpoint can be used to start NATS dispatcher.
// @Tags    agent, connector
// @Accept  json
// @Produce json
// @Success 200 {object} services.ConnectorStatusDTO
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal server error"
// @Router  /start [post]
// @Param   GWOS-APP-NAME    header    string     true        "Auth header"
// @Param   GWOS-API-TOKEN   header    string     true        "Auth header"
func (controller *Controller) start(c *gin.Context) {
	status := controller.Status()
	if status.Transport.Value() == StatusRunning && status.task == nil {
		c.JSON(http.StatusOK, ConnectorStatusDTO{StatusRunning, 0})
		return
	}
	task, err := controller.StartTransportAsync()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, ConnectorStatusDTO{StatusProcessing, task.Idx})
}

// @Description The following API endpoint can be used to stop NATS dispatcher.
// @Tags    agent, connector
// @Accept  json
// @Produce json
// @Success 200 {object} services.ConnectorStatusDTO
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal server error"
// @Router  /stop [post]
// @Param   GWOS-APP-NAME    header    string     true        "Auth header"
// @Param   GWOS-API-TOKEN   header    string     true        "Auth header"
func (controller *Controller) stop(c *gin.Context) {
	status := controller.Status()
	if status.Transport.Value() == StatusStopped && status.task == nil {
		c.JSON(http.StatusOK, ConnectorStatusDTO{StatusStopped, 0})
		return
	}
	task, err := controller.StopTransportAsync()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, ConnectorStatusDTO{StatusProcessing, task.Idx})
}

// @Description The following API endpoint can be used to get TCG statistics.
// @Tags    agent, connector
// @Accept  json
// @Produce json
// @Success 200 {object} services.AgentStatsExt
// @Failure 401 {string} string "Unauthorized"
// @Router  /stats [get]
// @Param   gwos-app-name    header    string     true        "Auth header"
// @Param   GWOS-API-TOKEN   header    string     true        "Auth header"
func (controller *Controller) stats(c *gin.Context) {
	c.JSON(http.StatusOK, controller.Stats())
}

// @Description The following API endpoint can be used to get a TCG agent id
// @Tags    agent, connector
// @Accept  json
// @Produce json
// @Success 200 {object} services.AgentIdentity
// @Router  /agent [get]
func (controller *Controller) agentIdentity(c *gin.Context) {
	c.JSON(http.StatusOK, controller.Connector.AgentIdentity)
}

// @Description The following API endpoint can be used to get TCG status.
// @Tags    agent, connector
// @Accept  json
// @Produce json
// @Success 200 {object} services.ConnectorStatusDTO
// @Failure 401 {string} string "Unauthorized"
// @Router  /status [get]
// @Param   GWOS-APP-NAME    header    string     true        "Auth header"
// @Param   GWOS-API-TOKEN   header    string     true        "Auth header"
func (controller *Controller) status(c *gin.Context) {
	status := controller.Status()
	statusDTO := ConnectorStatusDTO{Status(status.Transport.Value()), 0}
	if status.task != nil {
		statusDTO = ConnectorStatusDTO{StatusProcessing, status.task.Idx}
	}
	c.JSON(http.StatusOK, statusDTO)
}

// @Description The following API endpoint can be used to return actual TCG connector version.
// @Tags    agent, connector
// @Accept  json
// @Produce json
// @Success 200 {object} config.BuildVersion
// @Failure 401 {string} string "Unauthorized"
// @Router  /version [get]
// @Param   GWOS-APP-NAME    header    string     true        "Auth header"
// @Param   GWOS-API-TOKEN   header    string     true        "Auth header"
func (controller *Controller) version(c *gin.Context) {
	c.JSON(http.StatusOK, config.GetBuildInfo())
}

func (controller *Controller) checkAccess(c *gin.Context) {
	if config.GetConfig().IsConfiguringPMC() {
		log.Info().Str("url", c.Request.URL.Redacted()).
			Msg("omit access check on configuring PARENT_MANAGED_CHILD")
		return
	}
	if len(controller.dsClient.HostName) == 0 && len(controller.gwClients) == 0 {
		log.Info().Str("url", c.Request.URL.Redacted()).
			Msg("omit access check on empty config")
		return
	}

	/* check local pin */
	pin := controller.Connector.ControllerPin
	if len(pin) > 0 && pin == c.Request.Header.Get("X-PIN") {
		log.Debug().Str("url", c.Request.URL.Redacted()).
			Msg("access allowed with X-PIN")
		return
	}

	hashFn := func(args ...string) (string, error) {
		h, err := blake2b.New512(nil)
		if err != nil {
			return "", err
		}
		for _, s := range args {
			if _, err := io.WriteString(h, s); err != nil {
				return "", err
			}
		}
		sum := h.Sum(nil)
		return hex.EncodeToString(sum[:]), nil
	}

	/* check basic auth */
	if username, password, hasAuth := c.Request.BasicAuth(); hasAuth {
		var err error
		defer func() {
			if err == nil {
				log.Debug().Str("url", c.Request.URL.Redacted()).
					Str("username", username).
					Msg("access allowed with BASIC")
				return
			}
			log.Warn().Err(err).Str("url", c.Request.URL.Redacted()).
				Str("username", username).
				Msg("access disallowed with BASIC")
			c.AbortWithStatusJSON(http.StatusUnauthorized,
				gin.H{"error": err.Error()})
		}()

		if !(len(username) > 0 && len(password) > 0 && len(controller.gwClients) > 0) {
			err = fmt.Errorf("misconfigured BASIC auth")
			return
		}
		ck, err := hashFn(username, password)
		if err == nil {
			if _, isCached := controller.authCache.Get(ck); !isCached {
				/* restrict by mutex for one-thread at one-time */
				controller.muBASIC.Lock()
				if _, isCached := controller.authCache.Get(ck); !isCached {
					if _, err = controller.gwClients[0].AuthenticatePassword(username, password); err == nil {
						err = controller.authCache.Add(ck, true, time.Hour)
					}
				}
				controller.muBASIC.Unlock()
			}
		}
		return
	}

	/* check gwos auth */
	gwosAppName := c.Request.Header.Get("GWOS-APP-NAME")
	gwosAPIToken := c.Request.Header.Get("GWOS-API-TOKEN")
	var err error
	defer func() {
		if err == nil {
			log.Debug().Str("url", c.Request.URL.Redacted()).
				Str("gwosAppName", gwosAppName).
				Msg("access allowed with GWOS")
			return
		}
		log.Warn().Err(err).Str("url", c.Request.URL.Redacted()).
			Str("gwosAppName", gwosAppName).
			Msg("access disallowed with GWOS")
		c.AbortWithStatusJSON(http.StatusUnauthorized,
			gin.H{"error": err.Error()})
	}()

	if !(len(gwosAppName) > 0 && len(gwosAPIToken) > 0 && len(controller.dsClient.HostName) > 0) {
		err = fmt.Errorf("misconfigured GWOS auth")
		return
	}
	ck, err := hashFn(gwosAppName, gwosAPIToken)
	if err == nil {
		if _, isCached := controller.authCache.Get(ck); !isCached {
			/* restrict by mutex for one-thread at one-time */
			controller.muGWOS.Lock()
			if _, isCached := controller.authCache.Get(ck); !isCached {
				if err = controller.dsClient.ValidateToken(gwosAppName, gwosAPIToken); err == nil {
					err = controller.authCache.Add(ck, true, time.Hour)
				}
			}
			controller.muGWOS.Unlock()
		}
	}
}

func (controller *Controller) registerAPI1(router *gin.Engine, addr string, entrypoints []Entrypoint) {
	swaggerURL := ginSwagger.URL("http://" + addr + "/swagger/doc.json")
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, swaggerURL))

	/* private entrypoints */
	apiV1Group := router.Group("/api/v1")
	apiV1Group.Use(controller.checkAccess)

	apiV1Group.POST("/config", controller.config)
	apiV1Group.POST("/downtime-clear", controller.clearInDowntime)
	apiV1Group.POST("/downtime-set", controller.setInDowntime)
	apiV1Group.POST("/events", controller.sendEvents)
	apiV1Group.POST("/events-ack", controller.sendEventsAck)
	apiV1Group.POST("/events-unack", controller.sendEventsUnack)
	apiV1Group.POST("/inventory", controller.syncInventory)
	apiV1Group.POST("/metrics", controller.sendMetrics)
	apiV1Group.GET("/metrics", controller.listMetrics)
	apiV1Group.POST("/reset-nats", controller.resetNats)
	apiV1Group.POST("/start", controller.start)
	apiV1Group.POST("/stop", controller.stop)

	for _, entrypoint := range entrypoints {
		switch entrypoint.Method {
		case http.MethodGet:
			apiV1Group.GET(entrypoint.URL, entrypoint.Handler)
		case http.MethodPost:
			apiV1Group.POST(entrypoint.URL, entrypoint.Handler)
		case http.MethodPut:
			apiV1Group.PUT(entrypoint.URL, entrypoint.Handler)
		case http.MethodDelete:
			apiV1Group.DELETE(entrypoint.URL, entrypoint.Handler)
		}
	}

	/* public entrypoints */
	router.GET("/api/v1/identity", controller.agentIdentity)
	router.GET("/api/v1/stats", controller.stats)
	router.GET("/api/v1/status", controller.status)
	router.GET("/api/v1/version", controller.version)

	apiV1Debug := router.Group("/api/v1/debug")
	apiV1Debug.GET("/vars", gin.WrapH(expvar.Handler()))
	apiV1Debug.GET("/metrics", func() gin.HandlerFunc {
		if !controller.Connector.ExportProm {
			msg := "export Prometheus metrics is not configured"
			return func(c *gin.Context) { c.AbortWithStatusJSON(http.StatusServiceUnavailable, msg) }
		}
		return gin.WrapH(promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{}))
	}())

	pprofGroup := apiV1Debug.Group("/pprof")
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
