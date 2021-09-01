package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/transit"
	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	metricsApi "k8s.io/metrics/pkg/client/clientset/versioned"
	mv1 "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
)

// ExtConfig defines the MonitorConnection extensions configuration
// extended with general configuration fields
type ExtConfig struct {
	EndPoint      string                                                 `json:"kubernetesClusterEndpoint"`
	Views         map[KubernetesView]map[string]transit.MetricDefinition `json:"views"`
	Groups        []transit.ResourceGroup                                `json:"groups"`
	CheckInterval time.Duration                                          `json:"checkIntervalMinutes"`
	Ownership     transit.HostOwnershipType                              `json:"ownership,omitempty"`
	AuthType      AuthType                                               `json:"authType"`

	KubernetesUserName     string `json:"kubernetesUserName,omitempty"`
	KubernetesUserPassword string `json:"kubernetesUserPassword,omitempty"`
	KubernetesBearerToken  string `json:"kubernetesBearerToken,omitempty"`
	KubernetesConfigFile   string `json:"kubernetesConfigFile,omitempty"`
}

type KubernetesView string

const (
	ViewNodes KubernetesView = "Nodes"
	ViewPods  KubernetesView = "Pods"
)

type AuthType string

const (
	InCluster   AuthType = "InCluster"
	Credentials AuthType = "UsernamePassword"
	BearerToken AuthType = "BearerToken"
	ConfigFile  AuthType = "ConfigFile"
)

const (
	ClusterHostGroup                 = "cluster-"
	ClusterNameLabel                 = "alpha.eksctl.io/cluster-name"
	PodsHostGroup                    = "pods-"
	NamespaceDefault                 = "default"
	defaultKubernetesClusterEndpoint = "gwos.bluesunrise.com:8001"
)

type KubernetesConnector struct {
	config     ExtConfig
	kapi       kv1.CoreV1Interface
	kClientSet kubernetes.Interface
	mapi       mv1.MetricsV1beta1Interface
	ctx        context.Context
}

type KubernetesResource struct {
	Name     string
	Type     transit.ResourceType
	Status   transit.MonitorStatus
	Message  string
	Labels   map[string]string
	Services map[string]transit.MonitoredService
}

func (connector *KubernetesConnector) Initialize(config ExtConfig) error {
	// kubeStateMetricsEndpoint := "http://" + config.EndPoint + "/api/v1/namespaces/kube-system/services/kube-state-metrics:http-metrics/proxy/metrics"
	kConfig := rest.Config{
		Host:                config.EndPoint,
		APIPath:             "",
		ContentConfig:       rest.ContentConfig{},
		Username:            "",
		Password:            "",
		BearerToken:         "",
		BearerTokenFile:     "",
		Impersonate:         rest.ImpersonationConfig{},
		AuthProvider:        nil,
		AuthConfigPersister: nil,
		ExecProvider:        nil,
		TLSClientConfig:     rest.TLSClientConfig{},
		UserAgent:           "",
		DisableCompression:  false,
		Transport:           nil,
		WrapTransport:       nil,
		QPS:                 0,
		Burst:               0,
		RateLimiter:         nil,
		Timeout:             0,
		Dial:                nil,
	}

	switch config.AuthType {
	case InCluster:
		log.Info().Msg("using InCluster auth")
	case Credentials:
		kConfig.Username = extConfig.KubernetesUserName
		kConfig.Password = extConfig.KubernetesUserPassword
		log.Info().Msg("using Credentials auth")
	case BearerToken:
		kConfig.BearerToken = extConfig.KubernetesBearerToken
		log.Info().Msg("using Bearer Token auth")
	case ConfigFile:
		// TODO:
		log.Info().Msg("using YAML File auth")
	}

	x, err := kubernetes.NewForConfig(&kConfig)
	if err != nil {
		return err
	}
	connector.kClientSet = x
	mClientSet, err := metricsApi.NewForConfig(&kConfig)
	if err != nil {
		return err
	}

	version, err := connector.kClientSet.Discovery().ServerVersion()
	if err != nil {
		return err
	}
	connector.kapi = connector.kClientSet.CoreV1()
	connector.mapi = mClientSet.MetricsV1beta1()
	connector.ctx = context.TODO()

	log.Debug().Msgf("initialized Kubernetes connection to server version %s, for client version: %s, and endPoint %s",
		version.String(), connector.kapi.RESTClient().APIVersion(), config.EndPoint)

	return nil
}

