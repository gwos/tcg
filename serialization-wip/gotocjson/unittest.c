// Unit tests for routines generated from the transit.go source code.

#include <string.h>

#include "jansson.h"

#include "convert_go_to_c.h"

#include "config.h"
#include "milliseconds.h"
#include "transit.h"

// Internal routines:
// extern json_t *transit_Transit_as_JSON(const transit_Transit *transit_Transit);
// extern transit_Transit *JSON_as_transit_Transit(json_t *json); 

// extern json_t *transit_MonitoredResource_as_JSON(const transit_MonitoredResource *transit_MonitoredResource);
// extern transit_MonitoredResource *JSON_as_transit_MonitoredResource(json_t *json);

// Routines for use by application code:
// extern char   *transit_Transit_as_JSON_str(const transit_Transit *transit_Transit);
// extern transit_Transit *JSON_str_as_transit_Transit(const char *json_str, json_t **json);

// extern char *transit_MonitoredResource_as_JSON_str(const transit_MonitoredResource *transit_MonitoredResource);
// extern transit_MonitoredResource *JSON_str_as_transit_MonitoredResource(const char *json_str, json_t **json);

char separation_line[] = "--------------------------------------------------------------------------------\n";

char *initial_transit_MonitoredResource_as_json_string = "{\n"
"    \"name\": \"dbserver\",\n"
"    \"type\": \"host\",\n"
"    \"owner\": \"charley\"\n"
"}";

char *initial_transit_Transit_as_json_string = "{\n"
"    \"Config\": {\n"
"        \"AgentConfig\": {\n"
"            \"ControllerAddr\": \":http\",\n"
"            \"ControllerCertFile\": \"/path/to/controller/certfile\",\n"
"            \"ControllerKeyFile\": \"/path/to/controller/keyfile\",\n"
"            \"NATSFilestoreDir\": \"/nats/filestore\",\n"
"            \"NATSStoreType\": \"MEMORY\",\n"
"            \"StartController\": true,\n"
"            \"StartNATS\": false,\n"
"            \"StartTransport\": false\n"
"        },\n"
"        \"GroundworkConfig\": {\n"
"            \"Host\": \"host_name\",\n"
"            \"Account\": \"account_name\",\n"
"            \"Password\": \"config_password\",\n"
"            \"Token\": \"config_token\",\n"
"            \"AppName\": \"config_app_name\"\n"
"        },\n"
"        \"GroundworkActions\": {\n"
"            \"Connect\": {\n"
"                \"Entrypoint\": \"connect_entry_point\"\n"
"            },\n"
"            \"Disconnect\": {\n"
"                \"Entrypoint\": \"disconnect_entry_point\"\n"
"            },\n"
"            \"SynchronizeInventory\": {\n"
"                \"Entrypoint\": \"synchronize_inventory_entry_point\"\n"
"            },\n"
"            \"SendResourceWithMetrics\": {\n"
"                \"Entrypoint\": \"send_resource_with_metrics_entry_point\"\n"
"            },\n"
"            \"ValidateToken\": {\n"
"                \"Entrypoint\": \"validate_token_entry_point\"\n"
"            }\n"
"        }\n"
"    }\n"
"}";

char *initial_transit_MetricSample_as_json_string = "{\n"
"    \"sampleType\": \"Warning\",\n"
"    \"interval\": {\n"
"        \"endTime\": 1572955806397,\n"
"        \"startTime\": 1572955806397\n"
"    },\n"
"    \"value\": {\n"
"        \"valueType\": \"IntegerType\",\n"
"        \"integerValue\": 1,\n"
"        \"timeValue\": 1572955806397\n"
"    }\n"
"}";

char *initial_transit_TracerContext_as_json_string = "{\n"
"    \"appType\": \"TestAppType\",\n"
"    \"agentID\": \"TestAgentID\",\n"
"    \"traceToken\": \"TestTraceToken\",\n"
"    \"timeStamp\": 1572955806398\n"
"}";

