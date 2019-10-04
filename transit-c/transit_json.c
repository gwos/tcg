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
  json_t *json, *jsonStatus, *jsonName, *jsonType, *jsonOwner, *jsonCategory,
      *jsonDescription, *jsonLastPlugInOutput, *jsonLastCheckTime,
      *jsonNextCheckTime, *jsonProperties, *jsonProp, *jsonPropValue;
  json = jsonStatus = jsonName = jsonType = jsonOwner = jsonCategory =
      jsonDescription = jsonLastPlugInOutput = jsonLastCheckTime =
          jsonNextCheckTime = jsonProperties = jsonProp = jsonPropValue = NULL;

  json = json_loads(str, 0, &error);
  if (json) {
    jsonStatus = json_object_get(json, "status");
    jsonName = json_object_get(json, "name");
    jsonType = json_object_get(json, "type");
    jsonOwner = json_object_get(json, "owner");
    jsonCategory = json_object_get(json, "category");
    jsonDescription = json_object_get(json, "description");
    jsonLastPlugInOutput = json_object_get(json, "lastPlugInOutput");
    jsonLastCheckTime = json_object_get(json, "lastCheckTime");
    jsonNextCheckTime = json_object_get(json, "nextCheckTime");
    jsonProperties = json_object_get(json, "properties");

    /* compute size for the target struct */
    size = sizeof(MonitoredResource);
    size += json_string_length(jsonName) + NUL_TERM_LEN;
    size += json_string_length(jsonType) + NUL_TERM_LEN;

    if (jsonOwner) {
      size += json_string_length(jsonOwner) + NUL_TERM_LEN;
    }
    if (jsonCategory) {
      size += json_string_length(jsonCategory) + NUL_TERM_LEN;
    }
    if (jsonDescription) {
      size += json_string_length(jsonDescription) + NUL_TERM_LEN;
    }
    if (jsonLastPlugInOutput) {
      size += json_string_length(jsonLastPlugInOutput) + NUL_TERM_LEN;
    }
    if (jsonProperties) {
      size += sizeof(TypedValuePairList);
      const char *key;
      json_object_foreach(jsonProperties, key, jsonProp) {
        size += strlen(key) + NUL_TERM_LEN;
        size += sizeof(TypedValuePair);
        size += sizeof(TypedValue);
        jsonPropValue = json_object_get(jsonProp, "stringValue");
        if (jsonPropValue) {
          size += json_string_length(jsonPropValue) + NUL_TERM_LEN;
        }
      }
    }

    /* allocate and fill the target struct by pointer */
    resource = (MonitoredResource *)malloc(size);
    MonitoredResource *ptr = resource;

    resource->status = json_integer_value(jsonStatus);
    resource->lastCheckTime = 0;
    resource->nextCheckTime = 0;
    if (jsonLastCheckTime) {
      resource->lastCheckTime = json_integer_value(jsonLastCheckTime);
    }
    if (jsonNextCheckTime) {
      resource->nextCheckTime = json_integer_value(jsonNextCheckTime);
    }

    ptr += sizeof(MonitoredResource);
    resource->name = strcpy((char *)ptr, json_string_value(jsonName));
    ptr += json_string_length(jsonName) + NUL_TERM_LEN;
    resource->type = strcpy((char *)ptr, json_string_value(jsonType));
    ptr += json_string_length(jsonType) + NUL_TERM_LEN;

    if (jsonOwner) {
      resource->owner = strcpy((char *)ptr, json_string_value(jsonOwner));
      ptr += json_string_length(jsonOwner) + NUL_TERM_LEN;
    }
    if (jsonCategory) {
      resource->owner = strcpy((char *)ptr, json_string_value(jsonCategory));
      ptr += json_string_length(jsonCategory) + NUL_TERM_LEN;
    }
    if (jsonDescription) {
      resource->owner = strcpy((char *)ptr, json_string_value(jsonDescription));
      ptr += json_string_length(jsonDescription) + NUL_TERM_LEN;
    }
    if (jsonLastPlugInOutput) {
      resource->owner =
          strcpy((char *)ptr, json_string_value(jsonLastPlugInOutput));
      ptr += json_string_length(jsonLastPlugInOutput) + NUL_TERM_LEN;
    }

    if (jsonProperties) {
      resource->properties.count = json_object_size(jsonProperties);
      resource->properties.items =
          (TypedValuePair *)(ptr + offsetof(TypedValuePairList, items));
      ptr += sizeof(TypedValuePairList);

      size_t i = 0;
      const char *key;
      TypedValue emptyTypedValue = {0, false, 0, 0, 0};
      json_object_foreach(jsonProperties, key, jsonProp) {
        ptr += sizeof(TypedValuePair);
        resource->properties.items[i].key = strcpy((char *)ptr, key);
        ptr += strlen(key) + NUL_TERM_LEN;

        memcpy(ptr, &emptyTypedValue, sizeof(TypedValue));
        ptr += sizeof(TypedValue);

        jsonPropValue = json_object_get(jsonProp, "valueType");
        VALUE_TYPE_ENUM valueType = json_integer_value(jsonPropValue);
        resource->properties.items[i].value.valueType = valueType;

        switch (valueType) {
          case IntegerType:
            jsonPropValue = json_object_get(jsonProp, "integerValue");
            resource->properties.items[i].value.integerValue =
                json_integer_value(jsonPropValue);
            break;

          case DoubleType:
            jsonPropValue = json_object_get(jsonProp, "doubleValue");
            resource->properties.items[i].value.doubleValue =
                json_real_value(jsonPropValue);
            break;

          case BooleanType:
            jsonPropValue = json_object_get(jsonProp, "boolValue");
            resource->properties.items[i].value.boolValue =
                json_boolean_value(jsonPropValue);
            break;

          case DateType:
            /* TODO: millis number */
            jsonPropValue = json_object_get(jsonProp, "dateValue");
            resource->properties.items[i].value.dateValue =
                json_integer_value(jsonPropValue);
            break;

          case StringType:
            jsonPropValue = json_object_get(jsonProp, "stringValue");
            resource->properties.items[i].value.stringValue =
                strcpy((char *)ptr, json_string_value(jsonPropValue));
            ptr += json_string_length(jsonPropValue) + NUL_TERM_LEN;
            break;

          default:
            break;
        }
        i++;
        json_decref(jsonPropValue);
      }
    }

    /*
        if (jsonOwner) {
          resource->owner = ptr;
          ptr += sizeof(MonitoredResource);
          resource->owner->name =
              strcpy((char *)ptr, json_string_value(jsonOwnerName));
          ptr += json_string_length(jsonOwnerName) + NUL_TERM_LEN;
          resource->owner->type =
              strcpy((char *)ptr, json_string_value(jsonOwnerType));
          ptr += json_string_length(jsonOwnerType) + NUL_TERM_LEN;
          resource->owner->status = indexOf(
              MONITOR_STATUS_STRING,
              sizeof(MONITOR_STATUS_STRING) / sizeof(*MONITOR_STATUS_STRING),
              json_string_value(jsonOwnerStatus));
        }

        if (jsonLabels) {
          resource->labels.count = json_object_size(jsonLabels);
          resource->labels.items =
              (StringPair *)(ptr + offsetof(StringPairs, items));
          ptr += sizeof(StringPairs);

          const char *key;
          json_t *value;
          size_t i = 0;
          json_object_foreach(jsonLabels, key, value) {
            ptr += sizeof(StringPair);
            resource->labels.items[i].key = strcpy((char *)ptr, key);
            ptr += strlen(key) + NUL_TERM_LEN;
            resource->labels.items[i].value =
                strcpy((char *)ptr, json_string_value(value));
            ptr += json_string_length(value) + NUL_TERM_LEN;
            i++;
          }
          json_decref(value);
        }
     */
  } else {
    fprintf(stderr, "decodeMonitoredResource error: %d: %s\n", error.line,
            error.text);
  }

  json_decref(json);
  json_decref(jsonStatus);
  json_decref(jsonName);
  json_decref(jsonType);
  json_decref(jsonOwner);
  json_decref(jsonCategory);
  json_decref(jsonDescription);
  json_decref(jsonLastPlugInOutput);
  json_decref(jsonLastCheckTime);
  json_decref(jsonNextCheckTime);
  json_decref(jsonProperties);
  json_decref(jsonProp);
  json_decref(jsonPropValue);

  return resource;
}

