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

Credentials *decodeCredentials(const char *json_str) {
  Credentials *credentials = NULL;
  size_t size = 0;
  json_error_t error;
  json_t *json, *jsonUser, *jsonPass;
  json = jsonUser = jsonPass = NULL;

  json = json_loads(json_str, 0, &error);
  if (json) {
    jsonUser = json_object_get(json, "user");
    jsonPass = json_object_get(json, "password");

    /* compute size for the target struct */
    size = sizeof(Credentials);
    size += json_string_length(jsonUser) + NUL_TERM_LEN;
    size += json_string_length(jsonPass) + NUL_TERM_LEN;

    /* allocate and fill the target struct by pointer */
    credentials = malloc(size);
    if (credentials) {
      Credentials *ptr = credentials;
      ptr += sizeof(Credentials);
      credentials->user = strcpy((char *)ptr, json_string_value(jsonUser));
      ptr += json_string_length(jsonUser) + NUL_TERM_LEN;
      credentials->password = strcpy((char *)ptr, json_string_value(jsonPass));
    } else {
      fprintf(stderr, "decodeCredentials error: %s\n", "malloc failed");
    }
  } else {
    fprintf(stderr, "decodeCredentials error: %d: %s\n", error.line,
            error.text);
  }

  json_decref(json);
  json_decref(jsonUser);
  json_decref(jsonPass);
  return credentials;
};

MonitoredResource *decodeMonitoredResource(const char *json_str) {
  MonitoredResource *resource = NULL;
  size_t size = 0;
  json_error_t error;
  json_t *json, *jsonStatus, *jsonName, *jsonType, *jsonOwner, *jsonCategory,
      *jsonDescription, *jsonLastPlugInOutput, *jsonLastCheckTime,
      *jsonNextCheckTime, *jsonProperties, *jsonProp, *jsonPropValue;
  json = jsonStatus = jsonName = jsonType = jsonOwner = jsonCategory =
      jsonDescription = jsonLastPlugInOutput = jsonLastCheckTime =
          jsonNextCheckTime = jsonProperties = jsonProp = jsonPropValue = NULL;

  json = json_loads(json_str, 0, &error);
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
          json_decref(jsonPropValue);
        }
      }
    }

    /* allocate and fill the target struct by pointer */
    resource = malloc(size);
    if (resource) {
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
        resource->category =
            strcpy((char *)ptr, json_string_value(jsonCategory));
        ptr += json_string_length(jsonCategory) + NUL_TERM_LEN;
      }
      if (jsonDescription) {
        resource->description =
            strcpy((char *)ptr, json_string_value(jsonDescription));
        ptr += json_string_length(jsonDescription) + NUL_TERM_LEN;
      }
      if (jsonLastPlugInOutput) {
        resource->lastPlugInOutput =
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
          json_decref(jsonPropValue);

          printf("\n #i:valueType %ld : %d", i, valueType);

          switch (valueType) {
            case IntegerType:
              jsonPropValue = json_object_get(jsonProp, "integerValue");
              resource->properties.items[i].value.integerValue =
                  json_integer_value(jsonPropValue);
              json_decref(jsonPropValue);
              break;

            case DoubleType:
              jsonPropValue = json_object_get(jsonProp, "doubleValue");
              resource->properties.items[i].value.doubleValue =
                  json_real_value(jsonPropValue);
              json_decref(jsonPropValue);
              break;

            case BooleanType:
              jsonPropValue = json_object_get(jsonProp, "boolValue");
              resource->properties.items[i].value.boolValue =
                  json_boolean_value(jsonPropValue);
              json_decref(jsonPropValue);
              break;

            case DateType:
              /* TODO: millis number */
              jsonPropValue = json_object_get(jsonProp, "dateValue");
              resource->properties.items[i].value.dateValue =
                  json_integer_value(jsonPropValue);
              json_decref(jsonPropValue);
              break;

            case StringType:
              jsonPropValue = json_object_get(jsonProp, "stringValue");
              resource->properties.items[i].value.stringValue =
                  strcpy((char *)ptr, json_string_value(jsonPropValue));
              ptr += json_string_length(jsonPropValue) + NUL_TERM_LEN;
              json_decref(jsonPropValue);
              break;

            default:
              break;
          }
          i++;
          //   json_decref(jsonPropValue);
        }
      }
    } else {
      fprintf(stderr, "decodeMonitoredResource error: %s\n", "malloc failed");
    }
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
  //   json_decref(jsonProp);
  //   json_decref(jsonPropValue);

  return resource;
}

Transit *decodeTransit(const char *json_str) {
  Transit *transit = NULL;
  size_t size = 0;
  json_error_t error;
  json_t *json, *jsonCfg, *jsonCfgAccount, *jsonCfgHostName, *jsonCfgToken,
      *jsonCfgSSL;
  json = jsonCfg = jsonCfgAccount = jsonCfgHostName = jsonCfgToken =
      jsonCfgSSL = NULL;

  json = json_loads(json_str, 0, &error);
  if (json) {
    jsonCfg = json_object_get(json, "config");
    jsonCfgAccount = json_object_get(jsonCfg, "account");
    jsonCfgHostName = json_object_get(jsonCfg, "hostName");
    jsonCfgToken = json_object_get(jsonCfg, "token");
    jsonCfgSSL = json_object_get(jsonCfg, "ssl");

    /* compute size for the target struct */
    size = sizeof(Transit) + sizeof(GroundworkConfig);
    size += json_string_length(jsonCfgAccount) + NUL_TERM_LEN;
    size += json_string_length(jsonCfgHostName) + NUL_TERM_LEN;
    size += json_string_length(jsonCfgToken) + NUL_TERM_LEN;

    /* allocate and fill the target struct by pointer */
    transit = malloc(size);
    if (transit) {
      Transit *ptr = transit;
      ptr += sizeof(Transit) + sizeof(GroundworkConfig);
      transit->config.hostName =
          strcpy((char *)ptr, json_string_value(jsonCfgHostName));
      ptr += json_string_length(jsonCfgHostName) + NUL_TERM_LEN;
      transit->config.account =
          strcpy((char *)ptr, json_string_value(jsonCfgAccount));
      ptr += json_string_length(jsonCfgAccount) + NUL_TERM_LEN;
      transit->config.token =
          strcpy((char *)ptr, json_string_value(jsonCfgToken));
      transit->config.ssl = json_boolean_value(jsonCfgSSL);
    } else {
      fprintf(stderr, "decodeTransit error: %s\n", "malloc failed");
    }
  } else {
    fprintf(stderr, "decodeTransit error: %d: %s\n", error.line, error.text);
  }

  json_decref(json);
  json_decref(jsonCfg);
  json_decref(jsonCfgAccount);
  json_decref(jsonCfgHostName);
  json_decref(jsonCfgToken);
  json_decref(jsonCfgSSL);
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

          //   json_object_set_new(jsonProp, "stringValue",
          //   json_string("val-2"));
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
