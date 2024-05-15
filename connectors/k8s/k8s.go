package k8s

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/sdk/clients"
	"github.com/gwos/tcg/sdk/mapping"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
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

	GWMapping
}

type GWMapping struct {
	HostGroup mapping.Mappings `json:"mapHostgroup"`
	HostName  mapping.Mappings `json:"mapHostname"`
}

// Prepare compiles mappings
func (m *GWMapping) Prepare() {
	var hg, hn mapping.Mappings
	for i := range m.HostGroup {
		if err := m.HostGroup[i].Compile(); err != nil {
			log.Warn().Err(err).Interface("mapping", m.HostGroup[i]).Msg("could not prepare mapping")
			continue
		}
		hg = append(hg, m.HostGroup[i])
	}
	for i := range m.HostName {
		if err := m.HostName[i].Compile(); err != nil {
			log.Warn().Err(err).Interface("mapping", m.HostName[i]).Msg("could not prepare mapping")
			continue
		}
		hn = append(hn, m.HostName[i])
	}
	m.HostGroup, m.HostName = hg, hn
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
	ContainerPOD     = "POD"
	ClusterNameLabel = "alpha.eksctl.io/cluster-name"
	// ClusterHostGroup = "cluster-"
	// PodsHostGroup    = "pods-"

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

	kClientSet kubernetes.Interface
	kapi       kv1.CoreV1Interface
	mapi       mv1.MetricsV1beta1Interface
}

type Cluster struct {
	Name    string `yaml:"name"`
	Cluster struct {
		// CAData contains PEM-encoded certificate authority certificates.
		CAData string `yaml:"certificate-authority-data"`
		// Server is the address of the kubernetes cluster (https://hostname:port).
		Server string `yaml:"server"`
	} `yaml:"cluster"`
}

type User struct {
	Name string `yaml:"name"`
	User struct {
		CertData string `yaml:"client-certificate-data"`
		KeyData  string `yaml:"client-key-data"`
		Token    string `yaml:"token"`
	} `yaml:"user"`
}

// KubernetesYaml defines config structure
// kubectl config view --flatten
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

type MonitoredState struct {
	State      map[string]KubernetesResource
	Groups     map[string]transit.ResourceGroup
	Mismatched map[string]bool
}

func (connector *KubernetesConnector) Initialize(ctx context.Context) error {
	// kubeStateMetricsEndpoint := "http://" + config.EndPoint + "/api/v1/namespaces/kube-system/services/kube-state-metrics:http-metrics/proxy/metrics"

	// Note:  kubernetes.NewForConfigAndClient(kConfig, clients.HttpClient)
	// The global HttpClient won't work with BearerToken
	// and global HttpClientTransport cannot be used with the TLS client certificate.
	// So, using a new default httpClient and configuring both according to connector settings.
	clients.HttpClientTransport.TLSClientConfig.InsecureSkipVerify =
		clients.HttpClientTransport.TLSClientConfig.InsecureSkipVerify || connector.ExtConfig.Insecure

	kConfig := &rest.Config{
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
			Insecure: clients.HttpClientTransport.TLSClientConfig.InsecureSkipVerify,
		},
		UserAgent:          "",
		DisableCompression: false,
		Transport:          nil,
		WrapTransport:      nil,
		QPS:                0,
		Burst:              0,
		RateLimiter:        nil,
		Timeout:            clients.HttpClient.Timeout,
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

		if fConfig.Kind != "Config" || len(fConfig.Clusters) == 0 || len(fConfig.Users) == 0 ||
			fConfig.Clusters[0].Cluster.Server == "" {
			return errors.New("invalid configuration file")
		}

		kConfig.BearerToken = fConfig.Users[0].User.Token
		kConfig.KeyData = []byte(fConfig.Users[0].User.KeyData)
		kConfig.CertData = []byte(fConfig.Users[0].User.CertData)
		kConfig.CAData = []byte(fConfig.Clusters[0].Cluster.CAData)
		kConfig.Host = fConfig.Clusters[0].Cluster.Server

		log.Info().Msg("using YAML file auth")
	}

	// kClientSet, err := kubernetes.NewForConfigAndClient(kConfig, clients.HttpClient)
	kClientSet, err := kubernetes.NewForConfig(kConfig)
	if err != nil {
		return err
	}
	version, err := kClientSet.Discovery().ServerVersion()
	if err != nil {
		return err
	}

	// mClientSet, err := metricsApi.NewForConfigAndClient(kConfig, clients.HttpClient)
	mClientSet, err := metricsApi.NewForConfig(kConfig)
	if err != nil {
		return err
	}

	connector.kClientSet = kClientSet
	connector.kapi = kClientSet.CoreV1()
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