char *initial_transit_TimeSeries_as_json_string = "{\n"
"    \"metricName\": \"TestMetric\",\n"
"    \"metricSamples\": [\n"
"        {\n"
"            \"sampleType\": \"Warning\",\n"
"            \"interval\": {\n"
"                \"endTime\": 1572955806397,\n"
"                \"startTime\": 1572955806397\n"
"            },\n"
"            \"value\": {\n"
"                \"valueType\": \"IntegerType\",\n"
"                \"integerValue\": 1,\n"
"                \"timeValue\": 1572955806397\n"
"            }\n"
"        }\n"
"    ],\n"
"    \"tags\": {\n"
"        \"tag1\": \"TAG_1\",\n"
"        \"tag2\": \"TAG_2\"\n"
"    },\n"
"    \"unit\": \"1\"\n"
"}";

char *initial_transit_ResourceWithMetrics_as_json_string = "{\n"
"    \"resource\": {\n"
"        \"name\": \"TestName\",\n"
"        \"type\": \"TestType\",\n"
"        \"owner\": \"TestOwner\",\n"
"        \"status\": \"SERVICE_OK\",\n"
"        \"lastCheckTime\": 1572955806398,\n"
"        \"nextCheckTime\": 1572955806398,\n"
"        \"lastPlugInOutput\": \"Test plugin output\",\n"
"        \"properties\": {\n"
"            \"property\": {\n"
"                \"valueType\": \"IntegerType\",\n"
"                \"integerValue\": 1,\n"
"                \"timeValue\": 1572955806397\n"
"            }\n"
"        }\n"
"    },\n"
"    \"metrics\": [\n"
"        {\n"
"        \"metricName\": \"TestMetric\",\n"
"        \"metricSamples\": [\n"
"            {\n"
"                \"sampleType\": \"Warning\",\n"
"                \"interval\": {\n"
"                    \"endTime\": 1572955806397,\n"
"                    \"startTime\": 1572955806397\n"
"                },\n"
"                \"value\": {\n"
"                    \"valueType\": \"IntegerType\",\n"
"                    \"integerValue\": 1,\n"
"                    \"timeValue\": 1572955806397\n"
"                }\n"
"            }\n"
"        ],\n"
"        \"tags\": {\n"
"            \"tag1\": \"TAG_1\",\n"
"            \"tag2\": \"TAG_2\"\n"
"        },\n"
"        \"unit\": \"1\"\n"
"        }\n"
"    ]\n"
"}";

char *initial_transit_OperationResults_as_json_string = "{\n"
"    \"successful\": 1,\n"
"    \"failed\": 0,\n"
"    \"entityType\": \"TestEntity\",\n"
"    \"operation\": \"Insert\",\n"
"    \"warning\": 0,\n"
"    \"count\": 1,\n"
"    \"results\": [\n"
"        {\n"
"        \"entity\": \"TestEntity\",\n"
"        \"status\": \"TestStatus\",\n"
"        \"message\": \"TestMessage\",\n"
"        \"location\": \"TestLocation\",\n"
"        \"entityID\": 173\n"
"        }\n"
"    ]\n"
"}";

char *initial_transit_AgentStats_as_json_string = "{\n"
"    \"AgentID\": \"TestAgentId\",\n"
"    \"AppType\": \"TestAgentType\",\n"
"    \"BytesSent\": 1567,\n"
"    \"MetricsSent\": 12,\n"
"    \"MessagesSent\": 4,\n"
"    \"LastInventoryRun\": 1572958409541,\n"
"    \"LastMetricsRun\": 1572958409541,\n"
"    \"ExecutionTimeInventory\": 133,\n"
"    \"ExecutionTimeMetrics\": 234,\n"
"    \"UpSince\": 1572958409541,\n"
"    \"LastError\": \"Test last error\"\n"
"}";

char *initial_transit_MetricDescriptor_as_json_string = "{\n"
"    \"name\": \"TestCustomName\",\n"
"    \"description\": \"TestDescription\",\n"
"    \"displayName\": \"TestDisplayName\",\n"
"    \"labels\": [\n"
"        {\n"
"        \"description\": \"TestDescription\",\n"
"        \"key\": \"TestKey\",\n"
"        \"valueType\": \"TestValue\"\n"
"        }\n"
"    ],\n"
"    \"thresholds\": [\n"
"        {\n"
"        \"key\": \"TestKey\",\n"
"        \"value\": 2\n"
"        }\n"
"    ],\n"
"    \"type\": \"TestType\",\n"
"    \"unit\": \"1\",\n"
"    \"valueType\": \"IntegerType\",\n"
"    \"computeType\": \"Query\",\n"
"    \"metricKind\": \"GAUGE\"\n"
"}";

