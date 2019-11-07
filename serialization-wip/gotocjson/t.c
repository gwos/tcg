#include "convert_go_to_c.h"
#include "config.h"
#include "milliseconds.h"
#include "transit.h"

const string const ValueType_String[] = {
    "IntegerType",
    "DoubleType",
    "StringType",
    "BooleanType",
    "TimeType",
    "UnspecifiedType",
};

const string const MonitoredResourceType_String[] = {
    "service",
    "host", 
};

// ----------------------------------------------------------------

/*
typedef config_Config *config_Config_Ptr;

typedef struct {
    config_Config_Ptr config_Config_ptr_;  // go: *config.Config
} transit_Transit;
*/
json_t *transit_Transit_as_JSON(const transit_Transit *transit_Transit) {
    json_t *json;

    if (transit_Transit->config_Config_ptr_ == NULL) {
        printf(FILE_LINE "transit_Transit->config_Config_ptr_ is NULL\n");
    }
    else {
        printf(FILE_LINE "transit_Transit->config_Config_ptr_ is not NULL\n");
    }

    // FIX QUICK:  this block is for development purposes
    json_t *json_config_Config = config_Config_as_JSON( transit_Transit->config_Config_ptr_ );
    if (json_config_Config == NULL) {
        printf(FILE_LINE "json_config_Config is NULL\n");
    }
    else {
        printf(FILE_LINE "json_config_Config is not NULL\n");
    }

    // FIX MAJOR:  change this to show error detail
    json_error_t error;
    size_t flags = 0;
    // json = json_pack_ex(&error, flags, "{s:o?}"
    json = json_pack("{s:o?}"
	// FIX MAJOR:  revisit the pointer stuff once that is settled out upstream
	// FIX MAJOR:  we used to use "config_Config_ptr_" here
        , "Config", config_Config_as_JSON( transit_Transit->config_Config_ptr_ )
    );
    if (json == NULL) {
	// printf(FILE_LINE "ERROR:  text '%s', source '%s', line %d, column %d, position %d\n",
	    // error.text, error.source, error.line, error.column, error.position);
	printf(FILE_LINE "transit_Transit_as_JSON is returning a NULL pointer from json_pack() instead of a json pointer\n");
    }
    return json;
}

// Encoding routine.
char *transit_Transit_as_JSON_str(const transit_Transit *transit_Transit) {
    printf(FILE_LINE "FIX QUICK:  got into transit_Transit_as_JSON_str()\n");
    size_t flags = 0;
    return JSON_as_string(transit_Transit_as_JSON(transit_Transit), flags);
}
/*
// FIX MAJOR:  for simplicity, publish this instead of the actual function just above
#define transit_Transit_as_JSON_str(transit_Transit) JSON_as_string(transit_Transit_as_JSON(transit_Transit), 0)
*/

// ----------------------------------------------------------------

/*
typedef struct {
    string Name;
    transit_MonitoredResourceType Type;
    string Owner;
} transit_MonitoredResource;
*/
json_t *transit_MonitoredResource_as_JSON(const transit_MonitoredResource *transit_MonitoredResource) {
    json_error_t error;
    size_t flags = 0;
    json_t *json;
    printf(FILE_LINE "before json_pack() in transit_MonitoredResource_as_JSON\n");
    printf(FILE_LINE "transit_MonitoredResource->Name is '%s'\n", transit_MonitoredResource->Name);
    // json = json_pack("{s:s s:s s:s}"
    json = json_pack_ex(&error, flags, "{s:s s:s s:s}"
        , "Name",  transit_MonitoredResource->Name
        , "Type",  MonitoredResourceType_String[transit_MonitoredResource->Type]
        , "Owner", transit_MonitoredResource->Owner
    );
    if (json == NULL) {
	printf(FILE_LINE "ERROR:  text '%s', source '%s', line %d, column %d, position %d\n",
	    error.text, error.source, error.line, error.column, error.position);
    }
    printf(FILE_LINE " after json_pack() in transit_MonitoredResource_as_JSON; json = %p\n", json);
    return json;
}

// Encoding routine.
char *transit_MonitoredResource_as_JSON_str(const transit_MonitoredResource *transit_MonitoredResource) {
    size_t flags = 0;
    return JSON_as_string(transit_MonitoredResource_as_JSON(transit_MonitoredResource), flags);
}
/*
// FIX MAJOR:  for simplicity, publish this instead of the actual function just above
#define transit_MonitoredResource_as_JSON_str(transit_MonitoredResource) JSON_as_string(transit_MonitoredResource_as_JSON(transit_MonitoredResource), 0)
*/

// ----------------------------------------------------------------

