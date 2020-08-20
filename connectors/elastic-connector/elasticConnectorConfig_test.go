package main

import (
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/elastic-connector/clients"
	"github.com/gwos/tcg/transit"
	"reflect"
	"testing"
)

func checkExpected(t *testing.T, actual *ElasticConnectorConfig, expected *ElasticConnectorConfig) {
	t.Helper()

	if !reflect.DeepEqual(actual.AppType, expected.AppType) {
		t.Errorf("Elastic Connector Config AppType actual: %v, expected: %v", actual.AppType, expected.AppType)
	}
	if !reflect.DeepEqual(actual.AgentId, expected.AgentId) {
		t.Errorf("Elastic Connector Config AgentId actual: %v, expected: %v", actual.AgentId, expected.AgentId)
	}
	if len(actual.Servers) != len(expected.Servers) {
		t.Errorf("Elastic Connector Config Servers actual: %v, expected: %v", len(actual.Servers), len(expected.Servers))
	}
	for i, server := range actual.Servers {
		if !reflect.DeepEqual(server, expected.Servers[i]) {
			t.Errorf("Elastic Connector Config Servers index: %v, actual: %v, expected: %v", i, server,
				expected.Servers[i])
		}
	}
	if !reflect.DeepEqual(actual.Kibana, expected.Kibana) {
		t.Errorf("Elastic Connector Config Kibana actual: %v, expected: %v", actual.Kibana, expected.Kibana)
	}
	if len(actual.Views) != len(expected.Views) {
		t.Errorf("Elastic Connector Config Views actual: %v, expected: %v", len(actual.Views), len(expected.Views))
	}
	for k, v := range actual.Views {
		if len(v) != len(expected.Views[k]) {
			t.Errorf("Elastic Connector Config Views key: %v, actual: %v, expected: %v", k, len(v), len(expected.Views[k]))
		}
		for kk, vv := range v {
			if !reflect.DeepEqual(vv, expected.Views[k][kk]) {
				t.Errorf("Elastic Connector Config Servers Views key: %v, metric definitions key: %v, actual: %v, expected: %v",
					k, kk, vv, expected.Views[k][kk])
			}
		}
	}
	if !reflect.DeepEqual(actual.CustomTimeFilter, expected.CustomTimeFilter) {
		t.Errorf("Elastic Connector Config CustomTimeFilter actual: %v, expected: %v", actual.CustomTimeFilter,
			expected.CustomTimeFilter)
	}
	if !reflect.DeepEqual(actual.OverrideTimeFilter, expected.OverrideTimeFilter) {
		t.Errorf("Elastic Connector Config OverrideTimeFilter actual: %v, expected: %v", actual.OverrideTimeFilter,
			expected.OverrideTimeFilter)
	}
	if !reflect.DeepEqual(actual.HostNameField, expected.HostNameField) {
		t.Errorf("Elastic Connector Config HostNameField actual: %v, expected: %v", actual.HostNameField,
			expected.HostNameField)
	}
	if !reflect.DeepEqual(actual.HostGroupField, expected.HostGroupField) {
		t.Errorf("Elastic Connector Config HostGroupField actual: %v, expected: %v", actual.HostGroupField,
			expected.HostGroupField)
	}
	if !reflect.DeepEqual(actual.GroupNameByUser, expected.GroupNameByUser) {
		t.Errorf("Elastic Connector Config GroupNameByUser actual: %v, expected: %v", actual.GroupNameByUser,
			expected.GroupNameByUser)
	}
	if !reflect.DeepEqual(actual.Timer, expected.Timer) {
		t.Errorf("Elastic Connector Config Timer actual: %v, expected: %v", actual.Timer, expected.Timer)
	}
	if !reflect.DeepEqual(actual.Ownership, expected.Ownership) {
		t.Errorf("Elastic Connector Config Ownership actual: %v, expected: %v", actual.Ownership, expected.Ownership)
	}
	if len(actual.GWConnections) != len(expected.GWConnections) {
		t.Errorf("Elastic Connector Config GWConnections got: %v, expected: %v", len(actual.GWConnections), len(expected.GWConnections))
	}
	for k, v := range actual.GWConnections {
		if !reflect.DeepEqual(v, expected.GWConnections[k]) {
			t.Errorf("Elastic Connector Config GWConnections key: %v, actual: %v, expected: %v", k, v, expected.GWConnections[k])
		}
	}
}

