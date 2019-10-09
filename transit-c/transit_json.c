#include <jansson.h>
#include <stdalign.h>  // Needed to supply alignof(), available starting with C11.
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

Credentials *decodeCredentials(const char *json_str) {
  Credentials *credentials = NULL;
  size_t size = 0;
  json_error_t error;
  json_t *json = NULL;

  json = json_loads(json_str, 0, &error);
  if (json) {
    json_t *jsonUser = json_object_get(json, "user");
    json_t *jsonPass = json_object_get(json, "password");
    size_t jsonUser_len = json_string_length(jsonUser);
    size_t jsonPass_len = json_string_length(jsonPass);

    // incrementally compute a total size for allocation of the
    // target struct, including all the strings it refers to
    size = sizeof(Credentials);
    size += jsonUser_len + NUL_TERM_LEN;
    size += jsonPass_len + NUL_TERM_LEN;

    /* allocate and fill the target struct by pointer */
    credentials = (Credentials *)malloc(size);
    if (credentials) {
      char *ptr = (char *)credentials;
      ptr += sizeof(Credentials);
      credentials->user = strcpy(ptr, json_string_value(jsonUser));
      ptr += jsonUser_len + NUL_TERM_LEN;
      credentials->password = strcpy(ptr, json_string_value(jsonPass));
    } else {
      fprintf(stderr, "decodeCredentials error: %s\n", "malloc failed");
    }

    json_decref(json);
  } else {
    fprintf(stderr, "decodeCredentials error: %d: %s\n", error.line,
            error.text);
  }

  return credentials;
};

MonitoredResource *decodeMonitoredResource(const char *json_str) {
  MonitoredResource *resource = NULL;
  size_t size = 0;
  json_error_t error;
  json_t *json = NULL;
  json_t *jsonProp = NULL;
  json_t *jsonPropValue = NULL;

  json = json_loads(json_str, 0, &error);
  if (json) {
    json_t *jsonStatus = json_object_get(json, "status");
    json_t *jsonName = json_object_get(json, "name");
    json_t *jsonType = json_object_get(json, "type");
    json_t *jsonOwner = json_object_get(json, "owner");
    json_t *jsonCategory = json_object_get(json, "category");
    json_t *jsonDescription = json_object_get(json, "description");
    json_t *jsonLastPlugInOutput = json_object_get(json, "lastPlugInOutput");
    json_t *jsonLastCheckTime = json_object_get(json, "lastCheckTime");
    json_t *jsonNextCheckTime = json_object_get(json, "nextCheckTime");
    json_t *jsonProperties = json_object_get(json, "properties");

    size_t jsonName_len = json_string_length(jsonName);
    size_t jsonType_len = json_string_length(jsonType);

    size_t jsonOwner_len;
    size_t jsonCategory_len;
    size_t jsonDescription_len;
    size_t jsonLastPlugInOutput_len;
    size_t TypedValuePair_alignment = alignof(TypedValuePair);
    size_t TypedValuePair_padding;

    // incrementally compute a total size for allocation of the
    // target struct, including all the strings and other objects
    // it refers to
    size = sizeof(MonitoredResource);
    size += jsonName_len + NUL_TERM_LEN;
    size += jsonType_len + NUL_TERM_LEN;

    if (jsonOwner) {
      jsonOwner_len = json_string_length(jsonOwner);
      size += jsonOwner_len + NUL_TERM_LEN;
    }
    if (jsonCategory) {
      jsonCategory_len = json_string_length(jsonCategory);
      size += jsonCategory_len + NUL_TERM_LEN;
    }
    if (jsonDescription) {
      jsonDescription_len = json_string_length(jsonDescription);
      size += jsonDescription_len + NUL_TERM_LEN;
    }
    if (jsonLastPlugInOutput) {
      jsonLastPlugInOutput_len = json_string_length(jsonLastPlugInOutput);
      size += jsonLastPlugInOutput_len + NUL_TERM_LEN;
    }
    if (jsonProperties) {
      // Here we have to be very careful, to account for padding needed to
      // align the additional structures in memory so their internal fields
      // will naturally align on proper memory boundaries.

      // Calculations such as this one must use only positive operands
      // for the % operator, to avoid implementation-defined behavior.
      // The clever expression is written to provide no padding bytes
      // if the size is already aligned.

      TypedValuePair_padding =
          ((TypedValuePair_alignment - 1) * size) % TypedValuePair_alignment;
      size += TypedValuePair_padding +
              json_object_size(jsonProperties) * sizeof(TypedValuePair);

      const char *key;
      json_object_foreach(jsonProperties, key, jsonProp) {
        size += strlen(key) + NUL_TERM_LEN;
        jsonPropValue = json_object_get(jsonProp, "stringValue");
        if (jsonPropValue) {
          size += json_string_length(jsonPropValue) + NUL_TERM_LEN;
        }
      }
    }

    /* allocate and fill the target struct by pointer */
    resource = (MonitoredResource *)malloc(size);
    if (resource) {
      char *ptr = (char *)resource;

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
      resource->name = strcpy(ptr, json_string_value(jsonName));
      ptr += jsonName_len + NUL_TERM_LEN;
      resource->type = strcpy(ptr, json_string_value(jsonType));
      ptr += jsonType_len + NUL_TERM_LEN;

      if (jsonOwner) {
        resource->owner = strcpy(ptr, json_string_value(jsonOwner));
        ptr += jsonOwner_len + NUL_TERM_LEN;
      }
      if (jsonCategory) {
        resource->category = strcpy(ptr, json_string_value(jsonCategory));
        ptr += jsonCategory_len + NUL_TERM_LEN;
      }
      if (jsonDescription) {
        resource->description = strcpy(ptr, json_string_value(jsonDescription));
        ptr += jsonDescription_len + NUL_TERM_LEN;
      }
      if (jsonLastPlugInOutput) {
        resource->lastPlugInOutput =
            strcpy(ptr, json_string_value(jsonLastPlugInOutput));
        ptr += jsonLastPlugInOutput_len + NUL_TERM_LEN;
      }

      if (jsonProperties) {
        ptr += TypedValuePair_padding;
        resource->properties.count = json_object_size(jsonProperties);
        resource->properties.items = (TypedValuePair *)ptr;
        ptr += json_object_size(jsonProperties) * sizeof(TypedValuePair);

        size_t i = 0;
        const char *key;
        TypedValue emptyTypedValue = {0, false, 0, 0, 0};
        json_object_foreach(jsonProperties, key, jsonProp) {
          resource->properties.items[i].key = strcpy(ptr, key);
          ptr += strlen(key) + NUL_TERM_LEN;
          memcpy(&resource->properties.items[i].value, &emptyTypedValue,
                 sizeof(TypedValue));

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
                  strcpy(ptr, json_string_value(jsonPropValue));
              ptr += json_string_length(jsonPropValue) + NUL_TERM_LEN;
              break;

            default:
              break;
          }
          i++;
        }
      }
    } else {
      fprintf(stderr, "decodeMonitoredResource error: %s\n", "malloc failed");
    }

    json_decref(json);
  } else {
    fprintf(stderr, "decodeMonitoredResource error: %d: %s\n", error.line,
            error.text);
  }

  return resource;
}

