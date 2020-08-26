package main

import (
	"reflect"
	"testing"

	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/elastic-connector/clients"
	"github.com/gwos/tcg/transit"
	_ "github.com/stretchr/testify/assert"
)

// func TestInitFullConfig(t *testing.T) {
// 	testExtensions := make(map[string]interface{})

// 	kibanaExt := make(map[string]interface{})
// 	kibanaExt[extensionsKeyServerName] = "eccTestKibanaServer"
// 	kibanaExt[extensionsKeyUsername] = "eccTestKibanaUser"
// 	kibanaExt[extensionsKeyPassword] = "eccTestKibanaPass"
// 	testExtensions[extensionsKeyKibana] = kibanaExt

// 	timeFilterExt := make(map[string]interface{})
// 	timeFilterExt[extensionsKeyFrom] = "now-$interval"
// 	timeFilterExt[extensionsKeyTo] = "now"
// 	timeFilterExt[extensionsKeyOverride] = true
// 	testExtensions[extensionsKeyTimeFilter] = timeFilterExt

// 	testExtensions[extensionsKeyHostNameLabelPath] = "eccTestHostNameLabel"
// 	testExtensions[extensionsKeyHostGroupLabelPath] = "eccTestHostGroupLabel"
// 	testExtensions[extensionsKeyGroupNameByUser] = false

// 	testExtensions[connectors.ExtensionsKeyTimer] = float64(1)

// 	testMonitorConnection := transit.MonitorConnection{
// 		ID:          0,
// 		Server:      "eccTestServer1,eccTestServer2",
// 		UserName:    "eccTestUsername",
// 		Password:    "ecctestPassword",
// 		SslEnabled:  false,
// 		URL:         "",
// 		Views:       nil,
// 		Extensions:  testExtensions,
// 		ConnectorID: 0,
// 	}

// 	testMetric1View1 := transit.MetricDefinition{
// 		Name:        "metric1",
// 		ServiceType: "view1",
// 		Monitored:   true,
// 	}
// 	testMetric2View1 := transit.MetricDefinition{
// 		Name:        "metric2",
// 		ServiceType: "view1",
// 		Monitored:   true,
// 	}
// 	testMetric3View2 := transit.MetricDefinition{
// 		Name:        "metric3",
// 		ServiceType: "view2",
// 		Monitored:   true,
// 	}
// 	testMetricsProfile := transit.MetricsProfile{
// 		Name:        "",
// 		ProfileType: "",
// 		IsTemplate:  false,
// 		Metrics:     []transit.MetricDefinition{testMetric1View1, testMetric2View1, testMetric3View2},
// 	}
// 	testGWConnections := config.GWConnections{
// 		&config.GWConnection{Enabled: true, LocalConnection: false, HostName: "localhost:80", UserName: "RESTAPIACCESS", Password: "SEC RET"},
// 		&config.GWConnection{HostName: "localhost:3001"},
// 	}

// 	actual := InitConfig("testAppType", "testAgentId", &testMonitorConnection, &testMetricsProfile, testGWConnections)

// 	expectedViews := make(map[string]map[string]transit.MetricDefinition)
// 	expectedView1 := make(map[string]transit.MetricDefinition)
// 	expectedView2 := make(map[string]transit.MetricDefinition)
// 	expectedView1["metric1"] = testMetric1View1
// 	expectedView1["metric2"] = testMetric2View1
// 	expectedView2["metric3"] = testMetric3View2
// 	expectedViews["view1"] = expectedView1
// 	expectedViews["view2"] = expectedView2

// 	expected := ExtConfig{
// 		AppType: "testAppType",
// 		AgentID: "testAgentId",
// 		Servers: []string{"http://eccTestServer1", "http://eccTestServer2"},
// 		Kibana: Kibana{
// 			ServerName: "http://eccTestKibanaServer/",
// 			Username:   "eccTestKibanaUser",
// 			Password:   "eccTestKibanaPass",
// 		},
// 		Views: expectedViews,
// 		CustomTimeFilter: clients.KTimeFilter{
// 			From: "now-60s",
// 			To:   "now",
// 		},
// 		OverrideTimeFilter: true,
// 		HostNameField:      "eccTestHostNameLabel",
// 		HostGroupField:     "eccTestHostGroupLabel",
// 		GroupNameByUser:    false,
// 		Timer:              60,
// 		GWConnections:      testGWConnections,
// 	}

// 	checkExpected(t, actual, &expected)
// }

// func TestInitConfigWithNotPresentedValues(t *testing.T) {
// 	expected := ExtConfig{
// 		AppType: "testAppType",
// 		AgentID: "testAgentId",
// 		Servers: []string{defaultElasticServer},
// 		Kibana: Kibana{
// 			ServerName: defaultKibanaServerName,
// 			Username:   defaultKibanaUsername,
// 			Password:   defaultKibanaPassword,
// 		},
// 		CustomTimeFilter: clients.KTimeFilter{
// 			From: "now-120s",
// 			To:   defaultTimeFilterTo,
// 		},
// 		OverrideTimeFilter: defaultAlwaysOverrideTimeFilter,
// 		HostNameField:      defaultHostNameLabel,
// 		HostGroupField:     defaultHostGroupLabel,
// 		GroupNameByUser:    defaultGroupNameByUser,
// 		Timer:              connectors.DefaultTimer,
// 		Ownership:          transit.Yield,
// 	}
// 	actual := InitConfig("testAppType", "testAgentId", nil, nil, nil)

// 	checkExpected(t, actual, &expected)
// }

// func TestInitConfigWithPartialPresentedValues(t *testing.T) {
// 	testExtensions := make(map[string]interface{})
// 	// set only checkbox "User Defined Host Group Name" as selected
// 	testExtensions[extensionsKeyGroupNameByUser] = true
// 	testMonitorConnection := transit.MonitorConnection{
// 		Extensions: testExtensions,
// 	}

// 	expected := ExtConfig{
// 		AppType: "testAppType",
// 		AgentID: "testAgentId",
// 		Servers: []string{defaultElasticServer},
// 		Kibana: Kibana{
// 			ServerName: defaultKibanaServerName,
// 			Username:   defaultKibanaUsername,
// 			Password:   defaultKibanaPassword,
// 		},
// 		CustomTimeFilter: clients.KTimeFilter{
// 			From: "now-120s",
// 			To:   defaultTimeFilterTo,
// 		},
// 		OverrideTimeFilter: defaultAlwaysOverrideTimeFilter,
// 		HostNameField:      defaultHostNameLabel,
// 		HostGroupField:     defaultHostGroupName,
// 		GroupNameByUser:    true,
// 		Timer:              connectors.DefaultTimer,
// 		Ownership:          transit.Yield,
// 	}

// 	actual := InitConfig("testAppType", "testAgentId", &testMonitorConnection, nil, nil)

// 	checkExpected(t, actual, &expected)

// 	// set checkbox "User Defined Host Group Name" as not selected
// 	testMonitorConnection.Extensions[extensionsKeyGroupNameByUser] = false
// 	expected.HostGroupField = defaultHostGroupLabel
// 	expected.GroupNameByUser = false

// 	actual = InitConfig("testAppType", "testAgentId", &testMonitorConnection, nil, nil)

// 	checkExpected(t, actual, &expected)
// }

func TestHandleEmptyConfig(t *testing.T) {
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
