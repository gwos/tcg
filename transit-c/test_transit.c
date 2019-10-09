#include <dlfcn.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "libtransit.h"
#include "transit.h"
#include "transit_json.h"
#include "util.h"

void test_defineMonitoredResource() {
  TypedValue prop0 = {BooleanType, boolValue : true};
  TypedValue prop1 = {DoubleType, doubleValue : 0.1};
  TypedValue prop2 = {StringType, stringValue : "val_002"};
  TypedValuePair props[] = {
      {"key0", prop0}, {"key01", prop1}, {"key002", prop2}};

  MonitoredResource resource = {
    status : SERVICE_OK,
    name : "the-unique-name-of-the-instance-02",
    type : "instance-type",
    owner : "instance-owner",
    category : "instance-category",
    description : "instance-description",
    lastPlugInOutput : "instance-lastPlugInOutput",
    lastCheckTime : 0,
    nextCheckTime : 0,
    properties : {count : 3, items : props}
  };

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
    status : SERVICE_OK,
    name : "the-unique-name-of-the-instance-02",
    type : "instance-type",
    owner : "instance-owner",
    category : "instance-category",
    description : "instance-description",
    lastPlugInOutput : "instance-lastPlugInOutput",
    lastCheckTime : 0,
    nextCheckTime : 0,
    properties : {
      count : 3,
      items :
          (TypedValuePair[]){
	      {"key--", {BooleanType, boolValue : true}},
              {"key_1", {DoubleType, doubleValue : 0.1}},
              {"key-2", {StringType, stringValue : "val-2"}},
	  }
    }
  };

  char *result = NULL;
  result = encodeMonitoredResource(&resource01, 0);
  if (!result || strcmp(result,
                        "{\"name\": \"the-unique-name-of-the-instance-01\", "
                        "\"status\": 7, \"type\": \"gce_instance\"}")) {
    fail(result);
  }

  free(result);
  result = encodeMonitoredResource(&resource02, 0);

  char *expected = "{"
      "\"category\": \"instance-category\", "
      "\"description\": \"instance-description\", "
      "\"lastPlugInOutput\": \"instance-lastPlugInOutput\", "
      "\"name\": \"the-unique-name-of-the-instance-02\", "
      "\"owner\": \"instance-owner\", "
      "\"properties\": {"
	  "\"key--\": {"
	      "\"boolValue\": true, "
	      "\"valueType\": 4"
	  "}, "
	  "\"key-2\": {"
	      "\"stringValue\": \"val-2\", "
	      "\"valueType\": 3"
	  "}, "
	  "\"key_1\": {"
	      "\"doubleValue\": 0.10000000000000001, "
	      "\"valueType\": 2"
	  "}"
      "}, "
      "\"status\": " "1, "
      "\"type\": \"instance-type\""
  "}";

  // printf("#test_encodeMonitoredResource: %s\n", result);
  if (!result || strcmp(result, expected)) {
    fail(result);
  }

  free(result);
}