Transit *decodeTransit(const char *json_str) {
  Transit *transit = NULL;
  size_t size = 0;
  json_error_t error;
  json_t *json = NULL;

  json = json_loads(json_str, 0, &error);
  if (json) {
    json_t *jsonCfg = json_object_get(json, "config");
    json_t *jsonCfgHostName = json_object_get(jsonCfg, "hostName");
    json_t *jsonCfgAccount = json_object_get(jsonCfg, "account");
    json_t *jsonCfgToken = json_object_get(jsonCfg, "token");
    json_t *jsonCfgSSL = json_object_get(jsonCfg, "ssl");

    size_t jsonCfgHostName_len = json_string_length(jsonCfgHostName);
    size_t jsonCfgAccount_len = json_string_length(jsonCfgAccount);
    size_t jsonCfgToken_len = json_string_length(jsonCfgToken);

    // incrementally compute a total size for allocation of the
    // target struct, including all the strings it refers to
    size = sizeof(Transit);
    size += jsonCfgHostName_len + NUL_TERM_LEN;
    size += jsonCfgAccount_len + NUL_TERM_LEN;
    size += jsonCfgToken_len + NUL_TERM_LEN;

    /* allocate and fill the target struct by pointer */
    transit = (Transit *)malloc(size);
    if (transit) {
      char *ptr = (char *)transit;
      ptr += sizeof(Transit);
      transit->config.hostName =
          strcpy(ptr, json_string_value(jsonCfgHostName));
      ptr += jsonCfgHostName_len + NUL_TERM_LEN;
      transit->config.account = strcpy(ptr, json_string_value(jsonCfgAccount));
      ptr += jsonCfgAccount_len + NUL_TERM_LEN;
      transit->config.token = strcpy(ptr, json_string_value(jsonCfgToken));
      transit->config.ssl = json_boolean_value(jsonCfgSSL);
    } else {
      fprintf(stderr, "decodeTransit error: %s\n", "malloc failed");
    }

    json_decref(json);
  } else {
    fprintf(stderr, "decodeTransit error: %d: %s\n", error.line, error.text);
  }

  return transit;
};

char *encodeCredentials(const Credentials *credentials, size_t flags) {
  char *result;
  json_t *json = json_object();

  json_object_set_new(json, "user", json_string(credentials->user));
  json_object_set_new(json, "password", json_string(credentials->password));

  if (!flags) {
    flags = JSON_SORT_KEYS;
  }
  result = json_dumps(json, flags);
  json_decref(json);
  return result;
};

char *encodeMonitoredResource(const MonitoredResource *resource, size_t flags) {
  char *result;
  json_t *json = json_object();
  json_t *jsonProperties = json_object();
  json_t *jsonProp = json_object();

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
    json_object_set(json, "properties", jsonProperties);
    for (size_t i = 0; i < resource->properties.count; i++) {
      json_object_clear(jsonProp);
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

      json_object_set_new(jsonProperties, resource->properties.items[i].key,
                          json_deep_copy(jsonProp));
    }
  }

  if (!flags) {
    flags = JSON_SORT_KEYS;
  }
  result = json_dumps(json, flags);
  json_decref(json);
  json_decref(jsonProperties);
  json_decref(jsonProp);
  return result;
}

char *encodeTransit(const Transit *transit, size_t flags) {
  char *result;
  json_t *json = json_object();
  json_t *jsonConfig = json_object();

  json_object_set(json, "config", jsonConfig);
  json_object_set_new(jsonConfig, "account",
                      json_string(transit->config.account));
  json_object_set_new(jsonConfig, "hostName",
                      json_string(transit->config.hostName));
  json_object_set_new(jsonConfig, "token", json_string(transit->config.token));
  json_object_set_new(jsonConfig, "ssl", json_boolean(transit->config.ssl));

  if (!flags) {
    flags = JSON_SORT_KEYS;
  }
  result = json_dumps(json, flags);
  json_decref(json);
  json_decref(jsonConfig);
  return result;
};