json_t *transit_TypedValue_as_JSON(const transit_TypedValue *transit_TypedValue) {
    json_error_t error;
    size_t flags = 0;
    json_t *json;
    printf(FILE_LINE "before json_pack() in transit_TypedValue_as_JSON\n");
    printf(FILE_LINE "transit_TypedValue->StringValue = %p, '%s'\n", transit_TypedValue->StringValue, transit_TypedValue->StringValue);
    // FIX MAJOR:  get all the element types correctly processed; transit_ValueType is processed as a 32-bit value
    json = json_pack_ex(&error, flags, "{s:s}"
        , "ValueType", ValueType_String[transit_TypedValue->ValueType]
    );
    if (json == NULL) {
	printf(FILE_LINE "ERROR:  text '%s', source '%s', line %d, column %d, position %d\n",
	    error.text, error.source, error.line, error.column, error.position);
    } else {
	switch (transit_TypedValue->ValueType) {
	    case BooleanType:
		if (json_object_set_new(json
		    , "BoolValue",    json_boolean(transit_TypedValue->BoolValue)    // bool
		) != 0) {
		}
		break;
	    case DoubleType:
		if (json_object_set_new(json
		    , "DoubleValue",  json_real(transit_TypedValue->DoubleValue)  // float64
		) != 0) {
		}
		break;
	    case IntegerType:
		if (json_object_set_new(json
		    , "IntegerValue", json_integer(transit_TypedValue->IntegerValue) // int64
		) != 0) {
		}
		break;
	    case StringType:
		if (json_object_set_new(json
		    , "StringValue",  json_string(transit_TypedValue->StringValue)  // string
		) != 0) {
		}
		break;
	    case TimeType:
		if (json_object_set_new(json
		    , "TimeValue",    milliseconds_MillisecondTimestamp_as_JSON( &transit_TypedValue->TimeValue ) // milliseconds_MillisecondTimestamp
		) != 0) {
		}
		break;
	}
    }
    printf(FILE_LINE " after json_pack() in transit_TypedValue_as_JSON; json = %p\n", json);
    return json;
}

// Encoding routine.
char *transit_TypedValue_as_JSON_str(const transit_TypedValue *transit_TypedValue) {
    size_t flags = 0;
    return JSON_as_string(transit_TypedValue_as_JSON(transit_TypedValue), flags);
}
/*
// FIX MAJOR:  for simplicity, publish this instead of the actual function just above
#define transit_TypedValue_as_JSON_str(transit_TypedValue) JSON_as_string(transit_TypedValue_as_JSON(transit_TypedValue), 0)
*/

// ----------------------------------------------------------------

/*
typedef struct {
    string key;
    transit_TypedValue value;
} string_transit_TypedValue_Pair;
*/
json_t *string_transit_TypedValue_Pair_as_JSON(const string_transit_TypedValue_Pair *string_transit_TypedValue_Pair) {
    json_error_t error;
    size_t flags = 0;
    json_t *json;
    printf(FILE_LINE "before json_pack() in string_transit_TypedValue_Pair_as_JSON\n");
    // json = json_pack("{s:i s:o}"
    json = json_pack_ex(&error, flags, "{s:s s:o}"
        , "key",   string_transit_TypedValue_Pair->key                                  // string
        , "value", transit_TypedValue_as_JSON( &string_transit_TypedValue_Pair->value ) // transit_TypedValue
    );
    if (json == NULL) {
	printf(FILE_LINE "ERROR:  text '%s', source '%s', line %d, column %d, position %d\n",
	    error.text, error.source, error.line, error.column, error.position);
    }
    printf(FILE_LINE " after json_pack() in string_transit_TypedValue_Pair_as_JSON; json = %p\n", json);
    return json;
}

