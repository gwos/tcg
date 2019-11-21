json_t *transit_Transit_as_JSON(const transit_Transit *transit_Transit) {
    json_t *json;
    if (transit_Transit == NULL) {
	json = NULL;
    } else {
	json_error_t error;
	size_t flags = 0;
	json = json_pack_ex(&error, flags, "{s:o?}"
	    // FIX MAJOR:  we used to use "config_Config_ptr_" here, and need to be careful when generating code
	    , "Config", config_Config_as_JSON( transit_Transit->config_Config_ptr_ )
	);
	if (json == NULL) {
	    printf(FILE_LINE "ERROR:  text '%s', source '%s', line %d, column %d, position %d\n",
		error.text, error.source, error.line, error.column, error.position);
	}
    }
    return json;
}

json_t *transit_MonitoredResource_as_JSON(const transit_MonitoredResource *transit_MonitoredResource) {
    json_t *json;
    if (transit_MonitoredResource == NULL) {
	json = NULL;
    } else {
	json_error_t error;
	size_t flags = 0;
	json = json_pack_ex(&error, flags, "{s:s s:s s:s}"
	    , "Name",  transit_MonitoredResource->Name
	    , "Type",  MonitoredResourceType_String[transit_MonitoredResource->Type]
	    , "Owner", transit_MonitoredResource->Owner
	);
	if (json == NULL) {
	    printf(FILE_LINE "ERROR:  text '%s', source '%s', line %d, column %d, position %d\n",
		error.text, error.source, error.line, error.column, error.position);
	}
    }
    return json;
}

json_t *transit_InventoryResource_as_JSON(const transit_InventoryResource *transit_InventoryResource) {
    json_t *json;
    if (transit_InventoryResource == NULL) {
	json = NULL;
    } else {
	json_error_t error;
	size_t flags = 0;
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
    }
    return json;
}

json_t *transit_TypedValue_as_JSON(const transit_TypedValue *transit_TypedValue) {
    json_t *json;
    if (transit_TypedValue == NULL) {
	json = NULL;
    } else {
	json_error_t error;
	size_t flags = 0;
	// FIX MAJOR:  get all the element types correctly processed; the transit_ValueType discriminator is processed as a 32-bit value
	json = json_pack_ex(&error, flags, "{s:s}"
	    , "ValueType", ValueType_String[transit_TypedValue->ValueType]
	);
	if (json == NULL) {
	    printf(FILE_LINE "ERROR:  text '%s', source '%s', line %d, column %d, position %d\n",
		error.text, error.source, error.line, error.column, error.position);
	} else {
	    switch (transit_TypedValue->ValueType) {
		case BooleanType: // bool
		    if (json_object_set_new(json, "BoolValue",    json_boolean(transit_TypedValue->BoolValue)) != 0) {
		    }
		    break;
		case DoubleType: // float64
		    if (json_object_set_new(json, "DoubleValue",  json_real(transit_TypedValue->DoubleValue)) != 0) {
		    }
		    break;
		case IntegerType: // int64
		    if (json_object_set_new(json, "IntegerValue", json_integer(transit_TypedValue->IntegerValue)) != 0) {
		    }
		    break;
		case StringType: // string
		    if (json_object_set_new(json, "StringValue",  json_string(transit_TypedValue->StringValue)) != 0) {
		    }
		    break;
		case TimeType: // milliseconds_MillisecondTimestamp
		    if (json_object_set_new(json, "TimeValue",    milliseconds_MillisecondTimestamp_as_JSON( &transit_TypedValue->TimeValue )) != 0) {
		    }
		    break;
	    }
	}
    }
    return json;
}

/*
// we need these routines created:
#define transit_InventoryResource_List_Ptr_as_JSON(x)	json
#define transit_LabelDescriptor_Ptr_as_JSON(x)		json
#define transit_MetricSample_Ptr_as_JSON(x)		json
#define transit_OperationResult_List_Ptr_as_JSON(x)	json
#define transit_ResourceGroup_List_Ptr_as_JSON(x)	json
#define transit_ThresholdDescriptor_Ptr_as_JSON(x)	json
#define transit_TimeInterval_Ptr_as_JSON(x)		json

#define transit_XXX_List_Ptr_as_JSON(x)		json
#define transit_XXX_Ptr_as_JSON(x)		json

// we have:
typedef transit_LabelDescriptor *transit_LabelDescriptor_Ptr;
typedef transit_TimeInterval    *transit_TimeInterval_Ptr;
typedef transit_MetricSample    *transit_MetricSample_Ptr;

transit_InventoryResource	_List_Ptr
transit_LabelDescriptor		_Ptr
transit_MetricSample		_Ptr
transit_OperationResult		_List_Ptr
transit_ResourceGroup		_List_Ptr
transit_ThresholdDescriptor	_Ptr
transit_TimeInterval		_Ptr
transit_TypedValue		_Ptr
*/

