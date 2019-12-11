// Unit tests for routines generated from the transport.go source code.

#include <string.h>

#include "jansson.h"

#include "convert_go_to_c.h"

#include "setup.h"
#include "subseconds.h"
#include "transport.h"

#define	FAILURE	0	// for use in routine return values 
#define	SUCCESS	1	// for use in routine return values

// Sample routines for use by application code:

// extern char   *transport_Transit_ptr_as_JSON_str(const transport_Transit *transport_Transit);
// extern transport_Transit *JSON_str_as_transport_Transit_ptr(const char *json_str, json_t **json);

// extern char *transport_MonitoredResource_ptr_as_JSON_str(const transport_MonitoredResource *transport_MonitoredResource);
// extern transport_MonitoredResource *JSON_str_as_transport_MonitoredResource_ptr(const char *json_str, json_t **json);

// Sample internal conversion routines, generally not of interest to applications:

// extern json_t *transport_Transit_ptr_as_JSON_ptr(const transport_Transit *transport_Transit);
// extern transport_Transit *JSON_as_transport_Transit(json_t *json); 

// extern json_t *transport_MonitoredResource_ptr_as_JSON_ptr(const transport_MonitoredResource *transport_MonitoredResource);
// extern transport_MonitoredResource *JSON_as_transport_MonitoredResource(json_t *json);

// We make this a const string to attempt to bypass some overly aggressive compiler security warnings.
const char separation_line[] = "--------------------------------------------------------------------------------\n";

char *initial_transport_TimeInterval_as_json_string = "{\n"
"    \"endTime\": 1572955806397,\n"
"    \"startTime\": 1572955806397\n"
"}";

char *initial_transport_TypedValue_as_json_string = "{\n"
"    \"valueType\": \"IntegerType\",\n"
"    \"integerValue\": 1\n"
"}";

char *initial_transport_LabelDescriptor_as_json_string = "{\n"
"    \"description\": \"TestDescription\",\n"
"    \"key\": \"TestKey\",\n"
"    \"valueType\": \"StringType\"\n"
"}";

char *initial_transport_ThresholdDescriptor_as_json_string = "{\n"
"    \"key\": \"TestKey\",\n"
"    \"value\": 2\n"
"}";

char *initial_transport_SendInventoryRequest_as_json_string = "{\n"
"    \"resources\": [\n"
"        {\n"
"            \"name\": \"TestName\",\n"
"            \"type\": \"TestType\",\n"
"            \"owner\": \"TestOwner\",\n"
"            \"category\": \"TestCategory\",\n"
"            \"description\": \"TestDescription\",\n"
"            \"device\": \"TestDevice\",\n"
"            \"properties\": {\n"
"                \"property\": {\n"
"                    \"valueType\": \"IntegerType\",\n"
"                    \"integerValue\": 1\n"
"                }\n"
"            }\n"
"        }\n"
"    ],\n"
"    \"groups\": [\n"
"        {\n"
"            \"groupName\": \"TestGroupName\",\n"
"            \"resources\": [\n"
"                {\n"
"                    \"name\": \"TestName\",\n"
"                    \"type\": \"host\",\n"
"                    \"owner\": \"TestOwner\"\n"
"                }\n"
"            ]\n"
"        }\n"
"    ]\n"
"}";

char *initial_transport_OperationResult_as_json_string = "{\n"
"    \"entity\": \"TestEntity\",\n"
"    \"status\": \"TestStatus\",\n"
"    \"message\": \"TestMessage\",\n"
"    \"location\": \"TestLocation\",\n"
"    \"entityID\": 173\n"
"}";

char *initial_transport_ResourceGroup_as_json_string = "{\n"
"    \"groupName\": \"TestGroupName\",\n"
"    \"resources\": [\n"
"        {\n"
"            \"name\": \"TestName\",\n"
"            \"type\": \"service\",\n"
"            \"owner\": \"TestOwner\"\n"
"        }\n"
"    ]\n"
"}";