/*
typedef struct {
    size_t count;
    string_transit_TypedValue_Pair *items; 
} string_transit_TypedValue_Pair_List;
*/
// FIX MAJOR:  We ought to have an error reporting channel in routines such as this, as a separate return value
// other than just returning a NULL json object, to distinguish the result of a "omitempty" struct tag field
// from a NULL value which is used as an error response.  Of course, that will somewhat complicate the process
// of calling this routine.
json_t *string_transit_TypedValue_Pair_List_as_JSON(const string_transit_TypedValue_Pair_List *string_transit_TypedValue_Pair_List) {
    /*
    json_t *json = json_array();
    */
    json_t *json = json_object();
    if (json == NULL) {
        // FIX MAJOR:  fill in error reporting
    } else if (string_transit_TypedValue_Pair_List->count == 0) {
	json_decref(json);
	json = NULL;
    } else {
	for (size_t i = 0; i < string_transit_TypedValue_Pair_List->count; ++i) {
	    /*
	    if (json_array_append_new(json,
		// FIX QUICK:  replace this with an direct encoding of the pair, not referencing "key" and "value" keys
		string_transit_TypedValue_Pair_as_JSON( &string_transit_TypedValue_Pair_List->items[i] ) // string_transit_TypedValue_Pair*
	    ) != 0) {
		...
	    }
	    */
	    if (json_object_set_new(json
		, string_transit_TypedValue_Pair_List->items[i].key                                  // string
		, transit_TypedValue_as_JSON( &string_transit_TypedValue_Pair_List->items[i].value ) // transit_TypedValue
	    ) != 0) {
		// FIX MAJOR:
		// Report and handle the error condition.  Unfortunately, there is no json_error_t value to
		// look at, to determine the exact cause.  wlso, be aware that we might now have a memory leak.
		// since we don't know exactly what happened, we would rather suffer that leak than attempt to
		// decrement the reference count on the subsidiary object that we just tried to add to the array
		// (if in fact it was non-NULL).
		//
		// Since adding one element to the array didn't work, we abort the process of trying to add any
		// additional elements to the array.  Instead, we just clear out the entire array, and we return
		// a NULL value to indicate the error.
		json_array_clear(json);
		json_decref(json);
		json = NULL;
		break;
	    }
	}
    }
    return json;
}

/*
typedef struct _transit_InventoryResource_ {
    string Name;
    string Type;
    string Owner;
    string Category;
    string Description;
    string Device; 
    // // Foundation Properties
    // Properties map[string]TypedValue `json:"properties,omitempty"`
    string_transit_TypedValue_Pair_List Properties;  // go: map[string]TypedValue
} transit_InventoryResource;
*/
json_t *transit_InventoryResource_as_JSON(const transit_InventoryResource *transit_InventoryResource) {
    json_error_t error;
    size_t flags = 0;
    json_t *json;
    printf(FILE_LINE "before json_pack() in transit_InventoryResource_as_JSON\n");
    // json = json_pack("{s:s s:s s:s s:s s:s s:s s:o}"
    json = json_pack_ex(&error, flags, "{s:s s:s s:s s:s s:s s:s s:o}"
        , "Name",        transit_InventoryResource->Name        // string
        , "Type",        transit_InventoryResource->Type        // string
        , "Owner",       transit_InventoryResource->Owner       // string
        , "Category",    transit_InventoryResource->Category    // string
        , "Description", transit_InventoryResource->Description // string
        , "Device",      transit_InventoryResource->Device      // string
        , "Properties",  string_transit_TypedValue_Pair_List_as_JSON( &transit_InventoryResource->Properties ) // string_transit_TypedValue_Pair_List
    );
    if (json == NULL) {
	printf(FILE_LINE "ERROR:  text '%s', source '%s', line %d, column %d, position %d\n",
	    error.text, error.source, error.line, error.column, error.position);
    }
    printf(FILE_LINE " after json_pack() in transit_InventoryResource_as_JSON; json = %p\n", json);
    return json;
}

// Encoding routine.
char *transit_InventoryResource_as_JSON_str(const transit_InventoryResource *transit_InventoryResource) {
    size_t flags = 0;
    return JSON_as_string(transit_InventoryResource_as_JSON(transit_InventoryResource), flags);
}
/*
// FIX MAJOR:  for simplicity, publish this instead of the actual function just above
#define transit_InventoryResource_as_JSON_str(transit_InventoryResource) JSON_as_string(transit_InventoryResource_as_JSON(transit_InventoryResource), 0)
*/

// ----------------------------------------------------------------

/*
typedef struct {
    string ControllerAddr;
    string ControllerCertFile;
    string ControllerKeyFile;
    string NATSFilestoreDir;
    string NATSStoreType;
    bool StartController;
    bool StartNATS;
    bool StartTransport;
} config_AgentConfig;
*/
config_AgentConfig *JSON_as_config_AgentConfig(json_t *json) {
    config_AgentConfig *AgentConfig = (config_AgentConfig *)malloc(sizeof(config_AgentConfig));
    if (!AgentConfig) {
	// FIX MAJOR:  invoke proper logging for error conditions
	fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_config_AgentConfig, %s\n", "malloc failed");
    } else {
	if (json_unpack(json, "{s:s s:s s:s s:s s:s s:b s:b s:b}"
	    , "ControllerAddr",     &AgentConfig->ControllerAddr
	    , "ControllerCertFile", &AgentConfig->ControllerCertFile
	    , "ControllerKeyFile",  &AgentConfig->ControllerKeyFile
	    , "NATSFilestoreDir",   &AgentConfig->NATSFilestoreDir
	    , "NATSStoreType",      &AgentConfig->NATSStoreType
	    , "StartController",    &AgentConfig->StartController
	    , "StartNATS",          &AgentConfig->StartNATS
	    , "StartTransport",     &AgentConfig->StartTransport
	) != 0) {
	    // FIX MAJOR:  invoke proper logging for error conditions
	    fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_config_AgentConfig, %s\n", "JSON unpacking failed");
	    free(AgentConfig);
	    AgentConfig = NULL;
	}
    }
    return AgentConfig;
}

