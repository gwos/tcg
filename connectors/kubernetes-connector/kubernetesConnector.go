package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
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
	EndPoint string                                                 `json:"kubernetesClusterEndpoint"`
	Views    map[KubernetesView]map[string]transit.MetricDefinition `json:"views"`
	// Groups        []transit.ResourceGroup                                `json:"groups"`
	CheckInterval time.Duration             `json:"checkIntervalMinutes"`
	Ownership     transit.HostOwnershipType `json:"ownership,omitempty"`
	AuthType      AuthType                  `json:"authType"`
	Insecure      bool                      `json:"insecure"`

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
	ClusterHostGroup = "cluster-"
	ClusterNameLabel = "alpha.eksctl.io/cluster-name"
	ContainerPOD     = "POD"
	PodsHostGroup    = "pods-"

	defaultKubernetesClusterEndpoint = ""
)

const (
	cpu             = "cpu"
	pods            = "pods"
	memory          = "memory"
	cpuCores        = "cpu.cores"
	cpuAllocated    = "cpu.allocated"
	memoryCapacity  = "memory.capacity"
	memoryAllocated = "memory.allocated"
)

type KubernetesConnector struct {
	ExtConfig
	iChksum []byte
	ctx     context.Context

	kapi       kv1.CoreV1Interface
	kClientSet kubernetes.Interface
	mapi       mv1.MetricsV1beta1Interface
}

type Cluster struct {
	Cl struct {
		Server string `yaml:"server"`
	} `yaml:"cluster"`
}

type User struct {
	User struct {
		Token string `yaml:"token"`
	}
}

type KubernetesYaml struct {
	Kind     string    `yaml:"kind"`
	Users    []User    `yaml:"users"`
	Clusters []Cluster `yaml:"clusters"`
}

type KubernetesResource struct {
	Name     string
	Type     transit.ResourceType
	Status   transit.MonitorStatus
	Message  string
	Labels   map[string]string
	Services map[string]transit.MonitoredService
}