/*
typedef struct _transit_MonitoredResource_List_ {
    size_t count;
    transit_MonitoredResource *items; 
} transit_MonitoredResource_List;
*/
json_t *transit_MonitoredResource_List_as_JSON(const transit_MonitoredResource_List *transit_MonitoredResource_List) {
    json_t *json;
    if (transit_MonitoredResource_List == NULL) {
	json = NULL;
    } else if (transit_MonitoredResource_List->count == 0) {
	json = NULL;
    } else {
	json = json_array();
	if (json == NULL) {
	    printf(FILE_LINE "ERROR:  cannot create a JSON %s object\n", "transit_MonitoredResource_List");
	} else {
	    for (size_t i = 0; i < transit_MonitoredResource_List->count; ++i) {
		if (json_array_append_new(json,
		    transit_MonitoredResource_as_JSON( &transit_MonitoredResource_List->items[i] ) // transit_MonitoredResource*
		) != 0) {
                    //
                    // Report and handle the error condition.  Unfortunately, there is no json_error_t value to
                    // look at, to determine the exact cause.  Also, be aware that we might now have a memory leak.
                    // Since we don't know exactly what happened, we would rather suffer that leak than attempt to
                    // decrement the reference count on the subsidiary object that we just tried to add to the array
                    // (if in fact it was non-NULL).
                    //
                    // Since adding one element to the array didn't work, we abort the process of trying to add any 
                    // additional elements to the array.  Instead, we just clear out the entire array, and we return
                    // a NULL value to indicate the error.
                    //
                    // A future version might print at least the failing key, if not also the failing value (which
                    // could be of some complicated type).
                    //
                    printf(FILE_LINE "ERROR:  cannot append an element to a JSON %s array\n", "transit_MonitoredResource_List");
                    json_array_clear(json);
                    json_decref(json);
                    json = NULL; 
                    break;
		}
	    }
	}
    }
    return json;
}

json_t *string_transit_TypedValue_Pair_List_as_JSON(const string_transit_TypedValue_Pair_List *string_transit_TypedValue_Pair_List) {
    json_t *json;
    if (string_transit_TypedValue_Pair_List == NULL) {
	json = NULL;
    } else if (string_transit_TypedValue_Pair_List->count == 0) {
	json = NULL;
    } else {
	json = json_object();
	if (json == NULL) {
	    printf(FILE_LINE "ERROR:  cannot create a JSON %s object\n", "string_transit_TypedValue_Pair_List");
	} else {
	    for (size_t i = 0; i < string_transit_TypedValue_Pair_List->count; ++i) {
		if (json_object_set_new(json
		    , string_transit_TypedValue_Pair_List->items[i].key                                  // string
		    , transit_TypedValue_as_JSON( &string_transit_TypedValue_Pair_List->items[i].value ) // transit_TypedValue
		) != 0) {
		    // FIX MAJOR:
		    // Report and handle the error condition.  Unfortunately, there is no json_error_t value to
		    // look at, to determine the exact cause.  Also, be aware that we might now have a memory leak.
		    // Since we don't know exactly what happened, we would rather suffer that leak than attempt to
		    // decrement the reference count on the subsidiary object that we just tried to add to the array
		    // (if in fact it was non-NULL).
		    //
		    // Since adding one key/value pair to the object didn't work, we abort the process of trying to
		    // add any additional key/value pairs to the object.  Instead, we just clear out the entire object,
		    // and we return a NULL value to indicate the error.
		    json_object_clear(json);
		    json_decref(json);
		    json = NULL;
		    break;
		}
	    }
	}
    }
    return json;
}

// This routine is here only as an example of something we might do when processing some other kind of Pair.
// It is not likely that we will actually need this construction.
json_t *string_transit_TypedValue_Pair_as_JSON(const string_transit_TypedValue_Pair *string_transit_TypedValue_Pair) {
    json_t *json;
    if (string_transit_TypedValue_Pair == NULL) {
	json = NULL;
    }
    else {
	json_error_t error;
	size_t flags = 0;
	json = json_pack_ex(&error, flags, "{s:s s:o}"
	    , "key",   string_transit_TypedValue_Pair->key                                  // string
	    , "value", transit_TypedValue_as_JSON( &string_transit_TypedValue_Pair->value ) // transit_TypedValue
	);
	if (json == NULL) {
	    printf(FILE_LINE "ERROR:  text '%s', source '%s', line %d, column %d, position %d\n",
		error.text, error.source, error.line, error.column, error.position);
	}
    }
    return json;
}