/*
typedef struct {
    string Host;
    string Account;
    string Password;
    string Token;
    string AppName;
} config_GroundworkConfig;
*/
config_GroundworkConfig *JSON_as_config_GroundworkConfig(json_t *json) {
    config_GroundworkConfig *GroundworkConfig = (config_GroundworkConfig *)malloc(sizeof(config_GroundworkConfig));
    if (!GroundworkConfig) {
	// FIX MAJOR:  invoke proper logging for error conditions
	fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_config_GroundworkConfig, %s\n", "malloc failed");
    } else {
	if (json_unpack(json, "{s:s s:s s:s s:s s:s}"
	    , "Host",     &GroundworkConfig->Host
	    , "Account",  &GroundworkConfig->Account
	    , "Password", &GroundworkConfig->Password
	    , "Token",    &GroundworkConfig->Token
	    , "AppName",  &GroundworkConfig->AppName
	) != 0) {
	    // FIX MAJOR:  invoke proper logging for error conditions
	    fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_config_GroundworkConfig, %s\n", "JSON unpacking failed");
	    free(GroundworkConfig);
	    GroundworkConfig = NULL;
	}
    }
    return GroundworkConfig;
}

/*
typedef struct {
    string Entrypoint;
} config_GroundworkAction;
*/
config_GroundworkAction *JSON_as_config_GroundworkAction(json_t *json) {
    config_GroundworkAction *GroundworkAction = (config_GroundworkAction *)malloc(sizeof(config_GroundworkAction));
    if (!GroundworkAction) {
	// FIX MAJOR:  invoke proper logging for error conditions
	fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_config_GroundworkAction, %s\n", "malloc failed");
    } else {
	if (json_unpack(json, "{s:s}"
	    , "Entrypoint", &GroundworkAction->Entrypoint
	) != 0) {
	    // FIX MAJOR:  invoke proper logging for error conditions
	    fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_config_GroundworkAction, %s\n", "JSON unpacking failed");
	    free(GroundworkAction);
	    GroundworkAction = NULL;
	}
    }
    return GroundworkAction;
}

/*
typedef struct {
    config_GroundworkAction Connect;
    config_GroundworkAction Disconnect;
    config_GroundworkAction SynchronizeInventory;
    config_GroundworkAction SendResourceWithMetrics;
    config_GroundworkAction ValidateToken;
} config_GroundworkActions;
*/
config_GroundworkActions *JSON_as_config_GroundworkActions(json_t *json) {
    config_GroundworkActions *GroundworkActions = (config_GroundworkActions *)malloc(sizeof(config_GroundworkActions));
    if (!GroundworkActions) {
	// FIX MAJOR:  invoke proper logging for error conditions
	fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_config_GroundworkActions, %s\n", "malloc failed");
    } else {
	// FIX MAJOR:  correct this code; perhaps I need to allocate the json objects beforehand,
	// and delete them afterward?
	json_t *json_Connect;
	json_t *json_Disconnect;
	json_t *json_SynchronizeInventory;
	json_t *json_SendResourceWithMetrics;
	json_t *json_ValidateToken;
	if (json_unpack(json, "{s:o s:o s:o s:o s:o}"
	    , "Connect",                 &json_Connect
	    , "Disconnect",              &json_Disconnect
	    , "SynchronizeInventory",    &json_SynchronizeInventory
	    , "SendResourceWithMetrics", &json_SendResourceWithMetrics
	    , "ValidateToken",           &json_ValidateToken
	) != 0) {
	    // FIX MAJOR:  invoke proper logging for error conditions
	    fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_config_GroundworkActions, %s\n", "JSON unpacking failed");
	    free(GroundworkActions);
	    GroundworkActions = NULL;
	} else {
	    GroundworkActions->Connect                 = *JSON_as_config_GroundworkAction(json_Connect);
	    GroundworkActions->Disconnect              = *JSON_as_config_GroundworkAction(json_Disconnect);
	    GroundworkActions->SynchronizeInventory    = *JSON_as_config_GroundworkAction(json_SynchronizeInventory);
	    GroundworkActions->SendResourceWithMetrics = *JSON_as_config_GroundworkAction(json_SendResourceWithMetrics);
	    GroundworkActions->ValidateToken           = *JSON_as_config_GroundworkAction(json_ValidateToken);
	}
    }
    return GroundworkActions;
}