func TestInitFullConfig(t *testing.T) {
	testExtensions := make(map[string]interface{})

	kibanaExt := make(map[string]interface{})
	kibanaExt[extensionsKeyServerName] = "eccTestKibanaServer"
	kibanaExt[extensionsKeyUsername] = "eccTestKibanaUser"
	kibanaExt[extensionsKeyPassword] = "eccTestKibanaPass"
	testExtensions[extensionsKeyKibana] = kibanaExt

	timeFilterExt := make(map[string]interface{})
	timeFilterExt[extensionsKeyFrom] = "now-$interval"
	timeFilterExt[extensionsKeyTo] = "now"
	timeFilterExt[extensionsKeyOverride] = true
	testExtensions[extensionsKeyTimeFilter] = timeFilterExt

	testExtensions[extensionsKeyHostNameLabelPath] = "eccTestHostNameLabel"
	testExtensions[extensionsKeyHostGroupLabelPath] = "eccTestHostGroupLabel"
	testExtensions[extensionsKeyGroupNameByUser] = false

	testExtensions[connectors.ExtensionsKeyTimer] = float64(1)

	testMonitorConnection := transit.MonitorConnection{
		ID:          0,
		Server:      "eccTestServer1,eccTestServer2",
		UserName:    "eccTestUsername",
		Password:    "ecctestPassword",
		SslEnabled:  false,
		URL:         "",
		Views:       nil,
		Extensions:  testExtensions,
		ConnectorID: 0,
	}

	testMetric1View1 := transit.MetricDefinition{
		Name:        "metric1",
		ServiceType: "view1",
		Monitored:   true,
	}
	testMetric2View1 := transit.MetricDefinition{
		Name:        "metric2",
		ServiceType: "view1",
		Monitored:   true,
	}
	testMetric3View2 := transit.MetricDefinition{
		Name:        "metric3",
		ServiceType: "view2",
		Monitored:   true,
	}
	testMetricsProfile := transit.MetricsProfile{
		Name:        "",
		ProfileType: "",
		IsTemplate:  false,
		Metrics:     []transit.MetricDefinition{testMetric1View1, testMetric2View1, testMetric3View2},
	}
	testGWConnections := config.GWConnections{
		&config.GWConnection{Enabled: true, LocalConnection: false, HostName: "localhost:80", UserName: "RESTAPIACCESS", Password: "SEC RET"},
		&config.GWConnection{HostName: "localhost:3001"},
	}

	actual := InitConfig("testAppType", "testAgentId", &testMonitorConnection, &testMetricsProfile, testGWConnections)

	expectedViews := make(map[string]map[string]transit.MetricDefinition)
	expectedView1 := make(map[string]transit.MetricDefinition)
	expectedView2 := make(map[string]transit.MetricDefinition)
	expectedView1["metric1"] = testMetric1View1
	expectedView1["metric2"] = testMetric2View1
	expectedView2["metric3"] = testMetric3View2
	expectedViews["view1"] = expectedView1
	expectedViews["view2"] = expectedView2

	expected := ElasticConnectorConfig{
		AppType: "testAppType",
		AgentId: "testAgentId",
		Servers: []string{"http://eccTestServer1", "http://eccTestServer2"},
		Kibana: Kibana{
			ServerName: "http://eccTestKibanaServer/",
			Username:   "eccTestKibanaUser",
			Password:   "eccTestKibanaPass",
		},
		Views: expectedViews,
		CustomTimeFilter: clients.KTimeFilter{
			From: "now-60s",
			To:   "now",
		},
		OverrideTimeFilter: true,
		HostNameField:      "eccTestHostNameLabel",
		HostGroupField:     "eccTestHostGroupLabel",
		GroupNameByUser:    false,
		Timer:              60,
		GWConnections:      testGWConnections,
	}

	checkExpected(t, actual, &expected)
}

func TestInitConfigWithNotPresentedValues(t *testing.T) {
	expected := ElasticConnectorConfig{
		AppType: "testAppType",
		AgentId: "testAgentId",
		Servers: []string{defaultElasticServer},
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
	}
	actual := InitConfig("testAppType", "testAgentId", nil, nil, nil)

	checkExpected(t, actual, &expected)
}

func TestInitConfigWithPartialPresentedValues(t *testing.T) {
	testExtensions := make(map[string]interface{})
	// set only checkbox "User Defined Host Group Name" as selected
	testExtensions[extensionsKeyGroupNameByUser] = true
	testMonitorConnection := transit.MonitorConnection{
		Extensions: testExtensions,
	}

	expected := ElasticConnectorConfig{
		AppType: "testAppType",
		AgentId: "testAgentId",
		Servers: []string{defaultElasticServer},
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
	}

	actual := InitConfig("testAppType", "testAgentId", &testMonitorConnection, nil, nil)

	checkExpected(t, actual, &expected)

	// set checkbox "User Defined Host Group Name" as not selected
	testMonitorConnection.Extensions[extensionsKeyGroupNameByUser] = false
	expected.HostGroupField = defaultHostGroupLabel
	expected.GroupNameByUser = false

	actual = InitConfig("testAppType", "testAgentId", &testMonitorConnection, nil, nil)

	checkExpected(t, actual, &expected)
}

func TestInitConfigWithPresentedAsNilValues(t *testing.T) {
	testExtensions := make(map[string]interface{})
	testExtensions[extensionsKeyKibana] = nil
	testExtensions[extensionsKeyTimeFilter] = nil
	testExtensions[extensionsKeyHostNameLabelPath] = nil
	testExtensions[extensionsKeyHostGroupLabelPath] = nil
	testExtensions[extensionsKeyGroupNameByUser] = nil
	testExtensions[connectors.ExtensionsKeyTimer] = nil
	testMonitorConnection := transit.MonitorConnection{
		Extensions: testExtensions,
	}

	expected := ElasticConnectorConfig{
		AppType: "testAppType",
		AgentId: "testAgentId",
		Servers: []string{defaultElasticServer},
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
	}

	actual := InitConfig("testAppType", "testAgentId", &testMonitorConnection, nil, nil)

	checkExpected(t, actual, &expected)
}
