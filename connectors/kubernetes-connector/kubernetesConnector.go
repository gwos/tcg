package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/transit"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsApi "k8s.io/metrics/pkg/client/clientset/versioned"
	mv1 "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
	"strings"
	"time"
)

type KubernetesView string
const (
	ViewNodes KubernetesView = "Nodes"
    ViewPods  KubernetesView = "Pods"
)

const (
	ClusterHostGroup = "KubernetesCluster" // TODO: add actual clustername
)

type KubernetesConnector struct {
	config     KubernetesConnectorConfig
	kapi       kv1.CoreV1Interface
	kClientSet kubernetes.Interface
	mapi       mv1.MetricsV1beta1Interface
	ctx        context.Context
}

type KubernetesResource struct {
	Name         string
	Type         transit.ResourceType
	Status       transit.MonitorStatus
	Message      string
	Labels       map[string]string
	Services     map[string]transit.MonitoredService
}

func (connector *KubernetesConnector) Initialize(config KubernetesConnectorConfig) error {
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
	fmt.Printf("initialized Kubernetes connection to server version %s, for client version, %s, and endPoint %s",
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
func (connector *KubernetesConnector) Collect(cfg *KubernetesConnectorConfig) ([]transit.InventoryResource, []transit.MonitoredResource, []transit.ResourceGroup) {

	// gather inventory and Metrics
	monitoredState := make(map[string]KubernetesResource)
	groups := make(map[string]transit.ResourceGroup)
	connector.collectNodeInventory(monitoredState, groups, cfg)
	connector.collectPodInventory(monitoredState, groups, cfg)
	connector.collectNodeMetrics(monitoredState, cfg)
	// TODO: collectPodMetrics

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
			services[serviceIndex] = transit.InventoryService{
				Name:        service.Name,
				Type:        service.Type,
				Owner:       service.Owner,
			}
			serviceIndex = serviceIndex + 1
		}
		inventory[index] = transit.InventoryResource{
			Name:        resource.Name,
			Type:        resource.Type,
			Services:    services,
		}
		// convert monitored state
		mServices := make([]transit.MonitoredService, len(resource.Services))
		serviceIndex = 0
		for _, service := range resource.Services {
			mServices		[serviceIndex] = service
			serviceIndex = serviceIndex + 1
		}
		monitored[index] = transit.MonitoredResource{
			Name:             resource.Name,
			Type:             resource.Type,
			Status:           resource.Status,
			LastCheckTime:    milliseconds.MillisecondTimestamp{time.Now()},
			NextCheckTime:    milliseconds.MillisecondTimestamp{time.Now()}, // TODO: interval
			LastPlugInOutput: resource.Message,
			Services:         mServices,
		}
		index = index + 1
		fmt.Println(resource.Name)
	}
	index = 0
	for _, group := range hostGroups {
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
func (connector *KubernetesConnector) collectNodeInventory(monitoredState map[string]KubernetesResource, groups map[string]transit.ResourceGroup, cfg *KubernetesConnectorConfig) {
	nodes, _ := connector.kapi.Nodes().List(connector.ctx, metav1.ListOptions{}) // TODO: ListOptions can filter by label
	groups[ClusterHostGroup] = transit.ResourceGroup{
		GroupName: ClusterHostGroup,
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
			Name:       node.Name,
			Type:       transit.Host,
			Status:     monitorStatus,
			Message:    message,
			Labels: 	labels,
			Services:   make(map[string]transit.MonitoredService),
		}
		monitoredState[resource.Name] = resource
		// process services
		for key, metricDefinition := range cfg.Views[ViewNodes] {
			var value int64 = 0
			switch key {
			case "cpu.cores":
				value = node.Status.Capacity.Cpu().Value()
			case "cpu.allocated":
				value = node.Status.Allocatable.Cpu().Value()
			case "memory.capacity":
				value = node.Status.Capacity.Memory().Value()
			case "memory.allocated":
				value = node.Status.Allocatable.Memory().Value()
			case "pods":
				value = node.Status.Capacity.Pods().Value()
			default:
				continue
			}
			metricBuilder := connectors.MetricBuilder{
				Name:       key,
				CustomName: metricDefinition.CustomName,
				Value:      value,
				UnitType:   transit.UnitCounter,
				Warning:    metricDefinition.WarningThreshold,
				Critical:   metricDefinition.CriticalThreshold,
			}
			customServiceName := connectors.Name(metricBuilder.Name, metricDefinition.CustomName)
			monitoredService, err := connectors.BuildServiceForMetric(node.Name, metricBuilder)
			if err != nil {
				log.Error("Error when creating service ", node.Name, ":", customServiceName)
				log.Error(err)
			}
			if monitoredService != nil {
				resource.Services[metricBuilder.Name] = *monitoredService
			}

		}
		// add to default Cluster group
		groups[ClusterHostGroup].Resources[index] = transit.MonitoredResourceRef{
			Name: resource.Name,
			Type: transit.Host,
		}
		index = index + 1
	}
}

// Pod Inventory also retrieves status
// inventory also contains status, pod counts, capacity and allocation metrics
func (connector *KubernetesConnector) collectPodInventory(monitoredState map[string]KubernetesResource, groups map[string]transit.ResourceGroup, cfg *KubernetesConnectorConfig) {
	// TODO: filter pods by namespace(s)
	pods, err := connector.kapi.Pods("").List(connector.ctx, metav1.ListOptions{})
	if err != nil {
		// TODO:
	}
	for _, pod := range pods.Items {
		labels := make(map[string]transit.TypedValue)
		for key, element := range pod.Labels {
			labels[key] = transit.TypedValue{
				ValueType:   transit.StringType,
				StringValue: element,
			}
		}
		//resource := transit.InventoryResource{
		//	Name:       pod.Name,
		//	Type:       transit.Container,
		//	Properties: labels,
		//}
		// TODO: continue here
	}
}

func (connector *KubernetesConnector) collectNodeMetrics(monitoredState map[string]KubernetesResource, cfg *KubernetesConnectorConfig) {
	nodes, err := connector.mapi.NodeMetricses().List(connector.ctx, metav1.ListOptions{})
	if err != nil {
		// TODO:
	}
	for _, node := range nodes.Items {
		if resource, ok := monitoredState[node.Name]; ok {
			for key, metricDefinition := range cfg.Views[ViewNodes] {
				var value int64 = 0
				switch key {
				case "cpu":
					value = node.Usage.Cpu().Value()
				case "memory":
					value = node.Usage.Memory().Value()
				default:
					continue
				}
				metricBuilder := connectors.MetricBuilder{
					Name:       key,
					CustomName: metricDefinition.CustomName,
					Value:      value,
					UnitType:   transit.UnitCounter,
					Warning:    metricDefinition.WarningThreshold,
					Critical:   metricDefinition.CriticalThreshold,
				}
				// TODO: validate these times are correct
				metricBuilder.StartTimestamp = &milliseconds.MillisecondTimestamp{Time: node.Timestamp.Time}
				metricBuilder.EndTimestamp = &milliseconds.MillisecondTimestamp{Time: node.Timestamp.Time}
				customServiceName := connectors.Name(metricBuilder.Name, metricDefinition.CustomName)
				monitoredService, err := connectors.BuildServiceForMetric(node.Name, metricBuilder)
				if err != nil {
					log.Error("Error when creating service ", node.Name, ":", customServiceName)
					log.Error(err)
				}
				if monitoredService != nil {
					resource.Services[metricBuilder.Name] = *monitoredService
				}
			}
		} else {
			log.Error("Node not found in metrics collection: " + node.Name)
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
					status = transit.Warning
				}
			}
		}

	}
	if status == transit.HostUp {
		message.WriteString(upMessage)
	}
	return status, message.String()
}

func (connector *KubernetesConnector) calculatePodStatus(node v1beta1.NodeMetrics) transit.MonitorStatus {
	return transit.HostUp // TODO: calculate based on conditions
}
