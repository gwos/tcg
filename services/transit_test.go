package services

import (
	"encoding/json"
	"testing"

	"github.com/gwos/tcg/sdk/transit"
	"github.com/stretchr/testify/assert"
)

func Test_SyncExt(t *testing.T) {
	jsonInventoryExt := `{
        "groups": [
            {
                "groupName": "atest-hst",
                "resources": [
                    { "name": "atest22", "type": "host" },
                    { "name": "atest11", "type": "host" }
                ],
                "type": "HostGroup"
            },
            {
                "groupName": "atest-svc",
                "resources": [
                    { "name": "icmp_ping_alive", "owner": "atest22", "type": "service" },
                    { "name": "icmp_ping_alive", "owner": "atest11", "type": "service" }
                ],
                "type": "ServiceGroup"
            }
        ],
        "resources": [
            {
                "description": "atest11",
                "device": "127.1.1.11",
                "name": "atest11",
                "properties": {
                    "Alias": { "stringValue": "atest11", "valueType": "StringType" },
                    "CheckType": { "stringValue": "ACTIVE", "valueType": "StringType" },
                    "CurrentAttempt": { "integerValue": 1, "valueType": "IntegerType" },
                    "CurrentNotificationNumber": { "integerValue": 0, "valueType": "IntegerType" },
                    "ExecutionTime": { "doubleValue": 0.003, "valueType": "DoubleType" },
                    "LastCheckTime": { "timeValue": "1748333287000", "valueType": "TimeType" },
                    "LastNotificationTime": { "timeValue": "0", "valueType": "TimeType" },
                    "LastPluginOutput": {
                        "stringValue": "OK - 127.1.1.11 rta 0.044ms lost 0%",
                        "valueType": "StringType"
                    },
                    "LastStateChange": { "timeValue": "1747683022000", "valueType": "TimeType" },
                    "Latency": { "doubleValue": 0, "valueType": "DoubleType" },
                    "MaxAttempts": { "integerValue": 3, "valueType": "IntegerType" },
                    "MonitorStatus": { "stringValue": "HOST_UP", "valueType": "StringType" },
                    "NextCheckTime": { "timeValue": "1748333347000", "valueType": "TimeType" },
                    "PercentStateChange": { "doubleValue": 0, "valueType": "DoubleType" },
                    "PerformanceData": {
                        "stringValue": "rta=0.044ms;3000.000;5000.000;0; pl=0%;80;100;0;100 rtmax=0.044ms;;;; rtmin=0.044ms;;;;",
                        "valueType": "StringType"
                    },
                    "ScheduledDowntimeDepth": { "integerValue": 0, "valueType": "IntegerType" },
                    "StateType": { "stringValue": "HARD", "valueType": "StringType" },
                    "isAcknowledged": { "boolValue": false, "valueType": "BooleanType" },
                    "isChecksEnabled": { "boolValue": true, "valueType": "BooleanType" },
                    "isEventHandlersEnabled": { "boolValue": true, "valueType": "BooleanType" },
                    "isFlapDetectionEnabled": { "boolValue": true, "valueType": "BooleanType" },
                    "isNotificationsEnabled": { "boolValue": true, "valueType": "BooleanType" },
                    "isPassiveChecksEnabled": { "boolValue": true, "valueType": "BooleanType" }
                },
                "services": [
                    {
                        "name": "icmp_ping_alive",
                        "properties": {
                            "CheckType": { "stringValue": "ACTIVE", "valueType": "StringType" },
                            "CurrentAttempt": { "integerValue": 1, "valueType": "IntegerType" },
                            "CurrentNotificationNumber": { "integerValue": 0, "valueType": "IntegerType" },
                            "ExecutionTime": { "doubleValue": 0.004, "valueType": "DoubleType" },
                            "LastCheckTime": { "timeValue": "1748333287000", "valueType": "TimeType" },
                            "LastNotificationTime": { "timeValue": "0", "valueType": "TimeType" },
                            "LastPluginOutput": {
                                "stringValue": "OK - 127.1.1.11 rta 0.030ms lost 0%",
                                "valueType": "StringType"
                            },
                            "Latency": { "doubleValue": 0, "valueType": "DoubleType" },
                            "MaxAttempts": { "integerValue": 3, "valueType": "IntegerType" },
                            "MonitorStatus": { "stringValue": "SERVICE_OK", "valueType": "StringType" },
                            "NextCheckTime": { "timeValue": "1748333347000", "valueType": "TimeType" },
                            "PercentStateChange": { "doubleValue": 0, "valueType": "DoubleType" },
                            "PerformanceData": {
                                "stringValue": "rta=0.030ms;3000.000;5000.000;0; pl=0%;80;100;0;100 rtmax=0.030ms;;;; rtmin=0.030ms;;;;",
                                "valueType": "StringType"
                            },
                            "ScheduledDowntimeDepth": { "integerValue": 0, "valueType": "IntegerType" },
                            "StateType": { "stringValue": "HARD", "valueType": "StringType" },
                            "isAcceptPassiveChecks": { "boolValue": true, "valueType": "BooleanType" },
                            "isChecksEnabled": { "boolValue": true, "valueType": "BooleanType" },
                            "isEventHandlersEnabled": { "boolValue": true, "valueType": "BooleanType" },
                            "isFlapDetectionEnabled": { "boolValue": true, "valueType": "BooleanType" },
                            "isNotificationsEnabled": { "boolValue": true, "valueType": "BooleanType" },
                            "isProblemAcknowledged": { "boolValue": false, "valueType": "BooleanType" }
                        },
                        "type": "service"
                    }
                ],
                "type": "host"
            },
            {
                "description": "atest22",
                "device": "127.1.1.22",
                "name": "atest22",
                "properties": {
                    "Alias": { "stringValue": "atest22", "valueType": "StringType" },
                    "CheckType": { "stringValue": "ACTIVE", "valueType": "StringType" },
                    "CurrentAttempt": { "integerValue": 1, "valueType": "IntegerType" },
                    "CurrentNotificationNumber": { "integerValue": 0, "valueType": "IntegerType" },
                    "ExecutionTime": { "doubleValue": 0.003, "valueType": "DoubleType" },
                    "LastCheckTime": { "timeValue": "1748333257000", "valueType": "TimeType" },
                    "LastNotificationTime": { "timeValue": "0", "valueType": "TimeType" },
                    "LastPluginOutput": {
                        "stringValue": "OK - 127.1.1.22 rta 0.048ms lost 0%",
                        "valueType": "StringType"
                    },
                    "LastStateChange": { "timeValue": "1747683049000", "valueType": "TimeType" },
                    "Latency": { "doubleValue": 0.001, "valueType": "DoubleType" },
                    "MaxAttempts": { "integerValue": 3, "valueType": "IntegerType" },
                    "MonitorStatus": { "stringValue": "HOST_UP", "valueType": "StringType" },
                    "NextCheckTime": { "timeValue": "1748333317000", "valueType": "TimeType" },
                    "PercentStateChange": { "doubleValue": 0, "valueType": "DoubleType" },
                    "PerformanceData": {
                        "stringValue": "rta=0.048ms;3000.000;5000.000;0; pl=0%;80;100;0;100 rtmax=0.048ms;;;; rtmin=0.048ms;;;;",
                        "valueType": "StringType"
                    },
                    "ScheduledDowntimeDepth": { "integerValue": 0, "valueType": "IntegerType" },
                    "StateType": { "stringValue": "HARD", "valueType": "StringType" },
                    "isAcknowledged": { "boolValue": false, "valueType": "BooleanType" },
                    "isChecksEnabled": { "boolValue": true, "valueType": "BooleanType" },
                    "isEventHandlersEnabled": { "boolValue": true, "valueType": "BooleanType" },
                    "isFlapDetectionEnabled": { "boolValue": true, "valueType": "BooleanType" },
                    "isNotificationsEnabled": { "boolValue": true, "valueType": "BooleanType" },
                    "isPassiveChecksEnabled": { "boolValue": true, "valueType": "BooleanType" }
                },
                "services": [
                    {
                        "name": "icmp_ping_alive",
                        "properties": {
                            "CheckType": { "stringValue": "ACTIVE", "valueType": "StringType" },
                            "CurrentAttempt": { "integerValue": 1, "valueType": "IntegerType" },
                            "CurrentNotificationNumber": { "integerValue": 0, "valueType": "IntegerType" },
                            "ExecutionTime": { "doubleValue": 0.004, "valueType": "DoubleType" },
                            "LastCheckTime": { "timeValue": "1748333259000", "valueType": "TimeType" },
                            "LastNotificationTime": { "timeValue": "0", "valueType": "TimeType" },
                            "LastPluginOutput": {
                                "stringValue": "OK - 127.1.1.22 rta 0.033ms lost 0%",
                                "valueType": "StringType"
                            },
                            "Latency": { "doubleValue": 0, "valueType": "DoubleType" },
                            "MaxAttempts": { "integerValue": 3, "valueType": "IntegerType" },
                            "MonitorStatus": { "stringValue": "SERVICE_OK", "valueType": "StringType" },
                            "NextCheckTime": { "timeValue": "1748333319000", "valueType": "TimeType" },
                            "PercentStateChange": { "doubleValue": 0, "valueType": "DoubleType" },
                            "PerformanceData": {
                                "stringValue": "rta=0.033ms;3000.000;5000.000;0; pl=0%;80;100;0;100 rtmax=0.033ms;;;; rtmin=0.033ms;;;;",
                                "valueType": "StringType"
                            },
                            "ScheduledDowntimeDepth": { "integerValue": 0, "valueType": "IntegerType" },
                            "StateType": { "stringValue": "HARD", "valueType": "StringType" },
                            "isAcceptPassiveChecks": { "boolValue": true, "valueType": "BooleanType" },
                            "isChecksEnabled": { "boolValue": true, "valueType": "BooleanType" },
                            "isEventHandlersEnabled": { "boolValue": true, "valueType": "BooleanType" },
                            "isFlapDetectionEnabled": { "boolValue": true, "valueType": "BooleanType" },
                            "isNotificationsEnabled": { "boolValue": true, "valueType": "BooleanType" },
                            "isProblemAcknowledged": { "boolValue": false, "valueType": "BooleanType" }
                        },
                        "type": "service"
                    }
                ],
                "type": "host"
            }
        ]
    }`

	jsonInventory := `{
        "groups": [
            {
                "groupName": "atest-hst",
                "resources": [
                    { "name": "atest22", "type": "host" },
                    { "name": "atest11", "type": "host" }
                ],
                "type": "HostGroup"
            },
            {
                "groupName": "atest-svc",
                "resources": [
                    { "name": "icmp_ping_alive", "owner": "atest22", "type": "service" },
                    { "name": "icmp_ping_alive", "owner": "atest11", "type": "service" }
                ],
                "type": "ServiceGroup"
            }
        ],
        "resources": [
            {
                "description": "atest11",
                "device": "127.1.1.11",
                "name": "atest11",
                "properties": { "Alias": { "stringValue": "atest11", "valueType": "StringType" } },
                "services": [{ "name": "icmp_ping_alive", "type": "service" }],
                "type": "host"
            },
            {
                "description": "atest22",
                "device": "127.1.1.22",
                "name": "atest22",
                "properties": { "Alias": { "stringValue": "atest22", "valueType": "StringType" } },
                "services": [{ "name": "icmp_ping_alive", "type": "service" }],
                "type": "host"
            }
        ]
    }`

	jsonMonitoring := `{
        "resources": [
            {
                "description": "atest11",
                "device": "127.1.1.11",
                "lastCheckTime": "1748333287000",
                "lastPluginOutput": "OK - 127.1.1.11 rta 0.044ms lost 0%",
                "name": "atest11",
                "nextCheckTime": "1748333347000",
                "properties": {
                    "Alias": { "stringValue": "atest11", "valueType": "StringType" },
                    "CheckType": { "stringValue": "ACTIVE", "valueType": "StringType" },
                    "CurrentAttempt": { "integerValue": 1, "valueType": "IntegerType" },
                    "CurrentNotificationNumber": { "integerValue": 0, "valueType": "IntegerType" },
                    "ExecutionTime": { "doubleValue": 0.003, "valueType": "DoubleType" },
                    "LastNotificationTime": { "timeValue": "0", "valueType": "TimeType" },
                    "LastStateChange": { "timeValue": "1747683022000", "valueType": "TimeType" },
                    "Latency": { "doubleValue": 0, "valueType": "DoubleType" },
                    "MaxAttempts": { "integerValue": 3, "valueType": "IntegerType" },
                    "PercentStateChange": { "doubleValue": 0, "valueType": "DoubleType" },
                    "PerformanceData": {
                        "stringValue": "rta=0.044ms;3000.000;5000.000;0; pl=0%;80;100;0;100 rtmax=0.044ms;;;; rtmin=0.044ms;;;;",
                        "valueType": "StringType"
                    },
                    "StateType": { "stringValue": "HARD", "valueType": "StringType" },
                    "isAcknowledged": { "boolValue": false, "valueType": "BooleanType" },
                    "isChecksEnabled": { "boolValue": true, "valueType": "BooleanType" },
                    "isEventHandlersEnabled": { "boolValue": true, "valueType": "BooleanType" },
                    "isFlapDetectionEnabled": { "boolValue": true, "valueType": "BooleanType" },
                    "isNotificationsEnabled": { "boolValue": true, "valueType": "BooleanType" },
                    "isPassiveChecksEnabled": { "boolValue": true, "valueType": "BooleanType" }
                },
                "services": [
                    {
                        "lastCheckTime": "1748333287000",
                        "lastPluginOutput": "OK - 127.1.1.11 rta 0.030ms lost 0%",
                        "name": "icmp_ping_alive",
                        "nextCheckTime": "1748333347000",
                        "properties": {
                            "CheckType": { "stringValue": "ACTIVE", "valueType": "StringType" },
                            "CurrentAttempt": { "integerValue": 1, "valueType": "IntegerType" },
                            "CurrentNotificationNumber": { "integerValue": 0, "valueType": "IntegerType" },
                            "ExecutionTime": { "doubleValue": 0.004, "valueType": "DoubleType" },
                            "LastNotificationTime": { "timeValue": "0", "valueType": "TimeType" },
                            "Latency": { "doubleValue": 0, "valueType": "DoubleType" },
                            "MaxAttempts": { "integerValue": 3, "valueType": "IntegerType" },
                            "PercentStateChange": { "doubleValue": 0, "valueType": "DoubleType" },
                            "PerformanceData": {
                                "stringValue": "rta=0.030ms;3000.000;5000.000;0; pl=0%;80;100;0;100 rtmax=0.030ms;;;; rtmin=0.030ms;;;;",
                                "valueType": "StringType"
                            },
                            "StateType": { "stringValue": "HARD", "valueType": "StringType" },
                            "isAcceptPassiveChecks": { "boolValue": true, "valueType": "BooleanType" },
                            "isChecksEnabled": { "boolValue": true, "valueType": "BooleanType" },
                            "isEventHandlersEnabled": { "boolValue": true, "valueType": "BooleanType" },
                            "isFlapDetectionEnabled": { "boolValue": true, "valueType": "BooleanType" },
                            "isNotificationsEnabled": { "boolValue": true, "valueType": "BooleanType" },
                            "isProblemAcknowledged": { "boolValue": false, "valueType": "BooleanType" }
                        },
                        "status": "SERVICE_OK",
                        "type": "service"
                    }
                ],
                "status": "HOST_UP",
                "type": "host"
            },
            {
                "description": "atest22",
                "device": "127.1.1.22",
                "lastCheckTime": "1748333257000",
                "lastPluginOutput": "OK - 127.1.1.22 rta 0.048ms lost 0%",
                "name": "atest22",
                "nextCheckTime": "1748333317000",
                "properties": {
                    "Alias": { "stringValue": "atest22", "valueType": "StringType" },
                    "CheckType": { "stringValue": "ACTIVE", "valueType": "StringType" },
                    "CurrentAttempt": { "integerValue": 1, "valueType": "IntegerType" },
                    "CurrentNotificationNumber": { "integerValue": 0, "valueType": "IntegerType" },
                    "ExecutionTime": { "doubleValue": 0.003, "valueType": "DoubleType" },
                    "LastNotificationTime": { "timeValue": "0", "valueType": "TimeType" },
                    "LastStateChange": { "timeValue": "1747683049000", "valueType": "TimeType" },
                    "Latency": { "doubleValue": 0.001, "valueType": "DoubleType" },
                    "MaxAttempts": { "integerValue": 3, "valueType": "IntegerType" },
                    "PercentStateChange": { "doubleValue": 0, "valueType": "DoubleType" },
                    "PerformanceData": {
                        "stringValue": "rta=0.048ms;3000.000;5000.000;0; pl=0%;80;100;0;100 rtmax=0.048ms;;;; rtmin=0.048ms;;;;",
                        "valueType": "StringType"
                    },
                    "StateType": { "stringValue": "HARD", "valueType": "StringType" },
                    "isAcknowledged": { "boolValue": false, "valueType": "BooleanType" },
                    "isChecksEnabled": { "boolValue": true, "valueType": "BooleanType" },
                    "isEventHandlersEnabled": { "boolValue": true, "valueType": "BooleanType" },
                    "isFlapDetectionEnabled": { "boolValue": true, "valueType": "BooleanType" },
                    "isNotificationsEnabled": { "boolValue": true, "valueType": "BooleanType" },
                    "isPassiveChecksEnabled": { "boolValue": true, "valueType": "BooleanType" }
                },
                "services": [
                    {
                        "lastCheckTime": "1748333259000",
                        "lastPluginOutput": "OK - 127.1.1.22 rta 0.033ms lost 0%",
                        "name": "icmp_ping_alive",
                        "nextCheckTime": "1748333319000",
                        "properties": {
                            "CheckType": { "stringValue": "ACTIVE", "valueType": "StringType" },
                            "CurrentAttempt": { "integerValue": 1, "valueType": "IntegerType" },
                            "CurrentNotificationNumber": { "integerValue": 0, "valueType": "IntegerType" },
                            "ExecutionTime": { "doubleValue": 0.004, "valueType": "DoubleType" },
                            "LastNotificationTime": { "timeValue": "0", "valueType": "TimeType" },
                            "Latency": { "doubleValue": 0, "valueType": "DoubleType" },
                            "MaxAttempts": { "integerValue": 3, "valueType": "IntegerType" },
                            "PercentStateChange": { "doubleValue": 0, "valueType": "DoubleType" },
                            "PerformanceData": {
                                "stringValue": "rta=0.033ms;3000.000;5000.000;0; pl=0%;80;100;0;100 rtmax=0.033ms;;;; rtmin=0.033ms;;;;",
                                "valueType": "StringType"
                            },
                            "StateType": { "stringValue": "HARD", "valueType": "StringType" },
                            "isAcceptPassiveChecks": { "boolValue": true, "valueType": "BooleanType" },
                            "isChecksEnabled": { "boolValue": true, "valueType": "BooleanType" },
                            "isEventHandlersEnabled": { "boolValue": true, "valueType": "BooleanType" },
                            "isFlapDetectionEnabled": { "boolValue": true, "valueType": "BooleanType" },
                            "isNotificationsEnabled": { "boolValue": true, "valueType": "BooleanType" },
                            "isProblemAcknowledged": { "boolValue": false, "valueType": "BooleanType" }
                        },
                        "status": "SERVICE_OK",
                        "type": "service"
                    }
                ],
                "status": "HOST_UP",
                "type": "host"
            }
        ]
    }`

	t.Run("filterExtInfo", func(t *testing.T) {
		var inv struct {
			transit.InventoryRequest
			Context []byte `json:"context,omitempty"`
		}
		var mon struct {
			transit.ResourcesWithServicesRequest
			Context []byte `json:"context,omitempty"`
		}
		var dt struct {
			transit.Downtimes
			Context []byte `json:"context,omitempty"`
		}
		assert.NoError(t, json.Unmarshal([]byte(jsonInventoryExt), &inv))

		filterExtInfo(&inv.InventoryRequest, &mon.ResourcesWithServicesRequest, &dt.Downtimes)

		payload, err := json.Marshal(inv)
		assert.NoError(t, err)
		assert.JSONEq(t, jsonInventory, string(payload))

		payload, err = json.Marshal(mon)
		assert.NoError(t, err)
		assert.JSONEq(t, jsonMonitoring, string(payload))
	})
}