void test_decodeMonitoredResource() {
  char *resource_json01 =
      "{\"name\": \"the-unique-name-of-the-instance-01\", "
      "\"status\": 7, \"type\": \"gce_instance\"}";

  char *resource_json02 = "{"
      "\"category\": \"instance-category\", "
      "\"description\": \"instance-description\", "
      "\"lastPlugInOutput\": \"instance-lastPlugInOutput\", "
      "\"name\": \"the-unique-name-of-the-instance-02\", "
      "\"owner\": \"instance-owner\", "
      "\"properties\": {"
	  "\"key--\": {"
	      "\"boolValue\": true, "
	      "\"valueType\": 4"
	  "}, "
	  "\"key-2\": {"
	      "\"stringValue\": \"val-2\", "
	      "\"valueType\": 3"
	  "}, "
	  "\"key_1\": {"
	      "\"doubleValue\": 0.10000000000000001, "
	      "\"valueType\": 2"
	  "}"
      "}, "
      "\"status\": 1, "
      "\"type\": \"instance-type\""
  "}";

  MonitoredResource *resource = decodeMonitoredResource(resource_json01);

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
  resource = decodeMonitoredResource(resource_json02);

  if (!resource) {
    fail("!resource");
  };
  if (resource->status != SERVICE_OK) {
    fail("resource->status != SERVICE_OK");
  }
  if (strcmp(resource->name, "the-unique-name-of-the-instance-02")) {
    fail(resource->name);
  }
  if (strcmp(resource->type, "instance-type")) {
    fail(resource->type);
  }
  if (strcmp(resource->owner, "instance-owner")) {
    fail(resource->owner);
  }
  if (strcmp(resource->category, "instance-category")) {
    fail(resource->category);
  }
  if (strcmp(resource->description, "instance-description")) {
    fail(resource->description);
  }
  if (strcmp(resource->lastPlugInOutput, "instance-lastPlugInOutput")) {
    fail(resource->lastPlugInOutput);
  }
  if (resource->lastCheckTime != 0) {
    fail("resource->lastCheckTime != 0");
  }
  if (resource->nextCheckTime != 0) {
    fail("resource->nextCheckTime != 0");
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

void test_encodeCredentials() {
  Credentials creds = {"Username", "SecurePass"};
  char *result = NULL;
  result = encodeCredentials(&creds, 0);
  //   printf("\n#test_encodeCredentials: %s", result);
  if (!result ||
      strcmp(result,
             "{\"password\": \"SecurePass\", \"user\": \"Username\"}")) {
    fail(result);
  }
  free(result);
}

void test_decodeCredentials() {
  char *creds_json = "{\"password\": \"SecurePass\", \"user\": \"Username\"}";
  Credentials *creds = decodeCredentials(creds_json);

  if (!creds) {
    fail("!creds");
  };
  if (strcmp(creds->user, "Username")) {
    fail(creds->user);
  }
  if (strcmp(creds->password, "SecurePass")) {
    fail(creds->password);
  }

  free(creds);
}

void test_encodeTransit() {
  Transit transit = {
    config : {
      account : "Account",
      hostName : "host-name",
      token : "token-token",
      ssl : true
    }
  };
  char *result = NULL;
  result = encodeTransit(&transit, 0);
  //   printf("\n#test_encodeTransit: %s", result);
  if (!result ||
      strcmp(result,
             "{\"config\": {\"account\": \"Account\", \"hostName\": "
             "\"host-name\", \"ssl\": true, \"token\": \"token-token\"}}")) {
    fail(result);
  }
  free(result);
}

void test_decodeTransit() {
  char *transit_json =
      "{\"config\": {\"account\": \"Account\", \"hostName\": "
      "\"host-name\", \"ssl\": true, \"token\": \"token-token\"}}";
  Transit *transit = decodeTransit(transit_json);

  if (!transit) {
    fail("!transit");
  };
  if (strcmp(transit->config.account, "Account")) {
    fail(transit->config.account);
  }
  if (strcmp(transit->config.hostName, "host-name")) {
    fail(transit->config.hostName);
  }
  if (strcmp(transit->config.token, "token-token")) {
    fail(transit->config.token);
  }
  if (transit->config.ssl != true) {
    fail("transit->config.ssl != true");
  }

  free(transit);
}

int main(void) {
  printf("<<< TESTING test_defineMonitoredResource >>>\n");
  test_defineMonitoredResource();

  printf("<<< TESTING test_encodeMonitoredResource >>>\n");
  test_encodeMonitoredResource();

  printf("<<< TESTING test_decodeMonitoredResource >>>\n");
  test_decodeMonitoredResource();

  printf("<<< TESTING test_encodeCredentials >>>\n");
  test_encodeCredentials();

  printf("<<< TESTING test_decodeCredentials >>>\n");
  test_decodeCredentials();

  printf("<<< TESTING test_encodeTransit >>>\n");
  test_encodeTransit();

  printf("<<< TESTING test_decodeTransit >>>\n");
  test_decodeTransit();

  printf("all tests passed\n");

  return 0;
}