func (connector *KubernetesConnector) Ping() error {
	if connector.kClientSet == nil || connector.kapi == nil {
		return errors.New("kubernetes connector not initialized")
	}
	_, err := connector.kClientSet.Discovery().ServerVersion()
	if err != nil {
		return err
	}
	return nil
}

func (connector *KubernetesConnector) Shutdown() {
}

// Collect inventory and metrics for all kinds of Kubernetes resources. Sort resources into groups and return inventory of host resources and inventory of groups
func (connector *KubernetesConnector) Collect(cfg *ExtConfig) ([]transit.InventoryResource, []transit.MonitoredResource, []transit.ResourceGroup) {

	// gather inventory and Metrics
	metricsPerContainer := true
	monitoredState := make(map[string]KubernetesResource)
	groups := make(map[string]transit.ResourceGroup)
	connector.collectNodeInventory(monitoredState, groups, cfg)
	connector.collectPodInventory(monitoredState, groups, cfg, &metricsPerContainer)
	connector.collectNodeMetrics(monitoredState, cfg)
	if metricsPerContainer {
		connector.collectPodMetricsPerContainer(monitoredState, cfg)
	} else {
		connector.collectPodMetricsPerReplica(monitoredState, cfg)
	}

	// convert to arrays as expected by TCG
	inventory := make([]transit.InventoryResource, len(monitoredState))
	monitored := make([]transit.MonitoredResource, len(monitoredState))
	hostGroups := make([]transit.ResourceGroup, len(groups))
	index := 0
	for _, resource := range monitoredState {
		// convert inventory
		services := make([]transit.InventoryService, len(resource.Services))
		serviceIndex := 0
		for _, service := range resource.Services {
			services[serviceIndex] = connectors.CreateInventoryService(service.Name, service.Owner)
			serviceIndex = serviceIndex + 1
		}
		inventory[index] = connectors.CreateInventoryResource(resource.Name, services)
		// convert monitored state
		mServices := make([]transit.MonitoredService, len(resource.Services))
		serviceIndex = 0
		for _, service := range resource.Services {
			mServices[serviceIndex] = service
			serviceIndex = serviceIndex + 1
		}
		monitored[index] = transit.MonitoredResource{
			BaseResource: transit.BaseResource{
				BaseTransitData: transit.BaseTransitData{
					Name: resource.Name,
					Type: resource.Type,
				},
			},
			Status:           resource.Status,
			LastCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now()},
			NextCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now()}, // TODO: interval
			LastPlugInOutput: resource.Message,
			Services:         mServices,
		}
		index = index + 1
	}
	index = 0
	for _, group := range groups {
		hostGroups[index] = group
		index = index + 1
	}
	return inventory, monitored, hostGroups
}

