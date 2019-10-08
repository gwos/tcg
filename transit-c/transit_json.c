#include <jansson.h>
#include <stddef.h>
#include <string.h>
#include <stdalign.h>	// Needed to supply alignof(), available starting with C11.

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
  json_t * json     = NULL;
  json_t * jsonUser = NULL;
  json_t * jsonPass = NULL;
  size_t jsonUser_len;
  size_t jsonPass_len;

  json = json_loads(json_str, 0, &error);
  if (json) {
    jsonUser = json_object_get(json, "user");
    jsonPass = json_object_get(json, "password");
    jsonUser_len = json_string_length(jsonUser);
    jsonPass_len = json_string_length(jsonPass);

    // incrementally compute a total size for allocation of the
    // target struct, including all the strings it refers to
    size = sizeof(Credentials);
    size += jsonUser_len + NUL_TERM_LEN;
    size += jsonPass_len + NUL_TERM_LEN;

    /* allocate and fill the target struct by pointer */
    credentials = (Credentials *) malloc(size);
    if (credentials) {
      char *ptr = (char *) credentials;
      ptr += sizeof(Credentials);
      credentials->user = strcpy(ptr, json_string_value(jsonUser));
      ptr += jsonUser_len + NUL_TERM_LEN;
      credentials->password = strcpy(ptr, json_string_value(jsonPass));
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
  json_t * json                 = NULL;
  json_t * jsonStatus           = NULL;
  json_t * jsonName             = NULL;
  json_t * jsonType             = NULL;
  json_t * jsonOwner            = NULL;
  json_t * jsonCategory         = NULL;
  json_t * jsonDescription      = NULL;
  json_t * jsonLastPlugInOutput = NULL;
  json_t * jsonLastCheckTime    = NULL;
  json_t * jsonNextCheckTime    = NULL;
  json_t * jsonProperties       = NULL;
  json_t * jsonProp             = NULL;
  json_t * jsonPropValue        = NULL;
  size_t jsonName_len;
  size_t jsonType_len;
  size_t jsonOwner_len;
  size_t jsonCategory_len;
  size_t jsonDescription_len;
  size_t jsonLastPlugInOutput_len;
  // size_t TypedValuePairList_alignment = alignof(TypedValuePairList);
  size_t     TypedValuePair_alignment = alignof(TypedValuePair);
  // size_t         TypedValue_alignment = alignof(TypedValue);
  // size_t TypedValuePairList_padding;
  size_t     TypedValuePair_padding;
  // size_t         TypedValue_padding;

  json = json_loads(json_str, 0, &error);
  if (json) {
    jsonStatus           = json_object_get( json, "status" );
    jsonName             = json_object_get( json, "name" );
    jsonType             = json_object_get( json, "type" );
    jsonOwner            = json_object_get( json, "owner" );
    jsonCategory         = json_object_get( json, "category" );
    jsonDescription      = json_object_get( json, "description" );
    jsonLastPlugInOutput = json_object_get( json, "lastPlugInOutput" );
    jsonLastCheckTime    = json_object_get( json, "lastCheckTime" );
    jsonNextCheckTime    = json_object_get( json, "nextCheckTime" );
    jsonProperties       = json_object_get( json, "properties" );

    jsonName_len = json_string_length(jsonName);
    jsonType_len = json_string_length(jsonType);

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

      // TypedValuePairList_padding = ((TypedValuePairList_alignment - 1) * size) % TypedValuePairList_alignment;
      // size += TypedValuePairList_padding + sizeof(TypedValuePairList);

      TypedValuePair_padding = ((TypedValuePair_alignment - 1) * size) % TypedValuePair_alignment;
      size += TypedValuePair_padding + json_object_size(jsonProperties) * sizeof(TypedValuePair);

      // TypedValue_padding = ((TypedValue_alignment - 1) * size) % TypedValue_alignment;
      // size += TypedValue_padding + json_object_size(jsonProperties) * sizeof(TypedValue);

      const char *key;
      json_object_foreach(jsonProperties, key, jsonProp) {
        size += strlen(key) + NUL_TERM_LEN;
        jsonPropValue = json_object_get(jsonProp, "stringValue");
        if (jsonPropValue) {
          size += json_string_length(jsonPropValue) + NUL_TERM_LEN;
          json_decref(jsonPropValue);
        }
      }
    }

    /* allocate and fill the target struct by pointer */
    resource = (MonitoredResource *) malloc(size);
    if (resource) {
      char *ptr = (char *) resource;

      resource->status = json_integer_value(jsonStatus);
      resource->lastCheckTime = 0;
      resource->nextCheckTime = 0;
      if (jsonLastCheckTime) {
        resource->lastCheckTime = json_integer_value(jsonLastCheckTime);
      }
      if (jsonNextCheckTime) {
        resource->nextCheckTime = json_integer_value(jsonNextCheckTime);
      }

printf ("about to process data\n");

      ptr += sizeof(MonitoredResource);
      resource->name = strcpy(ptr, json_string_value(jsonName));
printf ("resource->name = %s\n", resource->name);
      ptr += jsonName_len + NUL_TERM_LEN;
      resource->type = strcpy(ptr, json_string_value(jsonType));
printf ("resource->type = %s\n", resource->type);
      ptr += jsonType_len + NUL_TERM_LEN;

      if (jsonOwner) {
        resource->owner = strcpy(ptr, json_string_value(jsonOwner));
printf ("resource->owner = %s\n", resource->owner);
        ptr += jsonOwner_len + NUL_TERM_LEN;
      }
      if (jsonCategory) {
        resource->category = strcpy(ptr, json_string_value(jsonCategory));
printf ("resource->category = %s\n", resource->category);
        ptr += jsonCategory_len + NUL_TERM_LEN;
      }
      if (jsonDescription) {
        resource->description = strcpy(ptr, json_string_value(jsonDescription));
printf ("resource->description = %s\n", resource->description);
        ptr += jsonDescription_len + NUL_TERM_LEN;
      }
      if (jsonLastPlugInOutput) {
        resource->lastPlugInOutput = strcpy(ptr, json_string_value(jsonLastPlugInOutput));
printf ("resource->lastPlugInOutput = %s\n", resource->lastPlugInOutput);
        ptr += jsonLastPlugInOutput_len + NUL_TERM_LEN;
      }

      if (jsonProperties) {
	ptr += TypedValuePair_padding;
        resource->properties.count = json_object_size(jsonProperties);
        resource->properties.items = (TypedValuePair *)ptr;
	ptr += json_object_size(jsonProperties) * sizeof(TypedValuePair);

	// ptr += TypedValue_padding;
	// TypedValue *typed_value = (TypedValue *) ptr;
	// ptr += json_object_size(jsonProperties) * sizeof(TypedValue);

        size_t i = 0;
        const char *key;
        TypedValue emptyTypedValue = {0, false, 0, 0, 0};
        json_object_foreach(jsonProperties, key, jsonProp) {
          resource->properties.items[i].key = strcpy(ptr, key);
          ptr += strlen(key) + NUL_TERM_LEN;
	  memcpy(&resource->properties.items[i].value, &emptyTypedValue, sizeof(TypedValue));

          jsonPropValue = json_object_get(jsonProp, "valueType");
          VALUE_TYPE_ENUM valueType = json_integer_value(jsonPropValue);
          resource->properties.items[i].value.valueType = valueType;
          json_decref(jsonPropValue);

          switch (valueType) {
            case IntegerType:
              jsonPropValue = json_object_get(jsonProp, "integerValue");
              resource->properties.items[i].value.integerValue = json_integer_value(jsonPropValue);
              json_decref(jsonPropValue);
              break;

            case DoubleType:
              jsonPropValue = json_object_get(jsonProp, "doubleValue");
printf ("+++ DoubleType: ptr=%p, key='%s', double=%d\n", ptr, resource->properties.items[i].key, json_real_value(jsonPropValue));
              resource->properties.items[i].value.doubleValue = json_real_value(jsonPropValue);
              json_decref(jsonPropValue);
              break;

            case BooleanType:
              jsonPropValue = json_object_get(jsonProp, "boolValue");
printf ("+++ BooleanType: ptr=%p, key='%s', bool=%d\n", ptr, resource->properties.items[i].key, json_boolean_value(jsonPropValue));
              resource->properties.items[i].value.boolValue = json_boolean_value(jsonPropValue);
              json_decref(jsonPropValue);
              break;

            case DateType:
              /* TODO: millis number */
              jsonPropValue = json_object_get(jsonProp, "dateValue");
              resource->properties.items[i].value.dateValue = json_integer_value(jsonPropValue);
              json_decref(jsonPropValue);
              break;

            case StringType:
printf ("+++ calling json_object_get\n");
              jsonPropValue = json_object_get(jsonProp, "stringValue");
	      /*
	      if (0 && json_is_object(jsonPropValue)) {
printf ("+++ jsonPropValue is an object, not a string\n");
const char *propkey;
json_t * propprop = NULL;
        json_object_foreach(jsonPropValue, propkey, propprop) {
printf ("+++ jsonPropValue key is %s\n", propkey);
if (strcmp(propkey, "stringValue") == 0) {
// json_t *str = json_object_get(propprop, "stringValue");
printf (">>> jsonPropValue string value is %s\n", json_string_value(propprop));
}
	}
	      }
	      */
const char *string_value = json_string_value(jsonPropValue);
printf ("+++ calling strcpy; ptr=%p, key='%s', jsonPropValue=%p, typeof=%d, string=%p\n", ptr, resource->properties.items[i].key, jsonPropValue, json_typeof(jsonPropValue), string_value);
              resource->properties.items[i].value.stringValue = strcpy(ptr, json_string_value(jsonPropValue));
printf ("+++ calling json_string_length\n");
              ptr += json_string_length(jsonPropValue) + NUL_TERM_LEN;
printf ("+++ calling json_decref\n");
              json_decref(jsonPropValue);
printf ("+++ StringType end\n");
              break;

            default:
              break;
          }
printf ("+++ #10\n");
          i++;
        }
      }
    } else {
      fprintf(stderr, "decodeMonitoredResource error: %s\n", "malloc failed");
    }
  } else {
    fprintf(stderr, "decodeMonitoredResource error: %d: %s\n", error.line,
            error.text);
  }

printf ("+++ #11\n");
  // json_decref(json);
  // json_decref(jsonStatus);
  // json_decref(jsonName);
  // json_decref(jsonType);
  // json_decref(jsonOwner);
  // json_decref(jsonCategory);
  // json_decref(jsonDescription);
  // json_decref(jsonLastPlugInOutput);
  // json_decref(jsonLastCheckTime);
  // json_decref(jsonNextCheckTime);
  // json_decref(jsonProperties);

  //   json_decref(jsonProp);
printf ("+++ #12\n");

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
