#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "transit.h"
#include "transit_json.h"
#include "util.h"

void test_defineMonitoredResource() {
  MonitoredResource resource01 = {"the-unique-name-of-the-instance-01",
                                  "gce_instance", HOST_UP};

  StringPair labels[] = {
      {"key_1", "val_1"}, {"key_02", "val_02"}, {"key_003", "val_003"}};

  MonitoredResource resource02 = {"the-unique-name-of-the-instance-02",
                                  "gce_instance",
                                  SERVICE_OK,
                                  &resource01,
                                  {3, labels}};

  if (strcmp(resource02.type, "gce_instance")) {
    fail(resource02.type);
  }
  if (strcmp(MONITOR_STATUS_STRING[resource02.status], "SERVICE_OK")) {
    fail(MONITOR_STATUS_STRING[resource02.status]);
  }
  if (strcmp(resource02.labels.items[1].value, "val_02")) {
    fail(resource02.labels.items[1].value);
  }
  if (strcmp(resource02.owner->type, "gce_instance")) {
    fail(resource02.owner->type);
  }
  if (strcmp(MONITOR_STATUS_STRING[resource02.owner->status], "HOST_UP")) {
    fail(MONITOR_STATUS_STRING[resource02.owner->status]);
  }
}

void test_encodeMonitoredResource() {
  MonitoredResource resource01 = {"the-unique-name-of-the-instance-01",
                                  "gce_instance", HOST_UP};

  MonitoredResource resource02 = {"the-unique-name-of-the-instance-02",
                                  "gce_instance",
                                  SERVICE_OK,
                                  &resource01,
                                  {3, (StringPair[]){{"key_1", "val_1"},
                                                     {"key_02", "val_02"},
                                                     {"key_003", "val_003"}}}};

  char *result = NULL;
  result = encodeMonitoredResource(&resource01, 0);
  if (!result ||
      strcmp(result,
             "{\"name\": \"the-unique-name-of-the-instance-01\", \"status\": "
             "\"HOST_UP\", \"type\": \"gce_instance\"}")) {
    fail("!result");
  }

  free(result);
  result = encodeMonitoredResource(&resource02, 0);

  char *expected =
      "{\"labels\": {\"key_003\": \"val_003\", \"key_02\": \"val_02\", "
      "\"key_1\": \"val_1\"}, \"name\": "
      "\"the-unique-name-of-the-instance-02\", \"owner\": {\"name\": "
      "\"the-unique-name-of-the-instance-01\", \"status\": \"HOST_UP\", "
      "\"type\": \"gce_instance\"}, \"status\": \"SERVICE_OK\", \"type\": "
      "\"gce_instance\"}";

  //   printf("\n#test_encodeMonitoredResource: %s", result);
  if (!result || strcmp(result, expected)) {
    fail("!result");
  }

  free(result);
}

void test_decodeMonitoredResource() {
  char *resource_str01 =
      "{\"name\": \"the-unique-name-of-the-instance-01\", \"status\": "
      "\"HOST_UP\", \"type\": \"gce_instance\"}";

  char *resource_str02 =
      "{\"labels\": {\"key_003\": \"val_003\", \"key_02\": \"val_02\", "
      "\"key_1\": \"val_1\"}, \"name\": "
      "\"the-unique-name-of-the-instance-02\", \"owner\": {\"name\": "
      "\"the-unique-name-of-the-instance-01\", \"status\": \"HOST_UP\", "
      "\"type\": \"gce_instance\"}, \"status\": \"SERVICE_OK\", \"type\": "
      "\"gce_instance\"}";

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
  if (resource->labels.count != 3) {
    fail("resource->labels.count");
  }
  if (strcmp(resource->labels.items[0].key, "key_003")) {
    fail(resource->labels.items[0].key);
  }
  if (strcmp(resource->labels.items[1].key, "key_02")) {
    fail(resource->labels.items[1].key);
  }
  if (strcmp(resource->labels.items[2].key, "key_1")) {
    fail(resource->labels.items[2].key);
  }
  if (strcmp(resource->labels.items[0].value, "val_003")) {
    fail(resource->labels.items[0].value);
  }
  if (strcmp(resource->labels.items[1].value, "val_02")) {
    fail(resource->labels.items[1].value);
  }
  if (strcmp(resource->labels.items[2].value, "val_1")) {
    fail(resource->labels.items[2].value);
  }
  if (strcmp(resource->name, "the-unique-name-of-the-instance-02")) {
    fail(resource->name);
  }
  if (strcmp(resource->owner->name, "the-unique-name-of-the-instance-01")) {
    fail(resource->owner->name);
  }
  if (resource->status != SERVICE_OK) {
    fail("resource->status != SERVICE_OK");
  }
  if (resource->owner->status != HOST_UP) {
    fail("resource->owner->status != HOST_UP");
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
