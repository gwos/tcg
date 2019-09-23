#include <jansson.h>

#include "transit_json.h"

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