// Collect inventory and metrics for all kinds of Kubernetes resources.
// Sort resources into groups and return inventory of host resources and inventory of groups
func (connector *KubernetesConnector) Collect() (
	[]transit.InventoryResource,
	[]transit.MonitoredResource,
	[]transit.ResourceGroup,
	error) {
	// gather inventory and Metrics
	metricsPerContainer := true
	state := MonitoredState{
		State:      make(map[string]KubernetesResource),
		Groups:     make(map[string]transit.ResourceGroup),
		Mismatched: make(map[string]bool),
	}
	if err := connector.collectNodeInventory(&state); err != nil {
		return nil, nil, nil, err
	}
	if err := connector.collectPodInventory(&state, &metricsPerContainer); err != nil {
		return nil, nil, nil, err
	}
	_ = connector.collectNodeMetrics(&state)
	if metricsPerContainer {
		_ = connector.collectPodMetricsPerContainer(&state)
	} else {
		_ = connector.collectPodMetricsPerReplica(&state)
	}

	// convert to arrays as expected by TCG
	inventory := make([]transit.InventoryResource, 0, len(state.State))
	monitored := make([]transit.MonitoredResource, 0, len(state.State))
	hostGroups := make([]transit.ResourceGroup, 0, len(state.Groups))
	for _, resource := range state.State {
		// convert inventory
		services := make([]transit.InventoryService, 0, len(resource.Services))
		for _, service := range resource.Services {
			services = append(services, connectors.CreateInventoryService(service.Name, service.Owner))
		}
		slices.SortFunc(services, func(a, b transit.InventoryService) int { return cmp.Compare(a.Name, b.Name) })
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
	for _, group := range state.Groups {
		slices.SortFunc(group.Resources, func(a, b transit.ResourceRef) int { return cmp.Compare(a.Name, b.Name) })
		hostGroups = append(hostGroups, group)
	}
	/* sort inventory data structures to allow checksums */
	slices.SortFunc(hostGroups, func(a, b transit.ResourceGroup) int { return cmp.Compare(a.GroupName, b.GroupName) })
	slices.SortFunc(inventory, func(a, b transit.InventoryResource) int { return cmp.Compare(a.Name, b.Name) })

	return inventory, monitored, hostGroups, nil
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
func (connector *KubernetesConnector) collectNodeInventory(state *MonitoredState) error {
	// TODO: ListOptions can filter by label
	nodes, err := connector.kapi.Nodes().List(connector.ctx, metav1.ListOptions{})
	if err != nil {
		log.Err(err).Msg("could not collect nodes inventory")
		return err
	}

	for _, node := range nodes.Items {
		resourceName := node.Name
		labels := GetLabels(node)
		namespace := labels["namespace"]
		stateKey := fmt.Sprintf("node:%v:%v", namespace, node.Name)
		// apply mapping and skip unmatched entries
		if len(connector.ExtConfig.GWMapping.HostName) > 0 {
			var err error
			resourceName, err = connector.ExtConfig.GWMapping.HostName.ApplyOR(labels)
			if err != nil || resourceName == "" {
				log.Debug().Err(err).
					Interface("labels", labels).Interface("mappings", connector.ExtConfig.GWMapping.HostName).
					Msg("could not map hostname on node")

				state.Mismatched[stateKey] = true
				continue
			}
		}
		groupName, err := connector.ExtConfig.GWMapping.HostGroup.ApplyOR(labels)
		if err != nil || groupName == "" {
			groupName = "nodes-" + namespace
			log.Debug().Err(err).
				Interface("labels", labels).Interface("mappings", connector.ExtConfig.GWMapping.HostGroup).
				Msg("could not map hostgroup on node, adding to nodes-namespace group")
		}

		rf := transit.ResourceRef{Name: resourceName, Owner: groupName, Type: transit.ResourceTypeHost}
		if group, ok := state.Groups[groupName]; ok {
			group.Resources = append(group.Resources, rf)
			state.Groups[groupName] = group
		} else {
			state.Groups[groupName] = transit.ResourceGroup{
				GroupName: groupName,
				Resources: []transit.ResourceRef{rf},
				Type:      transit.HostGroup,
			}
		}

		monitorStatus, message := connector.calculateNodeStatus(&node)
		resource := KubernetesResource{
			Name:     resourceName,
			Type:     transit.ResourceTypeHost,
			Status:   monitorStatus,
			Message:  message,
			Labels:   labels,
			Services: make(map[string]transit.MonitoredService),
		}
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
			monitoredService, err := connectors.BuildServiceForMetric(resource.Name, metricBuilder)
			if err != nil {
				log.Err(err).Msgf("could not create service %s:%s", stateKey, customServiceName)
			}
			if monitoredService != nil {
				resource.Services[metricBuilder.Name] = *monitoredService
			}
		}
		state.State[stateKey] = resource
	}
	return nil
}

// Pod Inventory also retrieves status
// inventory also contains status, pod counts, capacity and allocation metrics
func (connector *KubernetesConnector) collectPodInventory(
	state *MonitoredState,
	metricsPerContainer *bool) error {
	// TODO: filter pods by namespace(s)
	pods, err := connector.kapi.Pods("").List(connector.ctx, metav1.ListOptions{})
	if err != nil {
		log.Err(err).Msg("could not collect pods inventory")
		return err
	}

	groupsMap := make(map[string]bool)
	addResource := func(
		stateKey string,
		resourceName string,
		monitorStatus transit.MonitorStatus,
		message string,
		labels map[string]string,
		groupName string,
	) {
		resource := KubernetesResource{
			Name:     resourceName,
			Type:     transit.ResourceTypeHost,
			Status:   monitorStatus,
			Message:  message,
			Labels:   labels,
			Services: make(map[string]transit.MonitoredService),
		}
		state.State[stateKey] = resource
		// no services to process at this stage

		if _, found := groupsMap[resource.Name]; found {
			return
		}
		groupsMap[resource.Name] = true
		rf := transit.ResourceRef{Name: resource.Name, Owner: groupName, Type: transit.ResourceTypeHost}
		if group, ok := state.Groups[groupName]; ok {
			group.Resources = append(group.Resources, rf)
			state.Groups[groupName] = group
		} else {
			state.Groups[groupName] = transit.ResourceGroup{
				GroupName: groupName,
				Resources: []transit.ResourceRef{rf},
				Type:      transit.HostGroup,
			}
		}
	}

	for _, pod := range pods.Items {
		monitorStatus, message := connector.calculatePodStatus(&pod)
		resourceName := pod.Name

		if *metricsPerContainer {
			for _, container := range pod.Spec.Containers {
				if ContainerPOD == strings.TrimSuffix(container.Name, "-") {
					continue
				}

				resourceName := container.Name
				labels := GetLabels(pod, container)
				namespace := labels["namespace"]
				stateKey := fmt.Sprintf("pod:%v:%v:%v", namespace, pod.Name, container.Name)
				// apply mapping and skip unmatched entries
				if len(connector.ExtConfig.GWMapping.HostName) > 0 {
					var err error
					resourceName, err = connector.ExtConfig.GWMapping.HostName.ApplyOR(labels)
					if err != nil || resourceName == "" {
						log.Debug().Err(err).
							Interface("labels", labels).Interface("mappings", connector.ExtConfig.GWMapping.HostName).
							Msg("could not map hostname on pod container")

						state.Mismatched[stateKey] = true
						continue
					}
				}
				groupName, err := connector.ExtConfig.GWMapping.HostGroup.ApplyOR(labels)
				if err != nil || groupName == "" {
					groupName = "pods-" + namespace
					log.Debug().Err(err).
						Interface("labels", labels).Interface("mappings", connector.ExtConfig.GWMapping.HostGroup).
						Msg("could not map hostgroup on pod container, adding to pods-namespace group")
				}

				addResource(stateKey,
					resourceName, monitorStatus, message, labels,
					groupName)
			}
		} else {
			labels := GetLabels(pod)
			namespace := labels["namespace"]
			stateKey := fmt.Sprintf("pod:%v:%v", namespace, pod.Name)
			// apply mapping and skip unmatched entries
			if len(connector.ExtConfig.GWMapping.HostName) > 0 {
				var err error
				resourceName, err = connector.ExtConfig.GWMapping.HostName.ApplyOR(labels)
				if err != nil || resourceName == "" {
					log.Debug().Err(err).
						Interface("labels", labels).Interface("mappings", connector.ExtConfig.GWMapping.HostName).
						Msg("could not map hostname on pod")

					state.Mismatched[stateKey] = true
					continue
				}
			}
			groupName, err := connector.ExtConfig.GWMapping.HostGroup.ApplyOR(labels)
			if err != nil || groupName == "" {
				groupName = "pods-" + namespace
				log.Debug().Err(err).
					Interface("labels", labels).Interface("mappings", connector.ExtConfig.GWMapping.HostGroup).
					Msg("could not map hostgroup on pod, adding to pods-namespace group")
			}

			addResource(stateKey,
				resourceName, monitorStatus, message, labels,
				groupName)
		}
	}
	return nil
}

func (connector *KubernetesConnector) collectNodeMetrics(state *MonitoredState) error {
	// TODO: filter by namespace
	nodes, err := connector.mapi.NodeMetricses().List(connector.ctx, metav1.ListOptions{})
	if err != nil {
		log.Err(err).Msg("could not collect nodes metrics")
		return err
	}

	for _, node := range nodes.Items {
		stateKey := fmt.Sprintf("node:%v:%v", GetLabels(node)["namespace"], node.Name)
		if resource, ok := state.State[stateKey]; ok {
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
				monitoredService, err := connectors.BuildServiceForMetric(resource.Name, metricBuilder)
				if err != nil {
					log.Err(err).Msgf("could not create service %v:%v", stateKey, customServiceName)
				}
				if monitoredService != nil {
					resource.Services[metricBuilder.Name] = *monitoredService
				}
			}
			state.State[stateKey] = resource
		} else if _, ok = state.Mismatched[stateKey]; !ok {
			log.Warn().Msgf("node not found in monitored state: %v", stateKey)
		}
	}
	return nil
}

