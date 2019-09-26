#include <jansson.h>
#include <stddef.h>
#include <string.h>

#include "transit_json.h"

int indexOf(const char **arr, size_t len, const char *target) {
  for (size_t i = 0; i < len; i++) {
    if (strncmp(arr[i], target, strlen(target)) == 0) {
      return i;
    }
  }
  return -1;
}

MonitoredResource *decodeMonitoredResource(const char *str) {
  MonitoredResource *resource = NULL;
  size_t size = 0;
  json_error_t error;
  json_t *json, *jsonName, *jsonType, *jsonStatus, *jsonLabels, *jsonOwner,
      *jsonOwnerName, *jsonOwnerType, *jsonOwnerStatus;
  json = jsonName = jsonType = jsonStatus = jsonLabels = jsonOwner =
      jsonOwnerName = jsonOwnerType = jsonOwnerStatus = NULL;

  json = json_loads(str, 0, &error);
  if (json) {
    jsonName = json_object_get(json, "name");
    jsonType = json_object_get(json, "type");
    jsonStatus = json_object_get(json, "status");

    /* compute size for the target struct */
    size = sizeof(MonitoredResource);
    size += json_string_length(jsonName) + 1;
    size += json_string_length(jsonType) + 1;

    jsonOwner = json_object_get(json, "owner");
    if (jsonOwner) {
      jsonOwnerName = json_object_get(jsonOwner, "name");
      jsonOwnerType = json_object_get(jsonOwner, "type");
      jsonOwnerStatus = json_object_get(jsonOwner, "status");

      size += sizeof(MonitoredResource);
      size += json_string_length(jsonOwnerName) + 1;
      size += json_string_length(jsonOwnerType) + 1;
    }

    jsonLabels = json_object_get(json, "labels");
    if (jsonLabels) {
      const char *key;
      json_t *value;
      size += sizeof(StringPairs);
      json_object_foreach(jsonLabels, key, value) {
        size += sizeof(StringPair);
        size += strlen(key) + 1;
        size += json_string_length(value) + 1;
      }
    }

    /* allocate and fill the target struct by pointer */
    resource = (MonitoredResource *)malloc(size);
    MonitoredResource *ptr = resource;

    ptr += sizeof(MonitoredResource);
    resource->name = strcpy((char *)ptr, json_string_value(jsonName));
    ptr += json_string_length(jsonName) + 1;
    resource->type = strcpy((char *)ptr, json_string_value(jsonType));
    ptr += json_string_length(jsonType) + 1;
    resource->status =
        indexOf(MONITOR_STATUS_STRING,
                sizeof(MONITOR_STATUS_STRING) / sizeof(*MONITOR_STATUS_STRING),
                json_string_value(jsonStatus));
    if (jsonOwner) {
      resource->owner = ptr;
      ptr += sizeof(MonitoredResource);
      resource->owner->name = strcpy((char *)ptr, json_string_value(jsonOwnerName));
      ptr += json_string_length(jsonOwnerName) + 1;
      resource->owner->type = strcpy((char *)ptr, json_string_value(jsonOwnerType));
      ptr += json_string_length(jsonOwnerType) + 1;
      resource->owner->status = indexOf(
          MONITOR_STATUS_STRING,
          sizeof(MONITOR_STATUS_STRING) / sizeof(*MONITOR_STATUS_STRING),
          json_string_value(jsonOwnerStatus));
    }
    if (jsonLabels) {
      resource->labels.count = json_object_size(jsonLabels);
      resource->labels.items = (StringPair *)(ptr + offsetof(StringPairs, items));
      ptr += sizeof(StringPairs);

      const char *key;
      json_t *value;
      size_t i = 0;
      json_object_foreach(jsonLabels, key, value) {
        ptr += sizeof(StringPair);
        resource->labels.items[i].key = strcpy((char *)ptr, key);
        ptr += strlen(key) + 1;
        resource->labels.items[i].value = strcpy((char *)ptr, json_string_value(value));
        ptr += json_string_length(value) + 1;
        i++;
      }
      json_decref(value);
    }
  } else {
    fprintf(stderr, "decodeMonitoredResource error: %d: %s\n", error.line,
            error.text);
  }

  json_decref(json);
  json_decref(jsonName);
  json_decref(jsonType);
  json_decref(jsonStatus);
  json_decref(jsonLabels);
  json_decref(jsonOwner);
  json_decref(jsonOwnerName);
  json_decref(jsonOwnerType);
  json_decref(jsonOwnerStatus);
  return resource;
}

char *encodeMonitoredResource(const MonitoredResource *resource, size_t flags) {
  char *result;
  json_t *json = json_object();
  json_t *json_owner = json_object();
  json_t *json_labels = json_object();

  json_object_set_new(json, "name", json_string(resource->name));
  json_object_set_new(json, "type", json_string(resource->type));
  json_object_set_new(json, "status",
                      json_string(MONITOR_STATUS_STRING[resource->status]));

  if (resource->owner) {
    json_object_set_new(json, "owner", json_owner);
    json_object_set_new(json_owner, "name", json_string(resource->owner->name));
    json_object_set_new(json_owner, "type", json_string(resource->owner->type));
    json_object_set_new(
        json_owner, "status",
        json_string(MONITOR_STATUS_STRING[resource->owner->status]));
  }
  if (resource->labels.count) {
    json_object_set_new(json, "labels", json_labels);
    for (size_t i = 0; i < resource->labels.count; i++) {
      json_object_set_new(json_labels, resource->labels.items[i].key,
                          json_string(resource->labels.items[i].value));
    }
  }

  if (!flags) {
    flags = JSON_SORT_KEYS;
  }
  result = json_dumps(json, flags);
  json_decref(json_owner);
  json_decref(json_labels);
  json_decref(json);
  return result;
}