char *initial_transport_ResourceWithMetricsRequest_as_json_string = "{\n"
"    \"context\": {\n"
"        \"appType\": \"TestAppType\",\n"
"        \"agentID\": \"TestAgentID\",\n"
"        \"traceToken\": \"TestTraceToken\",\n"
"        \"timeStamp\": 1572955806398\n"
"    },\n"
"    \"resources\": [\n"
"        {\n"
"            \"resource\": {\n"
"                \"name\": \"TestName\",\n"
"                \"type\": \"TestType\",\n"
"                \"owner\": \"TestOwner\",\n"
"                \"status\": \"SERVICE_OK\",\n"
"                \"lastCheckTime\": 1572955806398,\n"
"                \"nextCheckTime\": 1572955806398,\n"
"                \"lastPlugInOutput\": \"Test plugin output\",\n"
"                \"properties\": {\n"
"                    \"property\": {\n"
"                        \"valueType\": \"IntegerType\",\n"
"                        \"integerValue\": 1\n"
"                    }\n"
"                }\n"
"            },\n"
"            \"metrics\": [\n"
"                {\n"
"                    \"metricName\": \"TestMetric\",\n"
"                    \"metricSamples\": [\n"
"                        {\n"
"                            \"sampleType\": \"Warning\",\n"
"                            \"interval\": {\n"
"                                \"endTime\": 1572955806397,\n"
"                                \"startTime\": 1572955806397\n"
"                            },\n"
"                            \"value\": {\n"
"                                \"valueType\": \"IntegerType\",\n"
"                                \"integerValue\": 1\n"
"                            }\n"
"                        }\n"
"                    ],\n"
"                    \"tags\": {\n"
"                        \"tag1\": \"TAG_1\",\n"
"                        \"tag2\": \"TAG_2\"\n"
"                    },\n"
"                    \"unit\": \"1\"\n"
"                }\n"
"            ]\n"
"        }\n"
"    ]\n"
"}";

char *initial_transport_MonitoredResource_as_json_string = "{\n"
"    \"name\": \"dbserver\",\n"
"    \"type\": \"host\",\n"
"    \"owner\": \"charley\"\n"
"}";

char *initial_transport_Transit_as_json_string = "{\n"
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

char *initial_transport_MetricSample_as_json_string = "{\n"
"    \"sampleType\": \"Warning\",\n"
"    \"interval\": {\n"
"        \"endTime\": 1572955806397,\n"
"        \"startTime\": 1572955806397\n"
"    },\n"
"    \"value\": {\n"
"        \"valueType\": \"IntegerType\",\n"
"        \"integerValue\": 1\n"
"    }\n"
"}";

char *initial_transport_TracerContext_as_json_string = "{\n"
"    \"appType\": \"TestAppType\",\n"
"    \"agentID\": \"TestAgentID\",\n"
"    \"traceToken\": \"TestTraceToken\",\n"
"    \"timeStamp\": 1572955806398\n"
"}";

char *initial_transport_TimeSeries_as_json_string = "{\n"
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
"                \"integerValue\": 1\n"
"            }\n"
"        }\n"
"    ],\n"
"    \"tags\": {\n"
"        \"tag1\": \"TAG_1\",\n"
"        \"tag2\": \"TAG_2\"\n"
"    },\n"
"    \"unit\": \"1\"\n"
"}";

char *initial_transport_ResourceWithMetrics_as_json_string = "{\n"
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
"                \"integerValue\": 1\n"
"            }\n"
"        }\n"
"    },\n"
"    \"metrics\": [\n"
"        {\n"
"            \"metricName\": \"TestMetric\",\n"
"            \"metricSamples\": [\n"
"                {\n"
"                    \"sampleType\": \"Warning\",\n"
"                    \"interval\": {\n"
"                        \"endTime\": 1572955806397,\n"
"                        \"startTime\": 1572955806397\n"
"                    },\n"
"                    \"value\": {\n"
"                        \"valueType\": \"IntegerType\",\n"
"                        \"integerValue\": 1\n"
"                    }\n"
"                }\n"
"            ],\n"
"            \"tags\": {\n"
"                \"tag1\": \"TAG_1\",\n"
"                \"tag2\": \"TAG_2\"\n"
"            },\n"
"            \"unit\": \"1\"\n"
"        }\n"
"    ]\n"
"}";

