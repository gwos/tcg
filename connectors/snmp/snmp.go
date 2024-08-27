package snmp

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/snmp/clients"
	"github.com/gwos/tcg/connectors/snmp/utils"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/gwos/tcg/services"
	"github.com/rs/zerolog/log"
)

const (
	defaultNediServer = "nedi:8092"
	hostGroup         = "NEDI-M"
)

// SnmpView describes flow
type SnmpView string

// Define flows
const (
	Interfaces SnmpView = "interfaces"
)

type SnmpConnector struct {
	config     ExtConfig
	nediClient clients.NediClient
	snmpClient clients.SnmpClient
	mState     MonitoringState
}

type ExtConfig struct {
	NediServer    string        `json:"server"`
	CheckInterval time.Duration `json:"checkIntervalMinutes"`
	AppType       string
	AgentID       string
	GWConnections config.GWConnections
	Ownership     transit.HostOwnershipType
	// [viewName][metricName]MetricDefinition
	Views map[string]map[string]transit.MetricDefinition
}

// UnmarshalJSON implements json.Unmarshaler.
func (cfg *ExtConfig) UnmarshalJSON(input []byte) error {
	type plain ExtConfig
	c := plain(*cfg)
	if err := json.Unmarshal(input, &c); err != nil {
		return err
	}
	if c.CheckInterval != cfg.CheckInterval {
		c.CheckInterval = c.CheckInterval * time.Minute
	}
	*cfg = ExtConfig(c)
	return nil
}

func (connector *SnmpConnector) LoadConfig(config ExtConfig) error {
	connector.config = config
	err := connector.nediClient.Init(config.NediServer)
	if err != nil {
		log.Err(err).Msg("could not init NeDi client")
		return errors.New("could not init NeDi client")
	}
	connector.mState.Init()
	return nil
}

func (connector *SnmpConnector) CollectMetrics() ([]transit.MonitoredResource, []transit.InventoryResource,
	[]transit.ResourceGroup, error) {
	if connector.config.Views != nil && len(connector.config.Views) > 0 {
		devices, err := connector.nediClient.GetDevices()
		if err != nil {
			log.Err(err).Msg("could not get devices")
			return nil, nil, nil, errors.New("could not get devices")
		}
		for _, device := range devices {
			deviceExt := DeviceExt{Device: device}
			secData, err := utils.GetSecurityData(device.Community)
			if err != nil {
				log.Err(err).Msgf("could not get security data of device '%s' by community '%s'",
					device.Name, device.Community)
			}
			if secData != nil {
				deviceExt.SecData = secData
			}
			connector.mState.Lock()
			connector.mState.devices[device.Name] = deviceExt
			connector.mState.Unlock()
		}
	}

	metricDefinitions := make(map[string]transit.MetricDefinition)
	for view, metrics := range connector.config.Views {
		switch view {
		case string(Interfaces):
			var mibs []string
			for _, v := range clients.AvailableMetrics {
				mibs = append(mibs, v.Mib)
			}
			connector.collectInterfacesMetrics(mibs)
			break
		default:
			log.Warn().Msgf("not supported view: %s", view)
			continue
		}
		for k, v := range metrics {
			metricDefinitions[k] = v
		}
	}

	mrs := connector.mState.retrieveMonitoredResources(metricDefinitions)
	var (
		irs  = make([]transit.InventoryResource, 0, len(mrs))
		refs = make([]transit.ResourceRef, 0, len(mrs))
	)
	for _, mr := range mrs {
		irs = append(irs, mr.ToInventoryResource())
		refs = append(refs, mr.ToResourceRef())
	}
	rgs := []transit.ResourceGroup{{GroupName: hostGroup, Type: transit.HostGroup, Resources: refs}}

	return mrs, irs, rgs, nil
}