config_Config *JSON_as_config_Config(json_t *json) {
    config_Config *Config = (config_Config *)malloc(sizeof(config_Config));
    if (!Config) {
	// FIX MAJOR:  invoke proper logging for error conditions
	fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_config_Config, %s\n", "malloc failed");
    } else {
	// FIX MAJOR:  correct this code; perhaps I need to allocate the json objects beforehand,
	// and delete them afterward?
	json_t *json_AgentConfig;
	json_t *json_GroundworkConfig;
	json_t *json_GroundworkActions;
	if (json_unpack(json, "{s:o s:o s:o}"
	    , "AgentConfig",       &json_AgentConfig
	    , "GroundworkConfig",  &json_GroundworkConfig
	    , "GroundworkActions", &json_GroundworkActions
	) != 0) {
	    // FIX MAJOR:  invoke proper logging for error conditions
	    fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_config_Config, %s\n", "JSON unpacking failed");
	    free(Config);
	    Config = NULL;
	} else {
	    Config->AgentConfig       = *JSON_as_config_AgentConfig      (json_AgentConfig);
	    Config->GroundworkConfig  = *JSON_as_config_GroundworkConfig (json_GroundworkConfig);
	    Config->GroundworkActions = *JSON_as_config_GroundworkActions(json_GroundworkActions);
	}
    }
    return Config;
}

/*
typedef struct _transit_TypedValue_ {
    transit_ValueType ValueType;
    bool BoolValue;
    float64 DoubleValue;
    int64 IntegerValue;
    string StringValue;
    milliseconds_MillisecondTimestamp TimeValue;  // go:  milliseconds.MillisecondTimestamp
} transit_TypedValue;
*/
transit_TypedValue *JSON_as_transit_TypedValue(json_t *json) {
    transit_TypedValue *TypedValue = (transit_TypedValue *)malloc(sizeof(transit_TypedValue));
    if (!TypedValue) {
	// FIX MAJOR:  invoke proper logging for error conditions
	fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_transit_TypedValue, %s\n", "malloc failed");
    } else {
	// FIX MAJOR:  correct this code
	// We initialize all data elements of the data structure to what we consider to be their respective
	// zero values, to prepare for correct operation using the "omitempty" struct field tag.
	TypedValue->ValueType    = UnspecifiedType;
	TypedValue->BoolValue    = false;
	TypedValue->DoubleValue  = 0.0;
	TypedValue->IntegerValue = 0;
	TypedValue->StringValue  = NULL;  // FIX MAJOR:  Possibly, this should be some copy of the empty string instead.  But pay attention to issues of later deletion.
	TypedValue->TimeValue    = (milliseconds_MillisecondTimestamp) { (struct_timespec) { 0, 0 } };  // FIX MAJOR:  define an "epoch_milliseconds_MillisecondTimestamp"
	int failed = 0;
	char *ValueType_as_string;
	json_t *json_TimeValue;
	// json_object_foreach(json, key, value) {
	// }
	if (json_unpack(json, "{s:s s?:b s?:f s?:I s?:s s?:o}"
	    , "ValueType",    &ValueType_as_string
	    , "BoolValue",    &TypedValue->BoolValue
	    , "DoubleValue",  &TypedValue->DoubleValue
	    , "IntegerValue", &TypedValue->IntegerValue
	    , "StringValue",  &TypedValue->StringValue
	    , "TimeValue",    &json_TimeValue
	) != 0) {
	    // FIX MAJOR:  invoke proper logging for error conditions
	    fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_transit_TypedValue, %s\n", "JSON unpacking failed");
	    failed = 1;
	} else {
	    printf(FILE_LINE "NOTICE:  ValueType is %s\n", ValueType_as_string);
	    printf(FILE_LINE "NOTICE:  IntegerValue is 0x%lx\n", TypedValue->IntegerValue);
	    printf(FILE_LINE "JSON_INTEGER_FORMAT is " JSON_INTEGER_FORMAT "\n");
	    if ((TypedValue->ValueType = enumeration_value(ValueType_String, arraysize(ValueType_String), ValueType_as_string)) < 0) {
		fprintf(stderr, FILE_LINE "ERROR:  cannot find the ValueType enumeration value for ValueType '%s'\n", ValueType_as_string);
		failed = 1;
	    }
	    printf(FILE_LINE "NOTICE:  ValueType is %d\n", TypedValue->ValueType);
	    if (TypedValue->ValueType != StringType) {
		TypedValue->StringValue = NULL;
	    }
	    if (TypedValue->ValueType == TimeType) {
		TypedValue->TimeValue = *JSON_as_milliseconds_MillisecondTimestamp(json_TimeValue);
	    }
	}
	if (failed) {
	    free(TypedValue);
	    TypedValue = NULL;
	}
    }
    return TypedValue;
}