char *initial_transport_OperationResults_as_json_string = "{\n"
"    \"successful\": 1,\n"
"    \"failed\": 0,\n"
"    \"entityType\": \"TestEntity\",\n"
"    \"operation\": \"Insert\",\n"
"    \"warning\": 0,\n"
"    \"count\": 1,\n"
"    \"results\": [\n"
"        {\n"
"            \"entity\": \"TestEntity\",\n"
"            \"status\": \"TestStatus\",\n"
"            \"message\": \"TestMessage\",\n"
"            \"location\": \"TestLocation\",\n"
"            \"entityID\": 173\n"
"        }\n"
"    ]\n"
"}";

char *initial_transport_AgentStats_as_json_string = "{\n"
"    \"AgentID\": \"TestAgentID\",\n"
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

char *initial_transport_MetricDescriptor_as_json_string = "{\n"
"    \"name\": \"TestCustomName\",\n"
"    \"description\": \"TestDescription\",\n"
"    \"displayName\": \"TestDisplayName\",\n"
"    \"labels\": [\n"
"        {\n"
"            \"description\": \"TestDescription\",\n"
"            \"key\": \"TestKey\",\n"
"            \"valueType\": \"StringType\"\n"
"        }\n"
"    ],\n"
"    \"thresholds\": [\n"
"        {\n"
"            \"key\": \"TestKey\",\n"
"            \"value\": 2\n"
"        }\n"
"    ],\n"
"    \"type\": \"TestType\",\n"
"    \"unit\": \"1\",\n"
"    \"valueType\": \"IntegerType\",\n"
"    \"computeType\": \"Query\",\n"
"    \"metricKind\": \"GAUGE\"\n"
"}";

char *initial_transport_ResourceStatus_as_json_string = "{\n"
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
"            \"integerValue\": 1\n"
"        }\n"
"    }\n"
"}";

char *initial_transport_InventoryResource_as_json_string = "{\n"
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
	    character = 0;
	}
    }
    if (*a != *b) {
	char a_string[] = "  ";
	char b_string[] = "  ";
	if (*a < ' ') {
	    a_string[0] = '\\';
	    a_string[1] = *a == '\r' ? 'r' : *a == '\n' ? 'n' : *a == '\t' ? 't' : *a + 0x40;
	    if (a_string[1] < ' ') {
		a_string[0] = '^';
		a_string[1] = *a + 0x40;
	    }
	} else {
	    a_string[0] = *a;
	    a_string[1] = '\0';
	}
	if (*b < ' ') {
	    b_string[0] = '\\';
	    b_string[1] = *b == '\r' ? 'r' : *b == '\n' ? 'n' : *b == '\t' ? 't' : *b;
	    if (b_string[1] < ' ') {
		b_string[0] = '^';
		b_string[1] = *b + 0x40;
	    }
	} else {
	    b_string[0] = *b;
	    b_string[1] = '\0';
	}
	printf("strings are different at position %d (line %d char %d ['%s' vs. '%s'])\n", i, line, character, a_string, b_string);
    }
}