// Node Inventory also retrieves status, capacity, and allocations
// inventory also contains status, pod counts, capacity and allocation metrics
// Capacity:
//  (v1.ResourceName) (len=6) memory: (resource.Quantity) 3977916Ki,
//	(v1.ResourceName) (len=4) pods: (resource.Quantity) 17,
//	(v1.ResourceName) (len=3) cpu: (resource.Quantity) 2, // 2 cores
//	(v1.ResourceName) (len=17) ephemeral-storage: (resource.Quantity) 20959212Ki,
// Allocatable:
//	(v1.ResourceName) (len=6) memory: (resource.Quantity) 3422908Ki,
//	(v1.ResourceName) (len=4) pods: (resource.Quantity) 17,
//	(v1.ResourceName) (len=3) cpu: (resource.Quantity) 1930m
//	(v1.ResourceName) (len=17) ephemeral-storage: (resource.Quantity) 18242267924,
func (connector *KubernetesConnector) collectNodeInventory(monitoredState map[string]KubernetesResource, groups map[string]transit.ResourceGroup, cfg *ExtConfig) {
	nodes, _ := connector.kapi.Nodes().List(connector.ctx, metav1.ListOptions{}) // TODO: ListOptions can filter by label
	clusterHostGroupName := connector.makeClusterName(nodes)
	groups[clusterHostGroupName] = transit.ResourceGroup{
		GroupName: clusterHostGroupName,
		Type:      transit.HostGroup,
		Resources: make([]transit.MonitoredResourceRef, len(nodes.Items)),
	}
	index := 0
	for _, node := range nodes.Items {
		labels := make(map[string]string)
		for key, element := range node.Labels {
			labels[key] = element
		}
		monitorStatus, message := connector.calculateNodeStatus(&node)
		resource := KubernetesResource{
			Name:     node.Name,
			Type:     transit.ResourceTypeHost,
			Status:   monitorStatus,
			Message:  message,
			Labels:   labels,
			Services: make(map[string]transit.MonitoredService),
		}
		monitoredState[resource.Name] = resource
		// process services
		for key, metricDefinition := range cfg.Views[ViewNodes] {
			var value interface{} = 0
			switch key {
			case "cpu.cores":
				value = node.Status.Capacity.Cpu().Value()
			case "cpu.allocated":
				value = toPercentage(node.Status.Capacity.Cpu().MilliValue(), node.Status.Allocatable.Cpu().MilliValue())
			case "memory.capacity":
				value = node.Status.Capacity.Memory().Value()
			case "memory.allocated":
				value = toPercentage(node.Status.Capacity.Memory().MilliValue(), node.Status.Allocatable.Memory().MilliValue())
			case "pods":
				value = node.Status.Capacity.Pods().Value()
			default:
				continue
			}
			//if value > 4000000 { // TODO: how do we handle longs
			//	value = value / 1000
			//}
			metricBuilder := connectors.MetricBuilder{
				Name:       key,
				CustomName: metricDefinition.CustomName,
				Value:      value,
				UnitType:   transit.UnitCounter,
				// TODO: add these after merge
				//ComputeType: metricDefinition.ComputeType,
				//Expression:  metricDefinition.Expression,
				Warning:  metricDefinition.WarningThreshold,
				Critical: metricDefinition.CriticalThreshold,
			}
			customServiceName := connectors.Name(metricBuilder.Name, metricDefinition.CustomName)
			monitoredService, err := connectors.BuildServiceForMetric(node.Name, metricBuilder)
			if err != nil {
				log.Err(err).Msgf("could not create service %s:%s", node.Name, customServiceName)
			}
			if monitoredService != nil {
				resource.Services[metricBuilder.Name] = *monitoredService
			}

		}
		// add to default Cluster group
		groups[clusterHostGroupName].Resources[index] = transit.MonitoredResourceRef{
			Name:  resource.Name,
			Owner: clusterHostGroupName,
			Type:  transit.ResourceTypeHost,
		}
		index = index + 1
	}
}

