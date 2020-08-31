package main

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/elastic-connector/clients"
	"github.com/gwos/tcg/transit"
	_ "github.com/stretchr/testify/assert"
)

func TestInitFullConfig(t *testing.T) {
	mockConfig()

	testMetric1View1 := transit.MetricDefinition{
		Name:              "metric1",
		CustomName:        "custom-metric1",
		WarningThreshold:  -1,
		CriticalThreshold: -1,
		ServiceType:       "view1",
		Monitored:         true,
		Graphed:           false,
	}
	testMetric2View1 := transit.MetricDefinition{
		Name:              "metric2",
		ServiceType:       "view1",
		WarningThreshold:  10,
		CriticalThreshold: 20,
		Monitored:         true,
	}
	testMetric3View2 := transit.MetricDefinition{
		Name:        "metric3",
		ServiceType: "view2",
		Monitored:   true,
		Graphed:     false,
	}

	expectedViews := make(map[string]map[string]transit.MetricDefinition)
	expectedView1 := make(map[string]transit.MetricDefinition)
	expectedView2 := make(map[string]transit.MetricDefinition)
	expectedView1["metric1"] = testMetric1View1
	expectedView1["metric2"] = testMetric2View1
	expectedView2["metric3"] = testMetric3View2
	expectedViews["view1"] = expectedView1
	expectedViews["view2"] = expectedView2

	expected := ExtConfig{
		Kibana: Kibana{
			ServerName: "http://eccTestKibanaServer/",
			Username:   "eccTestKibanaUser",
			Password:   "eccTestKibanaPass",
		},
		CustomTimeFilter: clients.KTimeFilter{
			From: "now-300s",
			To:   "now",
		},
		OverrideTimeFilter: true,
		HostNameField:      "eccTestHostNameLabel",
		HostGroupField:     "eccTestHostGroupLabel",
		GroupNameByUser:    false,
		Timer:              time.Duration(5) * time.Minute,
		Ownership:          transit.Yield,
		AppType:            "testAppType",
		AgentID:            "testAgentId",
		Servers:            []string{"http://eccTestServer1", "http://eccTestServer2"},
		Views:              expectedViews,
	}

	data := []byte(`{
	  "agentId": "testAgentId",
	  "appType": "testAppType",
	  "monitorConnection": {
	    "server": "eccTestServer1,eccTestServer2",
	    "extensions": {
	      "checkIntervalMinutes": 5,
	      "checkTimeoutSeconds": 10,
	      "retryConnection": 10,
	      "kibana": {
	        "serverName": "eccTestKibanaServer",
	        "userName": "eccTestKibanaUser",
	        "password": "eccTestKibanaPass"
	      },
	      "timefilter": {
	        "from": "now-$interval",
	        "to": "now",
	        "override": true
	      },
	      "hostNameLabelPath": "eccTestHostNameLabel",
	      "hostGroupNameByUser": false,
	      "hostGroupLabelPath": "eccTestHostGroupLabel"
	    }
	  },
	  "metricsProfile": {
	    "name": "test-profile-1",
	    "profileType": "test",
	    "metrics": [
	      {
	        "warningThreshold": -1,
	        "criticalThreshold": -1,
	        "name": "metric1",
	        "customName": "custom-metric1",
	        "monitored": true,
	        "graphed": false,
	        "serviceType": "view1"
	      },
	      {
	        "name": "metric3",
	        "monitored": true,
	        "graphed": false,
	        "serviceType": "view2"
	      },
	      {
	        "warningThreshold": 10,
	        "criticalThreshold": 20,
	        "name": "metric2",
	        "customName": "",
	        "monitored": true,
	        "graphed": false,
	        "serviceType": "view1"
	      },
	      {
	        "warningThreshold": -1,
	        "criticalThreshold": -1,
	        "name": "metric4",
	        "customName": "custom-metric4",
	        "monitored": false,
	        "graphed": false,
	        "serviceType": "view2"
	      }
	    ]
	  }
	}`)

	cfgChksum, _ = connectors.Hashsum(expected)
	config.GetConfig().LoadConnectorDTO(data)
	configHandler(data)

	if !reflect.DeepEqual(*extConfig, expected) {
		t.Errorf("ExtConfig actual:\n%v\nexpected:\n%v", *extConfig, expected)
	}
}