#define test_json_string(OBJECT, OBJSTR)											\
int test_##OBJECT##_json_string(bool enable) {											\
    printf(separation_line);													\
    if (enable) {														\
	json_t *json;														\
	printf("Decoding "OBJSTR" JSON string ...\n");										\
	OBJECT *OBJECT##_ptr = JSON_str_as_##OBJECT##_ptr(initial_##OBJECT##_as_json_string, &json);				\
	if (OBJECT##_ptr == NULL) {												\
	    printf (FILE_LINE "ERROR:  JSON string cannot be decoded into a "OBJSTR" object\n");				\
	    return FAILURE;													\
	}															\
	else {															\
	    printf ("Encoding "OBJSTR" object tree ...\n");									\
	    char *final_##OBJECT##_as_json_string = OBJECT##_ptr_as_JSON_str(OBJECT##_ptr);					\
	    if (final_##OBJECT##_as_json_string == NULL) {									\
		printf (FILE_LINE "ERROR:  "OBJSTR" object cannot be encoded as a JSON string\n");				\
		return FAILURE;													\
	    }															\
	    else {														\
		int matches = !strcmp(final_##OBJECT##_as_json_string, initial_##OBJECT##_as_json_string);			\
		printf (													\
		    "Final string for decode/encode of "OBJSTR" %s the original string.\n",					\
		    (matches ? "matches" : "DOES NOT MATCH")									\
		);														\
		if (!matches) {													\
		    printf("original string:\n%s\n", initial_##OBJECT##_as_json_string);					\
		    printf("   final string:\n%s\n",   final_##OBJECT##_as_json_string);					\
		    print_first_different_character(initial_##OBJECT##_as_json_string, final_##OBJECT##_as_json_string);	\
		    return FAILURE;												\
		}														\
		free(final_##OBJECT##_as_json_string);										\
	    }															\
	    free_##OBJECT##_ptr_tree(OBJECT##_ptr, json);									\
	}															\
	/*															\
	// We use just the first of these two calls (done just above, for now), because it's our official cleanup routine.	\
	destroy_##OBJECT##_tree(OBJECT##_ptr, json);										\
	free_JSON(json);													\
	*/															\
    }																\
    else {															\
	printf("--- skipping "OBJSTR" JSON string ...\n");									\
    }																\
    return SUCCESS;														\
}

#define test_object(OBJECT)			test_json_string(OBJECT, stringify(OBJECT))
#define run_object_test(ENABLE, OBJECT)		test_##OBJECT##_json_string(ENABLE)

// Generate all the individual test functions for particular objects.
test_object(transport_TimeInterval)
test_object(transport_TypedValue)
test_object(transport_LabelDescriptor)
test_object(transport_ThresholdDescriptor)
test_object(transport_SendInventoryRequest)
test_object(transport_OperationResult)
test_object(transport_ResourceGroup)
test_object(transport_ResourceWithMetricsRequest)
test_object(transport_MonitoredResource)
test_object(transport_Transit)
test_object(transport_MetricSample)
test_object(transport_TracerContext)
test_object(transport_TimeSeries)
test_object(transport_ResourceWithMetrics)
test_object(transport_OperationResults)
// This section is awaiting an update of the upstream Go code, to make AgentStats available.
// test_object(transport_AgentStats)
test_object(transport_MetricDescriptor)
test_object(transport_ResourceStatus)
test_object(transport_InventoryResource)

int main (int argc, char *argv[]) {
    json_t *json;

    int success = 1
	&& run_object_test(true, transport_TimeInterval)
	&& run_object_test(true, transport_TypedValue)
	&& run_object_test(true, transport_LabelDescriptor)
	&& run_object_test(true, transport_ThresholdDescriptor)
	&& run_object_test(true, transport_SendInventoryRequest)
	&& run_object_test(true, transport_OperationResult)
	&& run_object_test(true, transport_ResourceGroup)
	&& run_object_test(true, transport_ResourceWithMetricsRequest)
	&& run_object_test(true, transport_MonitoredResource)
	&& run_object_test(true, transport_Transit)
	&& run_object_test(true, transport_MetricSample)
	&& run_object_test(true, transport_TracerContext)
	// || 0
	&& run_object_test(true, transport_TimeSeries)
	&& run_object_test(true, transport_ResourceWithMetrics)
	&& run_object_test(true, transport_OperationResults)
	// This section is awaiting an update of the upstream Go code, to make AgentStats available.
	// && run_object_test(true, transport_AgentStats)
	&& run_object_test(true, transport_MetricDescriptor)
	&& run_object_test(true, transport_ResourceStatus)
	&& run_object_test(true, transport_InventoryResource)
    ;
    printf(separation_line);
    if (success) {
	printf("All enabled tests PASSED.\n");
    }
    else {
	printf("Some enabled tests FAILED.\n");
    }

    return success ? EXIT_SUCCESS : EXIT_FAILURE;
}