// Pod Inventory also retrieves status
// inventory also contains status, pod counts, capacity and allocation metrics
func (connector *KubernetesConnector) collectPodInventory(monitoredState map[string]KubernetesResource, groups map[string]transit.ResourceGroup, cfg *ExtConfig, metricsPerContainer *bool) {
	// TODO: filter pods by namespace(s)
	groupsMap := make(map[string]bool)
	pods, err := connector.kapi.Pods("").List(connector.ctx, metav1.ListOptions{})
	if err != nil {
		log.Err(err).Msg("could not collect pod inventory")
		return
	}
	for _, pod := range pods.Items {
		labels := make(map[string]string)
		for key, element := range pod.Labels {
			labels[key] = element
		}
		podName := pod.Name
		if *metricsPerContainer {
			podName = strings.TrimSuffix(pod.Spec.Containers[0].Name, "-")
		}
		monitorStatus, message := connector.calculatePodStatus(&pod)
		resource := KubernetesResource{
			Name:     podName,
			Type:     transit.ResourceTypeHost,
			Status:   monitorStatus,
			Message:  message,
			Labels:   labels,
			Services: make(map[string]transit.MonitoredService),
		}
		monitoredState[resource.Name] = resource
		// no services to process at this stage

		// add to namespace group for starters, need to consider namespace filtering
		podHostGroup := PodsHostGroup + pod.Namespace
		if _, found := groupsMap[resource.Name]; found {
			continue
		}
		groupsMap[resource.Name] = true
		if group, ok := groups[podHostGroup]; ok {
			group.Resources = append(group.Resources, transit.MonitoredResourceRef{
				Name:  resource.Name,
				Owner: group.GroupName,
				Type:  transit.ResourceTypeHost,
			})
			groups[podHostGroup] = group
		} else {
			group = transit.ResourceGroup{
				GroupName: podHostGroup,
				Type:      transit.HostGroup,
				Resources: make([]transit.MonitoredResourceRef, 0),
			}
			group.Resources = append(group.Resources, transit.MonitoredResourceRef{
				Name:  resource.Name,
				Owner: group.GroupName,
				Type:  transit.ResourceTypeHost,
			})
			groups[podHostGroup] = group
		}
	}

}

func (connector *KubernetesConnector) collectNodeMetrics(monitoredState map[string]KubernetesResource, cfg *ExtConfig) {
	nodes, err := connector.mapi.NodeMetricses().List(connector.ctx, metav1.ListOptions{}) // TODO: filter by namespace
	if err != nil {
		log.Err(err).Msg("could not collect node metrics")
		return
	}

	for _, node := range nodes.Items {
		if resource, ok := monitoredState[node.Name]; ok {
			for key, metricDefinition := range cfg.Views[ViewNodes] {
				var value interface{} = 0
				switch key {
				case "cpu":
					value = node.Usage.Cpu().MilliValue()
				case "memory":
					value = node.Usage.Memory().MilliValue()
				default:
					continue
				}
				metricBuilder := connectors.MetricBuilder{
					Name:       key,
					CustomName: metricDefinition.CustomName,
					// TODO: add these after merge
					//ComputeType: metricDefinition.ComputeType,
					//Expression:  metricDefinition.Expression,
					Value:    value,
					UnitType: transit.UnitCounter,
					Warning:  metricDefinition.WarningThreshold,
					Critical: metricDefinition.CriticalThreshold,
				}
				metricBuilder.StartTimestamp = &milliseconds.MillisecondTimestamp{Time: node.Timestamp.Time}
				metricBuilder.EndTimestamp = &milliseconds.MillisecondTimestamp{Time: node.Timestamp.Time}
				customServiceName := connectors.Name(metricBuilder.Name, metricDefinition.CustomName)
				monitoredService, err := connectors.BuildServiceForMetric(node.Name, metricBuilder)
				if err != nil {
					log.Err(err).Msgf("could not create service %s:%s", node.Name, customServiceName)
				}
				if monitoredService != nil {
					resource.Services[metricBuilder.Name] = *monitoredService
				}
			}
		} else {
			log.Error().Msgf("node not found in monitored state: %s", node.Name)
		}
	}
}

