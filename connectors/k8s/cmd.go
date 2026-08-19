package k8s

import (
	"bytes"
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/gwos/tcg/services"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

const failuresToStop = 3

var (
	ctxCancel, cancel = context.WithCancel(context.Background())
	connector         = KubernetesConnector{}

	stateMu     sync.Mutex
	failures    int
	stoppedByUs bool

	transportRunning = func() bool {
		return services.GetTransitService().Status().Transport.Value() == services.StatusRunning
	}
	stopTransport  = func() error { return services.GetTransitService().StopTransport() }
	startTransport = func() error { return services.GetTransitService().StartTransport() }
)

func markCollectFailed(err error) {
	stateMu.Lock()
	defer stateMu.Unlock()

	failures++
	if stoppedByUs {
		return
	}
	if !IsAuthError(err) && failures < failuresToStop {
		return
	}
	if !transportRunning() {
		return
	}
	log.Error().Err(err).Int("failures", failures).
		Msg("k8s: data collection keeps failing, marking the connector stopped until it recovers")
	if err := stopTransport(); err != nil {
		log.Err(err).Msg("could not stop transport")
		return
	}
	stoppedByUs = true
}

func markCollectOk() {
	stateMu.Lock()
	defer stateMu.Unlock()

	failures = 0
	if !stoppedByUs {
		return
	}
	if err := startTransport(); err != nil {
		log.Err(err).Msg("could not start transport")
		return
	}
	stoppedByUs = false
	log.Info().Msg("k8s: data collection resumed, the connector is running again")
}

func resetCollectFailures() {
	stateMu.Lock()
	defer stateMu.Unlock()
	failures = 0
}

func Run() {
	transitService := services.GetTransitService()
	transitService.RegisterConfigHandler(configHandler)
	transitService.RegisterExitHandler(cancel)

	log.Info().Msg("Waiting for configuration to be delivered ...")
	if err := transitService.DemandConfig(); err != nil {
		log.Err(err).Msg("Could not demand config")
		return
	}

	if err := connectors.Start(); err != nil {
		log.Err(err).Msg("Could not start connector")
		return
	}

	connectors.StartPeriodic(ctxCancel, connectors.CheckInterval, periodicHandler)

	/* return on quit signal */
	<-transitService.Quit()
}

func configHandler(data []byte) {
	log.Info().
		Func(func(e *zerolog.Event) {
			if zerolog.GlobalLevel() <= zerolog.DebugLevel {
				e.RawJSON("data", data)
			}
		}).
		Msg("Configuration received")
	/* Init config with default values */
	tExt := &ExtConfig{
		EndPoint:  defaultKubernetesClusterEndpoint,
		Ownership: transit.Yield,
		Views:     make(map[KubernetesView]map[string]transit.MetricDefinition),
		// Groups:    []transit.ResourceGroup{},
	}
	tMonConn := &transit.MonitorConnection{Extensions: tExt}
	tMetProf := &transit.MetricsProfile{}

	err := connectors.UnmarshalConfig(data, tMetProf, tMonConn)
	if err != nil {
		log.Err(err).Msg("Could not parse config")
		return
	}

	var yamlData []byte
	switch tExt.AuthType {
	case ConfigFile:
		yamlData = []byte(tExt.KubernetesConfigFile)
	default:
		yamlData, err = yaml.Marshal(struct {
			Auth                   AuthType `yaml:"auth"`
			EndPoint               string   `yaml:"host"`
			KubernetesUserName     string   `yaml:"username,omitempty"`
			KubernetesUserPassword string   `yaml:"password,omitempty"`
			KubernetesBearerToken  string   `yaml:"token,omitempty"`
		}{
			tExt.AuthType,
			tExt.EndPoint,
			tExt.KubernetesUserName,
			tExt.KubernetesUserPassword,
			tExt.KubernetesBearerToken,
		})
		if err != nil {
			log.Err(err).Msg("Could not marshal struct to yaml")
		}
	}
	if err := writeDataToFile(yamlData); err != nil {
		log.Err(err).Msg("Could not write to file")
	}

	/* Update config with received values */
	tExt.GWMapping.Prepare()
	tExt.Views[ViewNodes] = buildNodeMetricsMap(tMetProf.Metrics)
	tExt.Views[ViewPods] = buildPodMetricsMap(tMetProf.Metrics)

	for _, conn := range config.GetConfig().GWConnections {
		if conn.DeferOwnership != "" {
			ownership := transit.HostOwnershipType(conn.DeferOwnership)
			if ownership != "" {
				tExt.Ownership = ownership
				break
			}
		}
	}

	/* Restart periodic loop */
	cancel()
	ctxCancel, cancel = context.WithCancel(context.Background())
	services.GetTransitService().RegisterExitHandler(cancel)

	connector = KubernetesConnector{ExtConfig: *tExt}
	resetCollectFailures()
	if tMonConn.ConnectorID != 0 {
		connectors.StartPeriodic(ctxCancel, connectors.CheckInterval, periodicHandler)
	}
}

func periodicHandler() {
	if connector.kapi == nil {
		if err := connector.Initialize(ctxCancel); err != nil {
			log.Err(err).Msg("Could not initialize connector")
			markCollectFailed(err)
			return
		}
	}

	inventory, monitored, groups, softErr, err := connector.Collect()
	log.Err(err).Msgf("Collect data  %d:%d:%d", len(inventory), len(monitored), len(groups))
	if err != nil {
		markCollectFailed(err)
		return
	}

	if chk, err := connectors.Hashsum(inventory, groups); err != nil || !bytes.Equal(connector.iChksum, chk) {
		if err == nil {
			connector.iChksum = chk
		}

		log.Err(connectors.SendInventory(ctxCancel, inventory, groups, connector.ExtConfig.Ownership)).
			Msg("Sending inventory")
		// TODO: better way to assure sync completion?
		time.Sleep(8 * time.Second)
	}

	log.Err(connectors.SendMetrics(ctxCancel, monitored, &groups)).
		Msg("Sending metrics")

	if softErr != nil {
		markCollectFailed(softErr)
		return
	}
	markCollectOk()
}

func buildNodeMetricsMap(metricsArray []transit.MetricDefinition) map[string]transit.MetricDefinition {
	metrics := make(map[string]transit.MetricDefinition)
	for _, metric := range metricsArray {
		if metric.ServiceType == string(ViewNodes) {
			metrics[metric.Name] = metric
		}
	}

	// TODO: storage is not supported yet
	return metrics
}

func buildPodMetricsMap(metricsArray []transit.MetricDefinition) map[string]transit.MetricDefinition {
	metrics := make(map[string]transit.MetricDefinition)
	for _, metric := range metricsArray {
		if metric.ServiceType == string(ViewPods) {
			metrics[metric.Name] = metric
		}
	}

	// TODO: storage is not supported yet
	return metrics
}

func writeDataToFile(data []byte) error {
	strPath := config.GetConfig().ConfigPath()
	strArray := strings.Split(strPath, "/")
	finalPath := ""
	for i := 0; i < len(strArray)-1; i++ {
		finalPath += strArray[i] + "/"
	}
	finalPath += "kubernetes_config.yaml"
	return os.WriteFile(finalPath, data, 0644)
}