func (connector *SnmpConnector) collectInterfacesMetrics(mibs []string) {
	log.Info().Msg("========= starting collection of interface metrics...")
	for deviceName, device := range connector.mState.devices {
		interfaces, err := connector.nediClient.GetDeviceInterfaces(deviceName)
		if err != nil {
			log.Err(err).Msgf("could not get interfaces of device '%s'", deviceName)
			continue
		}

		if len(interfaces) == 0 {
			log.Info().Msgf("no interfaces of device '%s' found", deviceName)
			continue
		}

		interfacesMap := make(map[int]InterfaceExt)
		for _, iFace := range interfaces {
			interfacesMap[iFace.Index] = InterfaceExt{
				Interface: iFace,
				Metrics:   make(map[string]InterfaceMetric),
			}
		}

		if device.SecData == nil {
			log.Error().Msgf("security data for device '%s' not found: skipping", deviceName)
			continue
		}

		snmpData, err := connector.snmpClient.GetSnmpData(mibs, device.IP, device.SecData)
		if err != nil {
			log.Err(err).Msgf("could not get SNMP data for device '%s'", deviceName)
			continue
		}

		for _, data := range snmpData {
			for _, snmpValue := range data.Values {
				mixes := strings.Split(snmpValue.Name, ".")
				idxMix := mixes[len(mixes)-1]
				idx, err := strconv.Atoi(idxMix)
				if err != nil {
					log.Err(err).Msgf("could not retrieve interface index from '%s', cannot convert '%s' to integer",
						snmpValue.Name, idxMix)
					continue
				}
				if iFace, has := interfacesMap[idx]; has {
					metric := InterfaceMetric{
						Key:   data.SnmpMetric.Key,
						Mib:   data.SnmpMetric.Mib,
						Value: snmpValue.Value,
					}
					iFace.Metrics[metric.Mib] = metric
					interfacesMap[idx] = iFace
				} else {
					log.Warn().Msgf("interface of index '%d' for device '%s' not found", idx, deviceName)
				}
			}
		}

		connector.mState.Lock()
		device.Interfaces = interfacesMap
		connector.mState.devices[deviceName] = device
		connector.mState.Unlock()
	}
	log.Info().Msg("========= ending collection of interface metrics...")
}

func (connector *SnmpConnector) listSuggestions(view string, name string) []string {
	var suggestions []string
	switch view {
	case string(Interfaces):
		for k := range clients.AvailableMetrics {
			if name == "" || strings.Contains(k, name) {
				suggestions = append(suggestions, k)
			}
		}
		for k := range clients.NonMibMetrics {
			if name == "" || strings.Contains(k, name) {
				suggestions = append(suggestions, k)
			}
		}
	default:
		log.Warn().Msgf("not supported view: %s", view)
	}
	return suggestions
}

func (connector *SnmpConnector) getInventoryHashSum() ([]byte, error) {
	var (
		hostsServices []string
		devices       = make([]string, 0, len(connector.mState.devices))
	)
	for deviceName := range connector.mState.devices {
		devices = append(devices, deviceName)
	}
	sort.Strings(devices)
	for _, host := range devices {
		device := connector.mState.devices[host]
		var interfaces []string
		for _, iFace := range device.Interfaces {
			interfaces = append(interfaces, iFace.Name)
		}
		sort.Strings(interfaces)
		for _, service := range interfaces {
			hostsServices = append(hostsServices, host+":"+service)
		}
	}
	return connectors.Hashsum(hostsServices)
}

func initializeEntryPoints() []services.Entrypoint {
	return []services.Entrypoint{
		{
			URL:    "/suggest/:viewName",
			Method: http.MethodGet,
			Handler: func(c *gin.Context) {
				c.JSON(http.StatusOK, connector.listSuggestions(c.Param("viewName"), ""))
			},
		},
		{
			URL:    "/suggest/:viewName/:name",
			Method: http.MethodGet,
			Handler: func(c *gin.Context) {
				c.JSON(http.StatusOK, connector.listSuggestions(c.Param("viewName"), c.Param("name")))
			},
		},
		{
			URL:    "/expressions/suggest/:name",
			Method: http.MethodGet,
			Handler: func(c *gin.Context) {
				c.JSON(http.StatusOK, connectors.ListExpressions(c.Param("name")))
			},
		},
		{
			URL:     "/expressions/evaluate",
			Method:  http.MethodPost,
			Handler: connectors.EvaluateExpressionHandler,
		},
	}
}