func (connector *KubernetesConnector) collectPodMetricsPerReplica(state *MonitoredState) error {
	// TODO: filter by namespace
	pods, err := connector.mapi.PodMetricses("").List(connector.ctx, metav1.ListOptions{})
	if err != nil {
		log.Err(err).Msg("could not collect pods metrics")
		return err
	}

	for _, pod := range pods.Items {
		stateKey := fmt.Sprintf("pod:%v:%v", GetLabels(pod)["namespace"], pod.Name)
		if resource, ok := state.State[stateKey]; ok {
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
					monitoredService, err := connectors.BuildServiceForMultiMetric(resource.Name, metricDefinition.Name, metricDefinition.CustomName, metricBuilders)
					if err != nil {
						log.Err(err).Msgf("could not create service %v:%v", stateKey, metricDefinition.Name)
					}
					if monitoredService != nil {
						resource.Services[metricBuilder.Name] = *monitoredService
					}
				}
			}
			state.State[stateKey] = resource
		} else if _, ok = state.Mismatched[stateKey]; !ok {
			log.Warn().Msgf("pod not found in monitored state: %v", stateKey)
		}
	}
	return nil
}

// treat each container uniquely -- store multi-metrics per pod replica for each node
func (connector *KubernetesConnector) collectPodMetricsPerContainer(state *MonitoredState) error {
	// TODO: filter by namespace
	pods, err := connector.mapi.PodMetricses("").List(connector.ctx, metav1.ListOptions{})
	if err != nil {
		log.Err(err).Msg("could not collect pod metrics")
		return err
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

				stateKey := fmt.Sprintf("pod:%v:%v:%v", GetLabels(pod, container)["namespace"], pod.Name, container.Name)
				if resource, ok := state.State[stateKey]; ok {
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
					state.State[stateKey] = resource
				} else if _, ok = state.Mismatched[stateKey]; !ok {
					log.Debug().Msgf("pod container not found in monitored state: %v", stateKey)
					debugDetails = true
				}
			}
		}
	}
	if debugDetails {
		log.Debug().Interface("monitoredState", state).Send()
	}
	return nil
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