/*
typedef struct _string_transit_TypedValue_Pair_ {
    string key;
    transit_TypedValue value;
} string_transit_TypedValue_Pair;
*/
string_transit_TypedValue_Pair *JSON_as_string_transit_TypedValue_Pair(json_t *json) {
    string_transit_TypedValue_Pair *Pair = (string_transit_TypedValue_Pair *)malloc(sizeof(string_transit_TypedValue_Pair));
    if (!Pair) {
	// FIX MAJOR:  invoke proper logging for error conditions
	fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_string_transit_TypedValue_Pair, %s\n", "malloc failed");
    } else {
	// FIX MAJOR:  correct this code
	int failed = 0;
	json_t *json_value;
	if (json_unpack(json, "{s:s s:o}"
	    , "key",   &Pair->key
	    , "value", &json_value
	) != 0) {
	    // FIX MAJOR:  invoke proper logging for error conditions
	    fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_string_transit_TypedValue_Pair, %s\n", "JSON unpacking failed");
	    failed = 1;
	} else {
	    printf(FILE_LINE "in JSON_as_string_transit_TypedValue_Pair, key = '%s'\n", Pair->key);
	    transit_TypedValue *TypedValue_ptr = JSON_as_transit_TypedValue(json_value);
	    if (TypedValue_ptr == NULL) {
		fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_string_transit_TypedValue_Pair, %s\n", "TypedValue_ptr is NULL");
		failed = 1;
	    } else {
		Pair->value = *TypedValue_ptr;
	    }
	}
	if (failed) {
	    free(Pair);
	    Pair = NULL;
	}
    }
    return Pair;
}

/*
typedef struct _string_transit_TypedValue_Pair_List_ {
    size_t count;
    string_transit_TypedValue_Pair *items;
} string_transit_TypedValue_Pair_List;
*/
string_transit_TypedValue_Pair_List *JSON_as_string_transit_TypedValue_Pair_List(json_t *json) {
    string_transit_TypedValue_Pair_List *Pair_List = (string_transit_TypedValue_Pair_List *)malloc(sizeof(string_transit_TypedValue_Pair_List));
    if (!Pair_List) {
	// FIX MAJOR:  invoke proper logging for error conditions
	fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_string_transit_TypedValue_Pair_List, %s\n", "malloc failed");
    } else {
	// FIX MAJOR:  correct this code
	int failed = 0;
	if (json_is_array(json)) {
	    // This code is speculative, a possible guide to how we might implement non-Pair lists.
	    Pair_List->count = json_array_size(json);
	    Pair_List->items = (string_transit_TypedValue_Pair *)malloc(Pair_List->count * sizeof(string_transit_TypedValue_Pair));
	    for (size_t i = 0; i < Pair_List->count; ++i) {
		json_t *json_Pair = json_array_get(json, i);
		if (json_Pair == NULL) {
		    // FIX MAJOR:  invoke proper logging for error conditions
		    fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_string_transit_TypedValue_Pair_List, %s\n", "JSON unpacking failed");
		    failed = 1;
		} else {
		    Pair_List->items[i] = *JSON_as_string_transit_TypedValue_Pair(json_Pair);
		}
	    }
	} else if (json_is_object(json)) {
	    // This is actually the normal, expected condition for a Pair_List.
	    Pair_List->count = json_object_size(json);
	    Pair_List->items = (string_transit_TypedValue_Pair *)malloc(Pair_List->count * sizeof(string_transit_TypedValue_Pair));
	    const char *key;
	    json_t *value;
	    size_t i = 0;
	    json_object_foreach(json, key, value) {
		// FIX QUICK
		// Here we throw away constness as far as the compiler is concerned, but by convention
		// the calling code will never alter the key, so that won't matter.
		Pair_List->items[i].key   = (char *) key;
		transit_TypedValue *TypedValue_ptr = JSON_as_transit_TypedValue(value);
		if (TypedValue_ptr == NULL) {
		    fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_string_transit_TypedValue_Pair_List, %s\n", "TypedValue_ptr is NULL");
		    failed = 1;
		} else {
		    Pair_List->items[i].value = *TypedValue_ptr;
		}
		fprintf(stderr, "processed key %s\n", key);
		/*
		json_t *json_Pair = json_array_get(json, i);
		if (json_Pair == NULL) {
		    // FIX MAJOR:  invoke proper logging for error conditions
		    fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_string_transit_TypedValue_Pair_List, %s\n", "JSON unpacking failed");
		    failed = 1;
		} else {
		    Pair_List->items[i] = *JSON_as_string_transit_TypedValue_Pair(json_Pair);
		}
		*/
		++i;
	    }
	} else {
	    // FIX MAJOR:  invoke proper logging for error conditions
	    fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_string_transit_TypedValue_Pair_List, %s\n", "JSON fragment is not an array or object");
	    failed = 1;
	}
	if (failed) {
	    free(Pair_List);
	    Pair_List = NULL;
	}
    }
    return Pair_List;
}