char *initial_transit_ResourceStatus_as_json_string = "{\n"
"    \"name\": \"TestName\",\n"
"    \"type\": \"TestType\",\n"
"    \"owner\": \"TestOwner\",\n"
"    \"status\": \"SERVICE_OK\",\n"
"    \"lastCheckTime\": 1572955806398,\n"
"    \"nextCheckTime\": 1572955806398,\n"
"    \"lastPlugInOutput\": \"Test plugin output\",\n"
"    \"properties\": {\n"
"        \"property\": {\n"
"            \"valueType\": \"IntegerType\",\n"
"            \"integerValue\": 1,\n"
"            \"timeValue\": 1572955806397\n"
"        }\n"
"    }\n"
"}";

char *initial_transit_InventoryResource_as_json_string = "{\n"
"    \"name\": \"TestName\",\n"
"    \"type\": \"TestType\",\n"
"    \"owner\": \"TestOwner\",\n"
"    \"category\": \"TestCategory\",\n"
"    \"description\": \"TestDescription\",\n"
"    \"device\": \"TestDevice\",\n"
"    \"properties\": {\n"
"        \"SampleBooleanProperty\": {\n"
"            \"valueType\": \"BooleanType\",\n"
"            \"boolValue\": true\n"
"        },\n"
"        \"SampleIntegerProperty\": {\n"
"            \"valueType\": \"IntegerType\",\n"
"            \"integerValue\": 1234\n"
"        },\n"
"        \"SampleTimeProperty\": {\n"
"            \"valueType\": \"TimeType\",\n"
"            \"timeValue\": 1572955806397\n"
"        },\n"
"        \"SampleStringProperty\": {\n"
"            \"valueType\": \"StringType\",\n"
"            \"stringValue\": \"arbitrary string\"\n"
"        },\n"
"        \"SampleDoubleProperty\": {\n"
"            \"valueType\": \"DoubleType\",\n"
"            \"doubleValue\": 2.7182818284590451\n"
"        }\n"
"    }\n"
"}";

void print_first_different_character(char *a, char *b) {
    int i;
    int line = 1;
    int character = 1;
    for (i = 0; *a && *b; ++i, ++character, ++a, ++b) {
        if (*a != *b) {
	    break;
	} else if (*a == '\n') {
	    ++line;
	    character = 1;
	}
    }
    if (*a != *b) {
        printf("strings are different at position %d (line %d char %d)\n", i, line, character);
    }
}