func (connector *KubernetesConnector) collectPodMetricsPerReplica(monitoredState map[string]KubernetesResource, cfg *ExtConfig) {
	pods, err := connector.mapi.PodMetricses("").List(connector.ctx, metav1.ListOptions{}) // TODO: filter by namespace
	if err != nil {
		log.Err(err).Msg("could not collect pod metrics")
		return
	}
	for _, pod := range pods.Items {
		if resource, ok := monitoredState[pod.Name]; ok {
			for index, container := range pod.Containers {
				metricBuilders := make([]connectors.MetricBuilder, 0)
				for key, metricDefinition := range cfg.Views[ViewPods] {
					var value interface{} = 0
					switch key {
					case "cpu.cores":
						value = container.Usage.Cpu().Value()
					case "cpu.allocated":
						value = container.Usage.Cpu().MilliValue()
					case "memory.capacity":
						value = container.Usage.Memory().Value()
					case "memory.allocated":
						value = container.Usage.Memory().Value()
					case "cpu":
						value = pod.Containers[index].Usage.Cpu().MilliValue()
					case "memory":
						value = pod.Containers[index].Usage.Memory().Value()
					default:
						continue
					}
					metricBuilder := connectors.MetricBuilder{
						Name: metricDefinition.Name,
						// CustomName: metricDefinition.CustomName,
						// TODO: add these after merge
						//ComputeType: metricDefinition.ComputeType,
						//Expression:  metricDefinition.Expression,
						Value:    value,
						UnitType: transit.UnitCounter,
						Warning:  metricDefinition.WarningThreshold,
						Critical: metricDefinition.CriticalThreshold,
					}
					metricBuilder.StartTimestamp = &milliseconds.MillisecondTimestamp{Time: pod.Timestamp.Time}
					metricBuilder.EndTimestamp = &milliseconds.MillisecondTimestamp{Time: pod.Timestamp.Time}
					metricBuilders = append(metricBuilders, metricBuilder)
					monitoredService, err := connectors.BuildServiceForMultiMetric(container.Name, metricDefinition.Name, metricDefinition.CustomName, metricBuilders)
					if err != nil {
						log.Err(err).Msgf("could not create service %s:%s", pod.Name, metricDefinition.Name)
					}
					if monitoredService != nil {
						resource.Services[metricBuilder.Name] = *monitoredService
					}
				}
			}
		} else {
			log.Error().Msgf("pod not found in monitored state: %s", pod.Name)
		}
	}
}

// treat each container uniquely -- store multi-metrics per pod replica for each node
func (connector *KubernetesConnector) collectPodMetricsPerContainer(monitoredState map[string]KubernetesResource, cfg *ExtConfig) {
	pods, err := connector.mapi.PodMetricses("").List(connector.ctx, metav1.ListOptions{}) // TODO: filter by namespace
	if err != nil {
		log.Err(err).Msg("could not collect pod metrics")
		return
	}
	builderMap := make(map[string][]connectors.MetricBuilder)
	serviceMap := make(map[string]transit.MonitoredService)
	for key, metricDefinition := range cfg.Views[ViewPods] {
		for _, pod := range pods.Items {
			for _, container := range pod.Containers {
				if resource, ok := monitoredState[container.Name]; ok {
					var value interface{} = 0
					switch key {
					case "cpu.cores":
						value = container.Usage.Cpu().Value()
					case "cpu.allocated":
						value = container.Usage.Cpu().MilliValue()
					case "memory.capacity":
						value = container.Usage.Memory().Value()
					case "memory.allocated":
						value = container.Usage.Memory().Value()
					case "cpu":
						value = container.Usage.Cpu().MilliValue()
					case "memory":
						value = container.Usage.Memory().Value()
					default:
						continue
					}
					splits := strings.Split(pod.Name, "-")
					suffix := ""
					if len(splits) > 1 {
						suffix = splits[len(splits)-1]
					}
					metricBuilder := connectors.MetricBuilder{
						Name: metricDefinition.Name + "-" + suffix,
						// CustomName: metricDefinition.CustomName,
						// TODO: add these after merge
						//ComputeType: metricDefinition.ComputeType,
						//Expression:  metricDefinition.Expression,
						Value:    value,
						UnitType: transit.UnitCounter,
						Warning:  metricDefinition.WarningThreshold,
						Critical: metricDefinition.CriticalThreshold,
					}
					metricBuilder.StartTimestamp = &milliseconds.MillisecondTimestamp{Time: pod.Timestamp.Time}
					metricBuilder.EndTimestamp = &milliseconds.MillisecondTimestamp{Time: pod.Timestamp.Time}
					var builders []connectors.MetricBuilder
					if result, found := builderMap[resource.Name]; found {
						builders = result
					} else {
						builders = make([]connectors.MetricBuilder, 0)
						builderMap[resource.Name] = builders
					}
					builders = append(builders, metricBuilder)
					smKey := key + "-" + resource.Name
					var monitoredService *transit.MonitoredService
					if result, found := serviceMap[smKey]; found {
						monitoredService = &result
						metric, _ := connectors.BuildMetric(metricBuilder)
						monitoredService.Metrics = append(monitoredService.Metrics, *metric)
					} else {
						monitoredService, _ = connectors.BuildServiceForMultiMetric(resource.Name, metricDefinition.Name, metricDefinition.CustomName, builders)
						serviceMap[smKey] = *monitoredService
					}
					if monitoredService != nil {
						resource.Services[metricDefinition.Name] = *monitoredService
					}
				} else {
					log.Error().Msgf("pod not found in monitored state: %s", pod.Name)
				}
			}
		}
	}
}