func (connector *KubernetesConnector) Initialize(ctx context.Context) error {
	// kubeStateMetricsEndpoint := "http://" + config.EndPoint + "/api/v1/namespaces/kube-system/services/kube-state-metrics:http-metrics/proxy/metrics"
	kConfig := rest.Config{
		Host:                connector.ExtConfig.EndPoint,
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
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: connector.ExtConfig.Insecure,
		},
		UserAgent:          "",
		DisableCompression: false,
		Transport:          nil,
		WrapTransport:      nil,
		QPS:                0,
		Burst:              0,
		RateLimiter:        nil,
		Timeout:            0,
		Dial:               nil,
	}

	switch connector.ExtConfig.AuthType {
	case InCluster:
		log.Info().Msg("using InCluster auth")
	case Credentials:
		kConfig.Username = connector.ExtConfig.KubernetesUserName
		kConfig.Password = connector.ExtConfig.KubernetesUserPassword
		log.Info().Msg("using Credentials auth")
	case BearerToken:
		kConfig.BearerToken = connector.ExtConfig.KubernetesBearerToken
		log.Info().Msg("using Bearer Token auth")
	case ConfigFile:
		fConfig := KubernetesYaml{}
		err := yaml.Unmarshal([]byte(connector.ExtConfig.KubernetesConfigFile), &fConfig)
		if err != nil {
			return err
		}

		if len(fConfig.Clusters) != 1 || len(fConfig.Users) != 1 ||
			fConfig.Clusters[0].Cl.Server == "" || fConfig.Users[0].User.Token == "" || fConfig.Kind != "Config" {
			return errors.New("invalid configuration file")
		}

		kConfig.BearerToken = fConfig.Users[0].User.Token
		kConfig.Host = fConfig.Clusters[0].Cl.Server

		log.Info().Msg("using YAML file auth")
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
	connector.ctx = ctx

	log.Debug().Msgf("initialized Kubernetes connection to server version %s, for client version: %s, and endPoint %s",
		version.String(), connector.kapi.RESTClient().APIVersion(), connector.ExtConfig.EndPoint)

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

// Collect inventory and metrics for all kinds of Kubernetes resources. Sort resources into groups and return inventory of host resources and inventory of groups
func (connector *KubernetesConnector) Collect() ([]transit.InventoryResource, []transit.MonitoredResource, []transit.ResourceGroup) {
	// gather inventory and Metrics
	metricsPerContainer := true
	monitoredState := make(map[string]KubernetesResource)
	groups := make(map[string]transit.ResourceGroup)
	connector.collectNodeInventory(monitoredState, groups)
	connector.collectPodInventory(monitoredState, groups, &metricsPerContainer)
	connector.collectNodeMetrics(monitoredState)
	if metricsPerContainer {
		connector.collectPodMetricsPerContainer(monitoredState)
	} else {
		connector.collectPodMetricsPerReplica(monitoredState)
	}

	// convert to arrays as expected by TCG
	inventory := make([]transit.InventoryResource, 0, len(monitoredState))
	monitored := make([]transit.MonitoredResource, 0, len(monitoredState))
	hostGroups := make([]transit.ResourceGroup, 0, len(groups))
	for _, resource := range monitoredState {
		// convert inventory
		services := make([]transit.InventoryService, 0, len(resource.Services))
		for _, service := range resource.Services {
			services = append(services, connectors.CreateInventoryService(service.Name, service.Owner))
		}
		inventory = append(inventory, connectors.CreateInventoryResource(resource.Name, services))
		// convert monitored state
		mServices := make([]transit.MonitoredService, 0, len(resource.Services))
		for _, service := range resource.Services {
			mServices = append(mServices, service)
		}
		monitored = append(monitored, transit.MonitoredResource{
			BaseResource: transit.BaseResource{
				BaseInfo: transit.BaseInfo{
					Name: resource.Name,
					Type: resource.Type,
				},
			},
			MonitoredInfo: transit.MonitoredInfo{
				Status:           resource.Status,
				LastCheckTime:    transit.NewTimestamp(),
				NextCheckTime:    transit.NewTimestamp(), // TODO: interval
				LastPluginOutput: resource.Message,
			},
			Services: mServices,
		})
	}
	for _, group := range groups {
		hostGroups = append(hostGroups, group)
	}
	return inventory, monitored, hostGroups
}

// Node Inventory also retrieves status, capacity, and allocations
// inventory also contains status, pod counts, capacity and allocation metrics
//
//	Capacity:
//	(v1.ResourceName) (len=6) memory: (resource.Quantity) 3977916Ki,
//	(v1.ResourceName) (len=4) pods: (resource.Quantity) 17,
//	(v1.ResourceName) (len=3) cpu: (resource.Quantity) 2, // 2 cores
//	(v1.ResourceName) (len=17) ephemeral-storage: (resource.Quantity) 20959212Ki,
//	Allocatable:
//	(v1.ResourceName) (len=6) memory: (resource.Quantity) 3422908Ki,
//	(v1.ResourceName) (len=4) pods: (resource.Quantity) 17,
//	(v1.ResourceName) (len=3) cpu: (resource.Quantity) 1930m
//	(v1.ResourceName) (len=17) ephemeral-storage: (resource.Quantity) 18242267924,
func (connector *KubernetesConnector) collectNodeInventory(monitoredState map[string]KubernetesResource, groups map[string]transit.ResourceGroup) {
	nodes, _ := connector.kapi.Nodes().List(connector.ctx, metav1.ListOptions{}) // TODO: ListOptions can filter by label
	clusterHostGroupName := connector.makeClusterName(nodes)
	groups[clusterHostGroupName] = transit.ResourceGroup{
		GroupName: clusterHostGroupName,
		Type:      transit.HostGroup,
		Resources: make([]transit.ResourceRef, len(nodes.Items)),
	}

	for index, node := range nodes.Items {
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
		for key, metricDefinition := range connector.ExtConfig.Views[ViewNodes] {
			var value interface{}
			switch key {
			case cpuCores:
				value = node.Status.Capacity.Cpu().Value()
			case cpuAllocated:
				value = toPercentage(node.Status.Capacity.Cpu().MilliValue(), node.Status.Allocatable.Cpu().MilliValue())
			case memoryCapacity:
				value = node.Status.Capacity.Memory().Value()
			case memoryAllocated:
				value = toPercentage(node.Status.Capacity.Memory().MilliValue(), node.Status.Allocatable.Memory().MilliValue())
			case pods:
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
		groups[clusterHostGroupName].Resources[index] = transit.ResourceRef{
			Name:  resource.Name,
			Owner: clusterHostGroupName,
			Type:  transit.ResourceTypeHost,
		}
	}
}

// Pod Inventory also retrieves status
// inventory also contains status, pod counts, capacity and allocation metrics
func (connector *KubernetesConnector) collectPodInventory(monitoredState map[string]KubernetesResource, groups map[string]transit.ResourceGroup, metricsPerContainer *bool) {
	// TODO: filter pods by namespace(s)
	groupsMap := make(map[string]bool)
	pods, err := connector.kapi.Pods("").List(connector.ctx, metav1.ListOptions{})
	if err != nil {
		log.Err(err).Msg("could not collect pod inventory")
		return
	}

	addResource := func(
		stateKey string,
		resourceName string,
		monitorStatus transit.MonitorStatus,
		message string,
		labels map[string]string,
		podHostGroup string,
	) {
		resource := KubernetesResource{
			Name:     resourceName,
			Type:     transit.ResourceTypeHost,
			Status:   monitorStatus,
			Message:  message,
			Labels:   labels,
			Services: make(map[string]transit.MonitoredService),
		}
		monitoredState[stateKey] = resource
		// no services to process at this stage

		if _, found := groupsMap[resource.Name]; found {
			return
		}
		groupsMap[resource.Name] = true
		// add to namespace group for starters, need to consider namespace filtering
		if group, ok := groups[podHostGroup]; ok {
			group.Resources = append(group.Resources, transit.ResourceRef{
				Name:  resource.Name,
				Owner: group.GroupName,
				Type:  transit.ResourceTypeHost,
			})
			groups[podHostGroup] = group
		} else {
			group = transit.ResourceGroup{
				GroupName: podHostGroup,
				Type:      transit.HostGroup,
				Resources: make([]transit.ResourceRef, 0),
			}
			group.Resources = append(group.Resources, transit.ResourceRef{
				Name:  resource.Name,
				Owner: group.GroupName,
				Type:  transit.ResourceTypeHost,
			})
			groups[podHostGroup] = group
		}
	}

	for _, pod := range pods.Items {
		labels := make(map[string]string)
		for key, element := range pod.Labels {
			labels[key] = element
		}

		monitorStatus, message := connector.calculatePodStatus(&pod)

		if *metricsPerContainer {
			for _, container := range pod.Spec.Containers {
				resourceName := strings.TrimSuffix(container.Name, "-")
				if resourceName == ContainerPOD {
					continue
				}
				addResource(pod.Name+"/"+resourceName,
					resourceName, monitorStatus, message, labels,
					PodsHostGroup+pod.Namespace)
			}
		} else {
			addResource(pod.Name,
				pod.Name, monitorStatus, message, labels,
				PodsHostGroup+pod.Namespace)
		}
	}
}

func (connector *KubernetesConnector) collectNodeMetrics(monitoredState map[string]KubernetesResource) {
	nodes, err := connector.mapi.NodeMetricses().List(connector.ctx, metav1.ListOptions{}) // TODO: filter by namespace
	if err != nil {
		log.Err(err).Msg("could not collect node metrics")
		return
	}

	for _, node := range nodes.Items {
		if resource, ok := monitoredState[node.Name]; ok {
			for key, metricDefinition := range connector.ExtConfig.Views[ViewNodes] {
				var value interface{}
				switch key {
				case cpu:
					value = node.Usage.Cpu().MilliValue()
				case memory:
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
				metricBuilder.StartTimestamp = &transit.Timestamp{Time: node.Timestamp.Time.UTC()}
				metricBuilder.EndTimestamp = &transit.Timestamp{Time: node.Timestamp.Time.UTC()}
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

func (connector *KubernetesConnector) collectPodMetricsPerReplica(monitoredState map[string]KubernetesResource) {
	pods, err := connector.mapi.PodMetricses("").List(connector.ctx, metav1.ListOptions{}) // TODO: filter by namespace
	if err != nil {
		log.Err(err).Msg("could not collect pod metrics")
		return
	}
	for _, pod := range pods.Items {
		if resource, ok := monitoredState[pod.Name]; ok {
			for index, container := range pod.Containers {
				if container.Name == ContainerPOD {
					continue
				}
				metricBuilders := make([]connectors.MetricBuilder, 0)
				for key, metricDefinition := range connector.ExtConfig.Views[ViewPods] {
					var value interface{}
					switch key {
					case cpuCores:
						value = container.Usage.Cpu().Value()
					case cpuAllocated:
						value = container.Usage.Cpu().MilliValue()
					case memoryCapacity:
						value = container.Usage.Memory().Value()
					case memoryAllocated:
						value = container.Usage.Memory().Value()
					case cpu:
						value = pod.Containers[index].Usage.Cpu().MilliValue()
					case memory:
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
					metricBuilder.StartTimestamp = &transit.Timestamp{Time: pod.Timestamp.Time.UTC()}
					metricBuilder.EndTimestamp = &transit.Timestamp{Time: pod.Timestamp.Time.UTC()}
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
func (connector *KubernetesConnector) collectPodMetricsPerContainer(monitoredState map[string]KubernetesResource) {
	pods, err := connector.mapi.PodMetricses("").List(connector.ctx, metav1.ListOptions{}) // TODO: filter by namespace
	if err != nil {
		log.Err(err).Msg("could not collect pod metrics")
		return
	}
	debugDetails := false
	builderMap := make(map[string][]connectors.MetricBuilder)
	serviceMap := make(map[string]transit.MonitoredService)
	for key, metricDefinition := range connector.ExtConfig.Views[ViewPods] {
		for _, pod := range pods.Items {
			for _, container := range pod.Containers {
				if container.Name == ContainerPOD {
					continue
				}
				if resource, ok := monitoredState[pod.Name+"/"+container.Name]; ok {
					var value interface{}
					switch key {
					case cpuCores:
						value = container.Usage.Cpu().Value()
					case cpuAllocated:
						value = container.Usage.Cpu().MilliValue()
					case memoryCapacity:
						value = container.Usage.Memory().Value()
					case memoryAllocated:
						value = container.Usage.Memory().Value()
					case cpu:
						value = container.Usage.Cpu().MilliValue()
					case memory:
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
					metricBuilder.StartTimestamp = &transit.Timestamp{Time: pod.Timestamp.Time.UTC()}
					metricBuilder.EndTimestamp = &transit.Timestamp{Time: pod.Timestamp.Time.UTC()}
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
					log.Error().Msgf("pod container not found in monitored state: %s/%s", pod.Name, container.Name)
					debugDetails = true
				}
			}
		}
	}
	if debugDetails {
		log.Debug().Interface("monitoredState", monitoredState).Send()
	}
}

// Calculate Node Status based on Conditions, PID Pressure, memory Pressure, Disk Pressure all treated as default
func (connector *KubernetesConnector) calculateNodeStatus(node *v1.Node) (transit.MonitorStatus, string) {
	var message strings.Builder
	var upMessage string
	var status = transit.HostUp
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
	var status = transit.HostUp
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
		if value, ok := nodes.Items[0].Labels[ClusterNameLabel]; ok {
			return ClusterHostGroup + value
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