/*
typedef struct _transit_InventoryResource_ {
    string Name;
    string Type;
    string Owner;
    string Category;
    string Description;
    string Device; 
    string_transit_TypedValue_Pair_List Properties;  // go: map[string]TypedValue
} transit_InventoryResource;
*/
transit_InventoryResource *JSON_as_transit_InventoryResource(json_t *json) {
    transit_InventoryResource *InventoryResource = (transit_InventoryResource *)malloc(sizeof(transit_InventoryResource));
    if (!InventoryResource) {
	// FIX MAJOR:  invoke proper logging for error conditions
	fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_transit_InventoryResource, %s\n", "malloc failed");
    } else {
	int failed = 0;
	json_t *json_Properties;
	if (json_unpack(json, "{s:s s:s s:s s:s s:s s:s s:o}"
	    , "Name",        &InventoryResource->Name
	    , "Type",        &InventoryResource->Type
	    , "Owner",       &InventoryResource->Owner
	    , "Category",    &InventoryResource->Category
	    , "Description", &InventoryResource->Description
	    , "Device",      &InventoryResource->Device
	    , "Properties",  &json_Properties
	) != 0) {
	    // FIX MAJOR:  invoke proper logging for error conditions
	    fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_transit_InventoryResource, %s\n", "JSON unpacking failed");
	    failed = 1;
	} else {
	    printf(FILE_LINE "in JSON_as_transit_InventoryResource, Name  = '%s'\n", InventoryResource->Name);
	    printf(FILE_LINE "in JSON_as_transit_InventoryResource, Owner = '%s'\n", InventoryResource->Owner);
	    string_transit_TypedValue_Pair_List *string_transit_TypedValue_Pair_List_ptr = JSON_as_string_transit_TypedValue_Pair_List(json_Properties);
	    if (string_transit_TypedValue_Pair_List_ptr == NULL) {
		fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_transit_InventoryResource, %s\n", "string_transit_TypedValue_Pair_List_ptr is NULL");
		failed = 1;
	    } else {
		InventoryResource->Properties = *string_transit_TypedValue_Pair_List_ptr;
	    }
	    /*
	    if ((InventoryResource->Type = enumeration_value(InventoryResourceType_String, arraysize(InventoryResourceType_String), Type_as_string)) < 0) {
		fprintf(stderr, FILE_LINE "ERROR:  cannot find the InventoryResourceType enumeration value for Type '%s'\n", Type_as_string);
		failed = 1;
	    }
	    */
	}
	if (failed) {
	    free(InventoryResource);
	    InventoryResource = NULL;
	}
    }
    fprintf(stderr, "exiting from JSON_as_transit_InventoryResource with pointer %p\n", InventoryResource);
    return InventoryResource;
}

transit_MonitoredResource *JSON_as_transit_MonitoredResource(json_t *json) {
    transit_MonitoredResource *MonitoredResource = (transit_MonitoredResource *)malloc(sizeof(transit_MonitoredResource));
    if (!MonitoredResource) {
	// FIX MAJOR:  invoke proper logging for error conditions
	fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_transit_MonitoredResource, %s\n", "malloc failed");
    } else {
	// FIX MAJOR:  correct this code
	int failed = 0;
	char *Type_as_string;
	if (json_unpack(json, "{s:s s:s s:s}"
	    , "Name",  &MonitoredResource->Name
	    , "Type",  &Type_as_string
	    , "Owner", &MonitoredResource->Owner
	) != 0) {
	    // FIX MAJOR:  invoke proper logging for error conditions
	    fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_transit_MonitoredResource, %s\n", "JSON unpacking failed");
	    failed = 1;
	} else {
	    printf(FILE_LINE "in JSON_as_transit_MonitoredResource, Name  = '%s'\n", MonitoredResource->Name);
	    printf(FILE_LINE "in JSON_as_transit_MonitoredResource, Owner = '%s'\n", MonitoredResource->Owner);
	    if ((MonitoredResource->Type = enumeration_value(MonitoredResourceType_String, arraysize(MonitoredResourceType_String), Type_as_string)) < 0) {
		fprintf(stderr, FILE_LINE "ERROR:  cannot find the MonitoredResourceType enumeration value for Type '%s'\n", Type_as_string);
		failed = 1;
	    }
	}
	if (failed) {
	    free(MonitoredResource);
	    MonitoredResource = NULL;
	}
    }
    return MonitoredResource;
}