// Calculate Node Status based on Conditions, PID Pressure, Memory Pressure, Disk Pressure all treated as default
func (connector *KubernetesConnector) calculateNodeStatus(node *v1.Node) (transit.MonitorStatus, string) {
	var message strings.Builder
	var upMessage string
	var status transit.MonitorStatus = transit.HostUp
	for _, condition := range node.Status.Conditions {
		switch condition.Type {
		case v1.NodeReady:
			if condition.Status != v1.ConditionTrue {
				if message.Len() > 0 {
					message.WriteString(", ")
				}
				message.WriteString(condition.Message)
				status = transit.HostUnscheduledDown
			} else {
				upMessage = condition.Message
			}
		default:
			if condition.Status == v1.ConditionTrue {
				if message.Len() > 0 {
					message.WriteString(", ")
				}
				message.WriteString(condition.Message)
				if status == transit.HostUp {
					status = transit.HostWarning
				}
			}
		}

	}
	if status == transit.HostUp {
		message.WriteString(upMessage)
	}
	return status, message.String()
}

func (connector *KubernetesConnector) calculatePodStatus(pod *v1.Pod) (transit.MonitorStatus, string) {
	var message strings.Builder
	var upMessage = "Pod is healthy"
	var status transit.MonitorStatus = transit.HostUp
	for _, condition := range pod.Status.Conditions {
		if condition.Status != v1.ConditionTrue {
			if message.Len() > 0 {
				message.WriteString(", ")
			}
			message.WriteString(condition.Message)
			status = transit.HostUnscheduledDown
		}
	}
	if status == transit.HostUp {
		message.WriteString(upMessage)
	}
	_ = connector.mapi
	return status, message.String()
}

func (connector *KubernetesConnector) makeClusterName(nodes *v1.NodeList) string {
	if len(nodes.Items) > 0 {
		for key, value := range nodes.Items[0].Labels {
			if key == ClusterNameLabel {
				return ClusterHostGroup + value
			}
		}
	}
	return ClusterHostGroup + "1"
}

// toPercentage - converts CPU from cores to percentage
//
// 1 core = 1000 Millicores = 100%
// Value we get from Cpu in cores can be transformed
// to percentage using mathematical formula:
// percentage = cores * 1000 * 100 / 1000
// let's simplify:
// percentage = cores * 100
//
// Example:
// You might want to assign a third of CPU each — or 33.33%.
// If you wish to assign a third of a CPU, you should assign 333Mi (millicores) or 0.333(cores) to your container.
func toPercentage(capacityMilliValue, allocatableMilliValue int64) float64 {
	return float64(allocatableMilliValue) / float64(capacityMilliValue) * 100
}