func GetLabels(a ...interface{}) map[string]string {
	labels := map[string]string{
		"cluster":   "default",
		"namespace": "default",
	}
	for _, v := range a {
		switch v := v.(type) {
		case v1.Container:
			labels["container_name"] = v.Name
		case v1beta1.ContainerMetrics:
			labels["container_name"] = v.Name
		case v1.Node:
			labels["node_name"] = v.Name
			for key, element := range v.GetLabels() {
				labels[key] = element
			}
			if ns := v.GetNamespace(); ns != "" {
				labels["namespace"] = ns
			}
		case v1beta1.NodeMetrics:
			labels["node_name"] = v.Name
			for key, element := range v.GetLabels() {
				labels[key] = element
			}
			if ns := v.GetNamespace(); ns != "" {
				labels["namespace"] = ns
			}
		case v1.Pod:
			labels["pod_name"] = v.Name
			for key, element := range v.GetLabels() {
				labels[key] = element
			}
			if ns := v.GetNamespace(); ns != "" {
				labels["namespace"] = ns
			}
		case v1beta1.PodMetrics:
			labels["pod_name"] = v.Name
			for key, element := range v.GetLabels() {
				labels[key] = element
			}
			if ns := v.GetNamespace(); ns != "" {
				labels["namespace"] = ns
			}
		}

		if value, ok := labels[ClusterNameLabel]; ok {
			labels["cluster"] = value
		}

		// TODO: looks better but won't work
		// if v, ok := v.(interface{ GetLabels() map[string]string }); ok {
		// 	for key, element := range v.GetLabels() {
		// 		labels[key] = element
		// 	}
		// 	if value, ok := labels[ClusterNameLabel]; ok {
		// 		labels["cluster"] = value
		// 	}
		// }
		// if v, ok := v.(interface{ GetNamespace() string }); ok {
		// 	if ns := v.GetNamespace(); ns != "" {
		// 		labels["namespace"] = ns
		// 	}
		// }
	}

	return labels
}