transit_Transit *JSON_as_transit_Transit(json_t *json) {
    transit_Transit *Transit = (transit_Transit *)malloc(sizeof(transit_Transit));
    if (!Transit) {
	// FIX MAJOR:  invoke proper logging for error conditions
	fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_transit_Transit, %s\n", "malloc failed");
    } else {
	int failed = 0;
	// FIX MAJOR:  correct this code
	json_t *json_config_Config_ptr_;
	if (json_unpack(json, "{s:o}"
	    , "Config", &json_config_Config_ptr_
	) != 0) {
	    // FIX MAJOR:  invoke proper logging for error conditions
	    fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_transit_Transit, %s\n", "JSON unpacking failed");
	    failed = 1;
	}
	else {
	    // We should have a JSON_OBJECT here now.
	    printf(FILE_LINE "before decoding config.Config, json_config_Config_ptr_ is %s\n",
		typeof_json_item(json_config_Config_ptr_));
	    Transit->config_Config_ptr_ = JSON_as_config_Config(json_config_Config_ptr_);
	    printf(FILE_LINE " after decoding config.Config\n");
	}
	if (failed) {
	    free(Transit);
	    Transit = NULL;
	}
    }
    return Transit;
}

// Decoding routine.
transit_InventoryResource *JSON_str_as_transit_InventoryResource(const char *json_str, json_t **json) {
    json_error_t error;
    *json = json_loads(json_str, 0, &error);
    if (*json == NULL) {
        // FIX MAJOR:  produce a log message based on the content of the "error" object
        printf(FILE_LINE "json for transit_InventoryResource is NULL\n");
	return NULL;
    }
    transit_InventoryResource *InventoryResource = JSON_as_transit_InventoryResource(*json);

    // Logically, we want to make this json_decref() call to release our hold on the JSON
    // object we obtained, because we are now supposedly done with that JSON object.  But
    // if we do so now, that will destroy all the strings we obtained from json_unpack()
    // and stuffed into the returned C object tree.  So instead, we just allow that pointer
    // to be returned to the caller, to be passed thereafter to free_JSON() once the caller
    // is completely done with the returned C object tree.
    //
    // json_decref(*json);

    return InventoryResource;
}

// Decoding routine.
transit_MonitoredResource *JSON_str_as_transit_MonitoredResource(const char *json_str, json_t **json) {
    json_error_t error;
    *json = json_loads(json_str, 0, &error);
    if (*json == NULL) {
        // FIX MAJOR:  produce a log message based on the content of the "error" object
        printf(FILE_LINE "json for transit_MonitoredResource is NULL\n");
	return NULL;
    }
    transit_MonitoredResource *MonitoredResource = JSON_as_transit_MonitoredResource(*json);

    // Logically, we want to make this json_decref() call to release our hold on the JSON
    // object we obtained, because we are now supposedly done with that JSON object.  But
    // if we do so now, that will destroy all the strings we obtained from json_unpack()
    // and stuffed into the returned C object tree.  So instead, we just allow that pointer
    // to be returned to the caller, to be passed thereafter to free_JSON() once the caller
    // is completely done with the returned C object tree.
    //
    // json_decref(*json);

    return MonitoredResource;
}

// Decoding routine.
transit_Transit *JSON_str_as_transit_Transit(const char *json_str, json_t **json) {
    json_error_t error;
    *json = json_loads(json_str, 0, &error);
    if (*json == NULL) {
        // FIX MAJOR:  produce a log message based on the content of the "error" object
        printf(FILE_LINE "json for transit_Transit is NULL\n");
	return NULL;
    }
    transit_Transit *Transit = JSON_as_transit_Transit(*json);

    // Logically, we want to make this json_decref() call to release our hold on the JSON
    // object we obtained, because we are now supposedly done with that JSON object.  But
    // if we do so now, that will destroy all the strings we obtained from json_unpack()
    // and stuffed into the returned C object tree.  So instead, we just allow that pointer
    // to be returned to the caller, to be passed thereafter to free_JSON() once the caller
    // is completely done with the returned C object tree.
    //
    // json_decref(*json);

    return Transit;
}