int main (int argc, char *argv[]) {
    json_t *json;
    printf(separation_line);
    if (true) {
	printf("--- decoding MonitoredResource JSON string ...\n");
	transit_MonitoredResource *transit_MonitoredResource_ptr = JSON_str_as_transit_MonitoredResource(initial_transit_MonitoredResource_as_json_string, &json);
	if (transit_MonitoredResource_ptr == NULL) {
	    printf ("ERROR:  JSON string cannot be decoded into a transit_MonitoredResource object\n");
	    return EXIT_FAILURE;
	}
	else {
	    // Before we encode, let's first run some obvious checks to make sure the decoding worked as expected.
	    /*
	    printf("after decoding string, transit_MonitoredResource_ptr->Name  = '%s'\n", transit_MonitoredResource_ptr->Name);
	    printf("after decoding string, transit_MonitoredResource_ptr->Type  = '%d'\n", transit_MonitoredResource_ptr->Type);
	    printf("after decoding string, transit_MonitoredResource_ptr->Owner = '%s'\n", transit_MonitoredResource_ptr->Owner);
	    */
	    printf ("--- encoding transit.MonitoredResource object tree ...\n");
	    char *final_transit_MonitoredResource_as_json_string = transit_MonitoredResource_as_JSON_str(transit_MonitoredResource_ptr);
	    printf ("--- encoding is complete, perhaps ...\n");
	    if (final_transit_MonitoredResource_as_json_string == NULL) {
		printf ("ERROR:  transit_MonitoredResource object cannot be encoded as a JSON string\n");
		return EXIT_FAILURE;
	    }
	    else {
		int matches = !strcmp(final_transit_MonitoredResource_as_json_string, initial_transit_MonitoredResource_as_json_string);
		printf ("Final string for decode/encode of transit.MonitoredResource %s the original string.\n", (matches ? "matches" : "DOES NOT MATCH"));
		if (!matches) {
		    printf("original string:\n%s\n", initial_transit_MonitoredResource_as_json_string);
		    printf("   final string:\n%s\n",   final_transit_MonitoredResource_as_json_string);
		    print_first_different_character(final_transit_MonitoredResource_as_json_string, initial_transit_MonitoredResource_as_json_string);
		    return EXIT_FAILURE;
		}
	    }
	}
	// We want to use just the first call, but we're not yet linking in the object code for it.
	// destroy_transit_MonitoredResource_tree(transit_MonitoredResource_ptr, json);
	free_JSON(json);
    }
    else {
	printf("--- skipping MonitoredResource JSON string ...\n");
    }
    printf(separation_line);
    if (true) {
	printf("--- decoding Transit JSON string ...\n");
	transit_Transit *transit_Transit_ptr = JSON_str_as_transit_Transit(initial_transit_Transit_as_json_string, &json);
	if (transit_Transit_ptr == NULL) {
	    printf ("ERROR:  JSON string cannot be decoded into a transit_Transit object\n");
	    return EXIT_FAILURE;
	}
	else {
	    // Before we go encoding the object tree, let's first run some tests to see whether
	    // the JSON string got decoded into the object-tree values that we expect.
	    /*
	    printf ("value of transit_Transit_ptr->config_Config_ptr_.AgentConfig.ControllerAddr = %s\n",
		transit_Transit_ptr->config_Config_ptr_->AgentConfig.ControllerAddr);
	    */
	    printf ("--- encoding transit.Transit object tree ...\n");
	    char *final_transit_Transit_as_json_string = transit_Transit_as_JSON_str(transit_Transit_ptr);
	    printf ("--- encoding is complete, perhaps ...\n");
	    if (final_transit_Transit_as_json_string == NULL) {
		printf ("ERROR:  transit_Transit object cannot be encoded as a JSON string\n");
		return EXIT_FAILURE;
	    }
	    else {
		int matches = ! strcmp(final_transit_Transit_as_json_string, initial_transit_Transit_as_json_string);
		printf ("Final string for decode/encode of transit.Transit %s the original string.\n", (matches ? "matches" : "DOES NOT MATCH"));
		if (!matches) {
		    printf("original string:\n%s\n", initial_transit_Transit_as_json_string);
		    printf("   final string:\n%s\n",   final_transit_Transit_as_json_string);
		    print_first_different_character(final_transit_Transit_as_json_string, initial_transit_Transit_as_json_string);
		    return EXIT_FAILURE;
		}
	    }
	}
	// We want to use just the first call, but we're not yet linking in the object code for it.
	// destroy_transit_Transit_tree(transit_Transit_ptr, json);
	free_JSON(json);
    }
    else {
	printf("--- skipping Transit JSON string ...\n");
    }
    printf(separation_line);
    if (true) {
	printf("--- decoding MetricSample JSON string ...\n");
	transit_MetricSample *transit_MetricSample_ptr = JSON_str_as_transit_MetricSample(initial_transit_MetricSample_as_json_string, &json);
	if (transit_MetricSample_ptr == NULL) {
	    printf ("ERROR:  JSON string cannot be decoded into a transit_MetricSample object\n");
	    return EXIT_FAILURE;
	}
	else {
	    printf ("--- encoding transit.MetricSample object tree ...\n");
	    char *final_transit_MetricSample_as_json_string = transit_MetricSample_as_JSON_str(transit_MetricSample_ptr);
	    printf ("--- encoding is complete, perhaps ...\n");
	    if (final_transit_MetricSample_as_json_string == NULL) {
		printf ("ERROR:  transit_MetricSample object cannot be encoded as a JSON string\n");
		return EXIT_FAILURE;
	    }
	    else {
		int matches = ! strcmp(final_transit_MetricSample_as_json_string, initial_transit_MetricSample_as_json_string);
		printf ("Final string for decode/encode of transit.MetricSample %s the original string.\n", (matches ? "matches" : "DOES NOT MATCH"));
		if (!matches) {
		    printf("original string:\n%s\n", initial_transit_MetricSample_as_json_string);
		    printf("   final string:\n%s\n",   final_transit_MetricSample_as_json_string);
		    print_first_different_character(final_transit_MetricSample_as_json_string, initial_transit_MetricSample_as_json_string);
		    return EXIT_FAILURE;
		}
	    }
	}
	// We want to use just the first call, but we're not yet linking in the object code for it.
	// destroy_transit_MetricSample_tree(transit_MetricSample_ptr, json);
	free_JSON(json);
    }
    else {
	printf("--- skipping MetricSample JSON string ...\n");
    }
    printf(separation_line);
    if (true) {
	printf("--- decoding TracerContext JSON string ...\n");
	transit_TracerContext *transit_TracerContext_ptr = JSON_str_as_transit_TracerContext(initial_transit_TracerContext_as_json_string, &json);
	if (transit_TracerContext_ptr == NULL) {
	    printf ("ERROR:  JSON string cannot be decoded into a transit_TracerContext object\n");
	    return EXIT_FAILURE;
	}
	else {
	    printf ("--- encoding transit.TracerContext object tree ...\n");
	    char *final_transit_TracerContext_as_json_string = transit_TracerContext_as_JSON_str(transit_TracerContext_ptr);
	    printf ("--- encoding is complete, perhaps ...\n");
	    if (final_transit_TracerContext_as_json_string == NULL) {
		printf ("ERROR:  transit_TracerContext object cannot be encoded as a JSON string\n");
		return EXIT_FAILURE;
	    }
	    else {
		int matches = ! strcmp(final_transit_TracerContext_as_json_string, initial_transit_TracerContext_as_json_string);
		printf ("Final string for decode/encode of transit.TracerContext %s the original string.\n", (matches ? "matches" : "DOES NOT MATCH"));
		if (!matches) {
		    printf("original string:\n%s\n", initial_transit_TracerContext_as_json_string);
		    printf("   final string:\n%s\n",   final_transit_TracerContext_as_json_string);
		    print_first_different_character(final_transit_TracerContext_as_json_string, initial_transit_TracerContext_as_json_string);
		    return EXIT_FAILURE;
		}
	    }
	}
	// We want to use just the first call, but we're not yet linking in the object code for it.
	// destroy_transit_TracerContext_tree(transit_TracerContext_ptr, json);
	free_JSON(json);
    }
    else {
	printf("--- skipping TracerContext JSON string ...\n");
    }
    printf(separation_line);
    if (true) {
	printf("--- decoding TimeSeries JSON string ...\n");
	transit_TimeSeries *transit_TimeSeries_ptr = JSON_str_as_transit_TimeSeries(initial_transit_TimeSeries_as_json_string, &json);
	if (transit_TimeSeries_ptr == NULL) {
	    printf ("ERROR:  JSON string cannot be decoded into a transit_TimeSeries object\n");
	    return EXIT_FAILURE;
	}
	else {
	    printf ("--- encoding transit.TimeSeries object tree ...\n");
	    char *final_transit_TimeSeries_as_json_string = transit_TimeSeries_as_JSON_str(transit_TimeSeries_ptr);
	    printf ("--- encoding is complete, perhaps ...\n");
	    if (final_transit_TimeSeries_as_json_string == NULL) {
		printf ("ERROR:  transit_TimeSeries object cannot be encoded as a JSON string\n");
		return EXIT_FAILURE;
	    }
	    else {
		int matches = ! strcmp(final_transit_TimeSeries_as_json_string, initial_transit_TimeSeries_as_json_string);
		printf ("Final string for decode/encode of transit.TimeSeries %s the original string.\n", (matches ? "matches" : "DOES NOT MATCH"));
		if (!matches) {
		    printf("original string:\n%s\n", initial_transit_TimeSeries_as_json_string);
		    printf("   final string:\n%s\n",   final_transit_TimeSeries_as_json_string);
		    print_first_different_character(final_transit_TimeSeries_as_json_string, initial_transit_TimeSeries_as_json_string);
		    return EXIT_FAILURE;
		}
	    }
	}
	// We want to use just the first call, but we're not yet linking in the object code for it.
	// destroy_transit_TimeSeries_tree(transit_TimeSeries_ptr, json);
	free_JSON(json);
    }
    else {
	printf("--- skipping TimeSeries JSON string ...\n");
    }
    printf(separation_line);
    if (false) {
	printf("--- decoding ResourceWithMetrics JSON string ...\n");
	transit_ResourceWithMetrics *transit_ResourceWithMetrics_ptr = JSON_str_as_transit_ResourceWithMetrics(initial_transit_ResourceWithMetrics_as_json_string, &json);
	if (transit_ResourceWithMetrics_ptr == NULL) {
	    printf ("ERROR:  JSON string cannot be decoded into a transit_ResourceWithMetrics object\n");
	    return EXIT_FAILURE;
	}
	else {
	    printf ("--- encoding transit.ResourceWithMetrics object tree ...\n");
	    char *final_transit_ResourceWithMetrics_as_json_string = transit_ResourceWithMetrics_as_JSON_str(transit_ResourceWithMetrics_ptr);
	    printf ("--- encoding is complete, perhaps ...\n");
	    if (final_transit_ResourceWithMetrics_as_json_string == NULL) {
		printf ("ERROR:  transit_ResourceWithMetrics object cannot be encoded as a JSON string\n");
		return EXIT_FAILURE;
	    }
	    else {
		int matches = ! strcmp(final_transit_ResourceWithMetrics_as_json_string, initial_transit_ResourceWithMetrics_as_json_string);
		printf ("Final string for decode/encode of transit.ResourceWithMetrics %s the original string.\n", (matches ? "matches" : "DOES NOT MATCH"));
		if (!matches) {
		    printf("original string:\n%s\n", initial_transit_ResourceWithMetrics_as_json_string);
		    printf("   final string:\n%s\n",   final_transit_ResourceWithMetrics_as_json_string);
		    print_first_different_character(final_transit_ResourceWithMetrics_as_json_string, initial_transit_ResourceWithMetrics_as_json_string);
		    return EXIT_FAILURE;
		}
	    }
	}
	// We want to use just the first call, but we're not yet linking in the object code for it.
	// destroy_transit_ResourceWithMetrics_tree(transit_ResourceWithMetrics_ptr, json);
	free_JSON(json);
    }
    else {
	printf("--- skipping ResourceWithMetrics JSON string ...\n");
    }
    printf(separation_line);
    if (false) {
	printf("--- decoding OperationResults JSON string ...\n");
	transit_OperationResults *transit_OperationResults_ptr = JSON_str_as_transit_OperationResults(initial_transit_OperationResults_as_json_string, &json);
	if (transit_OperationResults_ptr == NULL) {
	    printf ("ERROR:  JSON string cannot be decoded into a transit_OperationResults object\n");
	    return EXIT_FAILURE;
	}
	else {
	    printf ("--- encoding transit.OperationResults object tree ...\n");
	    char *final_transit_OperationResults_as_json_string = transit_OperationResults_as_JSON_str(transit_OperationResults_ptr);
	    printf ("--- encoding is complete, perhaps ...\n");
	    if (final_transit_OperationResults_as_json_string == NULL) {
		printf ("ERROR:  transit_OperationResults object cannot be encoded as a JSON string\n");
		return EXIT_FAILURE;
	    }
	    else {
		int matches = ! strcmp(final_transit_OperationResults_as_json_string, initial_transit_OperationResults_as_json_string);
		printf ("Final string for decode/encode of transit.OperationResults %s the original string.\n", (matches ? "matches" : "DOES NOT MATCH"));
		if (!matches) {
		    printf("original string:\n%s\n", initial_transit_OperationResults_as_json_string);
		    printf("   final string:\n%s\n",   final_transit_OperationResults_as_json_string);
		    print_first_different_character(final_transit_OperationResults_as_json_string, initial_transit_OperationResults_as_json_string);
		    return EXIT_FAILURE;
		}
	    }
	}
	// We want to use just the first call, but we're not yet linking in the object code for it.
	// destroy_transit_OperationResults_tree(transit_OperationResults_ptr, json);
	free_JSON(json);
    }
    else {
	printf("--- skipping OperationResults JSON string ...\n");
    }
    /*
    // This section is awaiting an update of the upstream Go code, to make AgentStats available.
    printf(separation_line);
    if (true) {
	printf("--- decoding AgentStats JSON string ...\n");
	transit_AgentStats *transit_AgentStats_ptr = JSON_str_as_transit_AgentStats(initial_transit_AgentStats_as_json_string, &json);
	if (transit_AgentStats_ptr == NULL) {
	    printf ("ERROR:  JSON string cannot be decoded into a transit_AgentStats object\n");
	    return EXIT_FAILURE;
	}
	else {
	    printf ("--- encoding transit.AgentStats object tree ...\n");
	    char *final_transit_AgentStats_as_json_string = transit_AgentStats_as_JSON_str(transit_AgentStats_ptr);
	    printf ("--- encoding is complete, perhaps ...\n");
	    if (final_transit_AgentStats_as_json_string == NULL) {
		printf ("ERROR:  transit_AgentStats object cannot be encoded as a JSON string\n");
		return EXIT_FAILURE;
	    }
	    else {
		int matches = ! strcmp(final_transit_AgentStats_as_json_string, initial_transit_AgentStats_as_json_string);
		printf ("Final string for decode/encode of transit.AgentStats %s the original string.\n", (matches ? "matches" : "DOES NOT MATCH"));
		if (!matches) {
		    printf("original string:\n%s\n", initial_transit_AgentStats_as_json_string);
		    printf("   final string:\n%s\n",   final_transit_AgentStats_as_json_string);
		    print_first_different_character(final_transit_AgentStats_as_json_string, initial_transit_AgentStats_as_json_string);
		    return EXIT_FAILURE;
		}
	    }
	}
	// We want to use just the first call, but we're not yet linking in the object code for it.
	// destroy_transit_AgentStats_tree(transit_AgentStats_ptr, json);
	free_JSON(json);
    }
    else {
	printf("--- skipping AgentStats JSON string ...\n");
    }
    */
    printf(separation_line);
    if (false) {
	printf("--- decoding MetricDescriptor JSON string ...\n");
	transit_MetricDescriptor *transit_MetricDescriptor_ptr = JSON_str_as_transit_MetricDescriptor(initial_transit_MetricDescriptor_as_json_string, &json);
	if (transit_MetricDescriptor_ptr == NULL) {
	    printf ("ERROR:  JSON string cannot be decoded into a transit_MetricDescriptor object\n");
	    return EXIT_FAILURE;
	}
	else {
	    printf ("--- encoding transit.MetricDescriptor object tree ...\n");
	    char *final_transit_MetricDescriptor_as_json_string = transit_MetricDescriptor_as_JSON_str(transit_MetricDescriptor_ptr);
	    printf ("--- encoding is complete, perhaps ...\n");
	    if (final_transit_MetricDescriptor_as_json_string == NULL) {
		printf ("ERROR:  transit_MetricDescriptor object cannot be encoded as a JSON string\n");
		return EXIT_FAILURE;
	    }
	    else {
		int matches = ! strcmp(final_transit_MetricDescriptor_as_json_string, initial_transit_MetricDescriptor_as_json_string);
		printf ("Final string for decode/encode of transit.MetricDescriptor %s the original string.\n", (matches ? "matches" : "DOES NOT MATCH"));
		if (!matches) {
		    printf("original string:\n%s\n", initial_transit_MetricDescriptor_as_json_string);
		    printf("   final string:\n%s\n",   final_transit_MetricDescriptor_as_json_string);
		    print_first_different_character(final_transit_MetricDescriptor_as_json_string, initial_transit_MetricDescriptor_as_json_string);
		    return EXIT_FAILURE;
		}
	    }
	}
	// We want to use just the first call, but we're not yet linking in the object code for it.
	// destroy_transit_MetricDescriptor_tree(transit_MetricDescriptor_ptr, json);
	free_JSON(json);
    }
    else {
	printf("--- skipping MetricDescriptor JSON string ...\n");
    }
    printf(separation_line);
    if (true) {
	printf("--- decoding ResourceStatus JSON string ...\n");
	transit_ResourceStatus *transit_ResourceStatus_ptr = JSON_str_as_transit_ResourceStatus(initial_transit_ResourceStatus_as_json_string, &json);
	if (transit_ResourceStatus_ptr == NULL) {
	    printf ("ERROR:  JSON string cannot be decoded into a transit_ResourceStatus object\n");
	    return EXIT_FAILURE;
	}
	else {
	    printf ("--- encoding transit.ResourceStatus object tree ...\n");
	    char *final_transit_ResourceStatus_as_json_string = transit_ResourceStatus_as_JSON_str(transit_ResourceStatus_ptr);
	    printf ("--- encoding is complete, perhaps ...\n");
	    if (final_transit_ResourceStatus_as_json_string == NULL) {
		printf ("ERROR:  transit_ResourceStatus object cannot be encoded as a JSON string\n");
		return EXIT_FAILURE;
	    }
	    else {
		int matches = ! strcmp(final_transit_ResourceStatus_as_json_string, initial_transit_ResourceStatus_as_json_string);
		printf ("Final string for decode/encode of transit.ResourceStatus %s the original string.\n", (matches ? "matches" : "DOES NOT MATCH"));
		if (!matches) {
		    printf("original string:\n%s\n", initial_transit_ResourceStatus_as_json_string);
		    printf("   final string:\n%s\n",   final_transit_ResourceStatus_as_json_string);
		    print_first_different_character(final_transit_ResourceStatus_as_json_string, initial_transit_ResourceStatus_as_json_string);
		    return EXIT_FAILURE;
		}
	    }
	}
	// We want to use just the first call, but we're not yet linking in the object code for it.
	// destroy_transit_ResourceStatus_tree(transit_ResourceStatus_ptr, json);
	free_JSON(json);
    }
    else {
	printf("--- skipping ResourceStatus JSON string ...\n");
    }
    printf(separation_line);
    if (true) {
	printf("--- decoding InventoryResource JSON string ...\n");
	transit_InventoryResource *transit_InventoryResource_ptr = JSON_str_as_transit_InventoryResource(initial_transit_InventoryResource_as_json_string, &json);
	if (transit_InventoryResource_ptr == NULL) {
	    printf ("ERROR:  JSON string cannot be decoded into a transit_InventoryResource object\n");
	    return EXIT_FAILURE;
	}
	else {
	    printf ("--- encoding transit.InventoryResource object tree ...\n");
	    char *final_transit_InventoryResource_as_json_string = transit_InventoryResource_as_JSON_str(transit_InventoryResource_ptr);
	    printf ("--- encoding is complete, perhaps ...\n");
	    if (final_transit_InventoryResource_as_json_string == NULL) {
		printf ("ERROR:  transit_InventoryResource object cannot be encoded as a JSON string\n");
		return EXIT_FAILURE;
	    }
	    else {
		int matches = ! strcmp(final_transit_InventoryResource_as_json_string, initial_transit_InventoryResource_as_json_string);
		printf ("Final string for decode/encode of transit.InventoryResource %s the original string.\n", (matches ? "matches" : "DOES NOT MATCH"));
		if (!matches) {
		    printf("original string:\n%s\n", initial_transit_InventoryResource_as_json_string);
		    printf("   final string:\n%s\n",   final_transit_InventoryResource_as_json_string);
		    print_first_different_character(final_transit_InventoryResource_as_json_string, initial_transit_InventoryResource_as_json_string);
		    return EXIT_FAILURE;
		}
	    }
	}
	// We want to use just the first call, but we're not yet linking in the object code for it.
	// destroy_transit_InventoryResource_tree(transit_InventoryResource_ptr, json);
	free_JSON(json);
    }
    else {
	printf("--- skipping InventoryResource JSON string ...\n");
    }
    return EXIT_SUCCESS;
}
