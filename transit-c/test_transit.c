#include <dlfcn.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "libtransit.h"
#include "transit.h"
#include "transit_json.h"
#include "util.h"

void test_defineMonitoredResource() {
  TypedValue prop0 = {BooleanType, true};
  TypedValue prop1 = {DoubleType, doubleValue : 0.1};
  TypedValue prop2 = {StringType, stringValue : "val_002"};
  TypedValuePair props[] = {
      {"key0", prop0}, {"key01", prop1}, {"key002", prop2}};

  MonitoredResource resource = {SERVICE_OK,
                                "the-unique-name-of-the-instance-02",
                                "instance-type",
                                "instance-owner",
                                "instance-category",
                                "instance-description",
                                "instance-lastPlugInOutput",
                                0,
                                0,
                                {3, props}};

  if (resource.status != SERVICE_OK) {
    fail("resource02.status != SERVICE_OK");
  }
  if (strcmp(resource.type, "instance-type")) {
    fail(resource.type);
  }
  if (strcmp(resource.properties.items[2].value.stringValue, "val_002")) {
    fail(resource.properties.items[2].value.stringValue);
  }
}

void test_encodeMonitoredResource() {
  MonitoredResource resource01 = {HOST_UP, "the-unique-name-of-the-instance-01",
                                  "gce_instance"};
  MonitoredResource resource02 = {
      SERVICE_OK,
      "the-unique-name-of-the-instance-02",
      "instance-type",
      "instance-owner",
      "instance-category",
      "instance-description",
      "instance-lastPlugInOutput",
      0,
      0,
      {3, (TypedValuePair[]){{"key--", {BooleanType, true}},
                             {"key_1", {DoubleType, doubleValue : 0.1}},
                             {"key-2", {StringType, stringValue : "val-2"}}}}};

  char *result = NULL;
  result = encodeMonitoredResource(&resource01, 0);
  if (!result || strcmp(result,
                        "{\"name\": \"the-unique-name-of-the-instance-01\", "
                        "\"status\": 7, \"type\": \"gce_instance\"}")) {
    fail("!result");
  }

  free(result);
  result = encodeMonitoredResource(&resource02, 0);

  char *expected =
      "{\"category\": \"instance-category\", \"description\": "
      "\"instance-description\", \"lastPlugInOutput\": "
      "\"instance-lastPlugInOutput\", \"name\": "
      "\"the-unique-name-of-the-instance-02\", \"owner\": \"instance-owner\", "
      "\"properties\": {\"key--\": {\"boolValue\": true, \"valueType\": 4}, "
      "\"key-2\": {\"stringValue\": \"val-2\", \"valueType\": 3}, \"key_1\": "
      "{\"doubleValue\": 0.10000000000000001, \"valueType\": 2}}, \"status\": "
      "1, \"type\": \"instance-type\"}";

  //   printf("\n#test_encodeMonitoredResource: %s", result);
  if (!result || strcmp(result, expected)) {
    fail("!result");
  }

  free(result);
}

void test_decodeMonitoredResource() {
  char *resource_str01 =
      "{\"name\": \"the-unique-name-of-the-instance-01\", "
      "\"status\": 7, \"type\": \"gce_instance\"}";

  char *resource_str02 =
      "{\"category\": \"instance-category\", \"description\": "
      "\"instance-description\", \"lastPlugInOutput\": "
      "\"instance-lastPlugInOutput\", \"name\": "
      "\"the-unique-name-of-the-instance-02\", \"owner\": \"instance-owner\", "
      "\"properties\": {\"key--\": {\"boolValue\": true, \"valueType\": 4}, "
      "\"key-2\": {\"stringValue\": \"val-2\", \"valueType\": 3}, \"key_1\": "
      "{\"doubleValue\": 0.10000000000000001, \"valueType\": 2}}, \"status\": "
      "1, \"type\": \"instance-type\"}";

  MonitoredResource *resource = decodeMonitoredResource(resource_str01);

  if (!resource) {
    fail("!resource");
  };
  if (strcmp(resource->name, "the-unique-name-of-the-instance-01")) {
    fail(resource->name);
  }
  if (strcmp(resource->type, "gce_instance")) {
    fail(resource->type);
  }
  if (resource->status != HOST_UP) {
    fail("resource->status != HOST_UP");
  }

  free(resource);
  resource = decodeMonitoredResource(resource_str02);

  if (!resource) {
    fail("!resource");
  };
  if (strcmp(resource->name, "the-unique-name-of-the-instance-02")) {
    fail(resource->name);
  }
  if (resource->status != SERVICE_OK) {
    fail("resource->status != SERVICE_OK");
  }
  if (resource->properties.count != 3) {
    fail("resource->properties.count");
  }
  if (strcmp(resource->properties.items[0].key, "key--")) {
    fail(resource->properties.items[0].key);
  }
  if (strcmp(resource->properties.items[1].key, "key-2")) {
    fail(resource->properties.items[1].key);
  }
  if (strcmp(resource->properties.items[2].key, "key_1")) {
    fail(resource->properties.items[2].key);
  }
  if (resource->properties.items[0].value.valueType != BooleanType) {
    fail("resource->properties.items[0].value.valueType != BooleanType");
  }
  if (resource->properties.items[0].value.boolValue != true) {
    fail("resource->properties.items[0].value.boolValue != true");
  }
  if (resource->properties.items[1].value.valueType != StringType) {
    fail("resource->properties.items[1].value.valueType != StringType");
  }
  if (strcmp(resource->properties.items[1].value.stringValue, "val-2")) {
    fail(resource->properties.items[1].value.stringValue);
  }
  if (resource->properties.items[2].value.valueType != DoubleType) {
    fail("resource->properties.items[2].value.valueType != DoubleType");
  }
  if (resource->properties.items[2].value.doubleValue != 0.1) {
    fail("resource->properties.items[2].value.doubleValue != 0.1");
  }

  free(resource);
}

int main(void) {
  test_defineMonitoredResource();
  test_encodeMonitoredResource();
  test_decodeMonitoredResource();

  fprintf(stdout, "\nall tests passed");
  return 0;
}
