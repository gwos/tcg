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

char *initial_transit_MonitoredResource_as_json_string = "{\n"
"    \"Name\": \"dbserver\",\n"
"    \"Type\": \"host\",\n"
"    \"Owner\": \"charley\"\n"
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

char *initial_transit_InventoryResource_as_json_string = "{\n"
"    \"Name\": \"TestName\",\n"
"    \"Type\": \"TestType\",\n"
"    \"Owner\": \"TestOwner\",\n"
"    \"Category\": \"TestCategory\",\n"
"    \"Description\": \"TestDescription\",\n"
"    \"Device\": \"TestDevice\",\n"
"    \"Properties\": {\n"
"        \"SampleTimeProperty\": {\n"
"            \"ValueType\": \"TimeType\",\n"
"            \"TimeValue\": 1572955806397\n"
"        },\n"
"        \"SampleBooleanProperty\": {\n"
"            \"ValueType\": \"BooleanType\",\n"
"            \"BoolValue\": true\n"
"        },\n"
"        \"SampleIntegerProperty\": {\n"
"            \"ValueType\": \"IntegerType\",\n"
"            \"IntegerValue\": 1234\n"
"        },\n"
"        \"SampleStringProperty\": {\n"
"            \"ValueType\": \"StringType\",\n"
"            \"StringValue\": \"arbitrary string\"\n"
"        },\n"
"        \"SampleDoubleProperty\": {\n"
"            \"ValueType\": \"DoubleType\",\n"
"            \"DoubleValue\": 2.7182818284590451\n"
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
    printf("--------------------------\n");
    if (true) {
	transit_MonitoredResource *transit_MonitoredResource_ptr = JSON_str_as_transit_MonitoredResource(initial_transit_MonitoredResource_as_json_string, &json);
	if (transit_MonitoredResource_ptr == NULL) {
	    printf ("ERROR:  JSON string cannot be decoded into a transit_MonitoredResource object\n");
	    return EXIT_FAILURE;
	}
	else {
	    // Before we encode, let's first run some obvious checks to make sure the decoding worked as expected.
	    printf("after decoding string, transit_MonitoredResource_ptr->Name  = '%s'\n", transit_MonitoredResource_ptr->Name);
	    printf("after decoding string, transit_MonitoredResource_ptr->Type  = '%d'\n", transit_MonitoredResource_ptr->Type);
	    printf("after decoding string, transit_MonitoredResource_ptr->Owner = '%s'\n", transit_MonitoredResource_ptr->Owner);

	    char *final_transit_MonitoredResource_as_json_string = transit_MonitoredResource_as_JSON_str(transit_MonitoredResource_ptr);
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
		    return EXIT_FAILURE;
		}
	    }
	}
	// We want to use just the first call, but we're not yet linking in the object code for it.
	// destroy_transit_MonitoredResource_tree(transit_MonitoredResource_ptr, json);
	free_JSON(json);
    }
    printf("--------------------------\n");
    if (true) {
	printf("--- decoding JSON string ...\n");
	transit_Transit *transit_Transit_ptr = JSON_str_as_transit_Transit(initial_transit_Transit_as_json_string, &json);
	if (transit_Transit_ptr == NULL) {
	    printf ("ERROR:  JSON string cannot be decoded into a transit_Transit object\n");
	    return EXIT_FAILURE;
	}
	else {
	    // Before we go encoding the object tree, let's first run some tests to see whether
	    // the JSON string got decoded into the object-tree values that we expect.
	    printf ("value of transit_Transit_ptr->config_Config_ptr_.AgentConfig.ControllerAddr = %s\n",
		transit_Transit_ptr->config_Config_ptr_->AgentConfig.ControllerAddr);
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
		    return EXIT_FAILURE;
		}
	    }
	}
	// We want to use just the first call, but we're not yet linking in the object code for it.
	// destroy_transit_Transit_tree(transit_Transit_ptr, json);
	free_JSON(json);
    }
    printf("--------------------------\n");
    if (true) {
	printf("--- decoding JSON string ...\n");
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
    return EXIT_SUCCESS;
}