char *encodeMonitoredResource(const MonitoredResource *resource, size_t flags) {
  char *result;
  json_t *json = json_object();
  json_t *jsonProperties = json_object();

  json_object_set_new(json, "name", json_string(resource->name));
  json_object_set_new(json, "type", json_string(resource->type));
  /* number */
  json_object_set_new(json, "status", json_integer(resource->status));
  if (resource->owner) {
    json_object_set_new(json, "owner", json_string(resource->owner));
  }
  if (resource->lastCheckTime) {
    /* TODO: millis number */
    json_object_set_new(json, "lastCheckTime",
                        json_integer(resource->lastCheckTime));
  }
  if (resource->nextCheckTime) {
    /* TODO: millis number */
    json_object_set_new(json, "nextCheckTime",
                        json_integer(resource->nextCheckTime));
  }
  if (resource->lastPlugInOutput) {
    json_object_set_new(json, "lastPlugInOutput",
                        json_string(resource->lastPlugInOutput));
  }
  if (resource->category) {
    json_object_set_new(json, "category", json_string(resource->category));
  }
  if (resource->description) {
    json_object_set_new(json, "description",
                        json_string(resource->description));
  }
  if (resource->properties.count) {  // go:map[string]TypedValue
    json_object_set_new(json, "properties", jsonProperties);

    for (size_t i = 0; i < resource->properties.count; i++) {
      json_t *jsonProp = json_object();

      json_object_set_new(
          jsonProp, "valueType",
          json_integer(resource->properties.items[i].value.valueType));

      switch (resource->properties.items[i].value.valueType) {
        case IntegerType:
          json_object_set_new(
              jsonProp, "integerValue",
              json_integer(resource->properties.items[i].value.integerValue));
          break;

        case DoubleType:
          json_object_set_new(
              jsonProp, "doubleValue",
              json_real(resource->properties.items[i].value.doubleValue));
          break;

        case StringType:
          json_object_set_new(
              jsonProp, "stringValue",
              json_string(resource->properties.items[i].value.stringValue));
          break;

        case BooleanType:
          json_object_set_new(
              jsonProp, "boolValue",
              json_boolean(resource->properties.items[i].value.boolValue));
          break;

        case DateType:
          /* TODO: millis number */
          json_object_set_new(
              jsonProp, "dateValue",
              json_integer(resource->properties.items[i].value.dateValue));
          break;

        default:
          break;
      }

      json_object_set(jsonProperties, resource->properties.items[i].key,
                      jsonProp);
      json_decref(jsonProp);
    }
  }

  if (!flags) {
    flags = JSON_SORT_KEYS;
  }
  result = json_dumps(json, flags);
  json_decref(jsonProperties);
  json_decref(json);
  return result;
}