func TestInitConfigWithNotPresentedValues(t *testing.T) {
	mockConfig()

	expected := ExtConfig{
		Kibana: Kibana{
			ServerName: defaultKibanaServerName,
			Username:   defaultKibanaUsername,
			Password:   defaultKibanaPassword,
		},
		CustomTimeFilter: clients.KTimeFilter{
			From: "now-120s",
			To:   defaultTimeFilterTo,
		},
		OverrideTimeFilter: defaultAlwaysOverrideTimeFilter,
		HostNameField:      defaultHostNameLabel,
		HostGroupField:     defaultHostGroupLabel,
		GroupNameByUser:    defaultGroupNameByUser,
		Timer:              connectors.DefaultTimer,
		Ownership:          transit.Yield,
		AppType:            "",
		AgentID:            "",
		Servers:            []string{defaultElasticServer},
		Views:              map[string]map[string]transit.MetricDefinition{},
	}

	data := []byte(`{}`)

	cfgChksum, _ = connectors.Hashsum(expected)
	config.GetConfig().LoadConnectorDTO(data)
	configHandler(data)

	if !reflect.DeepEqual(*extConfig, expected) {
		t.Errorf("ExtConfig actual:\n%v\nexpected:\n%v", *extConfig, expected)
	}
}

func TestInitConfigWithPartialPresentedValues(t *testing.T) {
	mockConfig()

	expected := ExtConfig{
		Kibana: Kibana{
			ServerName: defaultKibanaServerName,
			Username:   defaultKibanaUsername,
			Password:   defaultKibanaPassword,
		},
		CustomTimeFilter: clients.KTimeFilter{
			From: "now-120s",
			To:   defaultTimeFilterTo,
		},
		OverrideTimeFilter: defaultAlwaysOverrideTimeFilter,
		HostNameField:      defaultHostNameLabel,
		HostGroupField:     defaultHostGroupName,
		GroupNameByUser:    true,
		Timer:              connectors.DefaultTimer,
		Ownership:          transit.Yield,
		AppType:            "testAppType",
		AgentID:            "testAgentId",
		Servers:            []string{defaultElasticServer},
		Views:              map[string]map[string]transit.MetricDefinition{},
	}

	data := []byte(`{
		"agentId": "testAgentId",
		"appType": "testAppType",
		"monitorConnection": {
			"extensions": {"hostGroupNameByUser":true}
		}
	}`)

	cfgChksum, _ = connectors.Hashsum(expected)
	config.GetConfig().LoadConnectorDTO(data)
	configHandler(data)

	if !reflect.DeepEqual(*extConfig, expected) {
		t.Errorf("ExtConfig actual:\n%v\nexpected:\n%v", *extConfig, expected)
	}
}

func TestHandleEmptyConfig(t *testing.T) {
	mockConfig()

	expected := ExtConfig{
		Kibana: Kibana{
			ServerName: defaultKibanaServerName,
			Username:   defaultKibanaUsername,
			Password:   defaultKibanaPassword,
		},
		CustomTimeFilter: clients.KTimeFilter{
			From: "now-120s",
			To:   defaultTimeFilterTo,
		},
		OverrideTimeFilter: defaultAlwaysOverrideTimeFilter,
		HostNameField:      defaultHostNameLabel,
		HostGroupField:     defaultHostGroupLabel,
		GroupNameByUser:    defaultGroupNameByUser,
		Timer:              connectors.DefaultTimer,
		Ownership:          transit.Yield,
		AppType:            "testAppType",
		AgentID:            "testAgentId",
		Servers:            []string{defaultElasticServer},
		Views:              map[string]map[string]transit.MetricDefinition{},
	}

	data := []byte(`{
		"agentId": "testAgentId",
		"appType": "testAppType",
		"monitorConnection": {
			"extensions": {}
		}
	}`)

	cfgChksum, _ = connectors.Hashsum(expected)
	config.GetConfig().LoadConnectorDTO(data)
	configHandler(data)

	if !reflect.DeepEqual(*extConfig, expected) {
		t.Errorf("ExtConfig actual:\n%v\nexpected:\n%v", *extConfig, expected)
	}
}

func mockConfig() {
	tmpFile, _ := ioutil.TempFile("", "config")
	_ = os.Setenv(string(config.ConfigEnv), tmpFile.Name())
	defer os.Remove(tmpFile.Name())
}
