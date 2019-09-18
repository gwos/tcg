package transit

var MONITOR_STATUS_ENUM = `
{ "type": "enum",
  "name": "MonitorStatusEnum",
  "symbols" : ["SERVICE_OK", "SERVICE_UNSCHEDULED_CRITICAL", "SERVICE_WARNING", "SERVICE_PENDING", 
              "SERVICE_SCHEDULED_CRITICAL", "SERVICE_UNKNOWN", "HOST_UP", "HOST_UNSCHEDULED_DOWN", "HOST_WARNING"
              "HOST_PENDING", "HOST_SCHEDULED_DOWN", "HOST_UNREACHABLE"]
}A
`

var MONITORED_RESOURCE = `{
    "type": "record",
    "name": "MonitoredResource",
    "namespace": "gwos.tng",
	"fields": [
        {
            "name": "name",
            "type": "string"
        },
        {
            "name": "type",
            "type": "string"
        },
		{
			"name": "status",
			"type": "string"
		},
		{
			"name": "labels", 
			"type": {"type": "map", "values": "string"}
		}
	]
}`

