// Code that is used during conversion of Go types, enumerations, and
// data structures, but is not specific to any one generated package.

#include <string.h>	// for strcmp()

#include "convert_go_to_c.h"

// If this routine fails to find any matching string within the array, it returns a negative result. 
// It doesn't log anything in that situation, both because the caller is going to need to check the
// result anyway for such an out-of-bound result, and because the calling code has a much better
// idea of the full context of what ought to be included in a log message.
int enumeration_value(const string const enum_string[], int enum_string_count, const char *enum_value_as_string) {
    int enum_value;
    // We don't bother to test against the value at index 0 because we treat that slot as a
    // universally-matching wildcard.  That provides a mechanism for designating an unknown value.
    for (enum_value = enum_string_count; --enum_value > 0; ) {
        if (!strcmp(enum_value_as_string, enum_string[enum_value])) {
            break;
        }
    }
    return enum_value;
}

char *typeof_json_item(const json_t *json) {
    if (json == NULL) { return "NULL pointer"; }
    int type = json_typeof(json);
    switch (type) {
        case JSON_OBJECT : return "JSON_OBJECT";
        case JSON_ARRAY  : return "JSON_ARRAY";
        case JSON_STRING : return "JSON_STRING";
        case JSON_INTEGER: return "JSON_INTEGER";
        case JSON_REAL   : return "JSON_REAL";
        case JSON_TRUE   : return "JSON_TRUE";
        case JSON_FALSE  : return "JSON_FALSE";
        case JSON_NULL   : return "JSON_NULL";
    }
    static char buf[100];
    sprintf(buf, "UNKNOWN JSON TYPE %d", type);
    return buf;
}

// As a convenience for the caller, JSON_as_string() eats what is probably the last reference to the
// "json" object that is passed in.  That circumstance needs to be understood if you want to produce
// a JSON string in some context where you want the JSON object to stick around afterward.  In that
// case, you must call json_incref(json) before calling the JSON_as_str() routine.
string JSON_as_str(json_t *json, size_t flags) {
    char *result;
    if (!flags) {
        // FIX MAJOR:  These settings are for initial development use.
        // We probably want to change them for production use.
        // flags = JSON_SORT_KEYS | JSON_INDENT(4) | JSON_ENSURE_ASCII;
        flags = JSON_INDENT(4) | JSON_ENSURE_ASCII;
    }
    if (json == NULL) {
        // FIX MAJOR:  this message is just for development use, to track down the true source of failure
        printf(FILE_LINE "in JSON_as_str, received a NULL pointer\n");
        result = NULL;
    }
    else {
        result = json_dumps(json, flags);
        json_decref(json);
    }
    return result;
}

// This implementation must be supplied outside of anything generated by our automatic conversion,
// because it is generic to all packages and not specific to any one converted package.  The
// implementation seems a bit squirrely, in that it allocates and populates a single pointer which
// in turn contains the address of the string, and it passes back the address of that allocated
// pointer (that points to the actual string).  This extra level of indirection (as opposed to
// directly passing back the "char *" address of the string instead), helps to keep consistency with
// the ways in which our generated conversion routines are called, so we don't have to special-case
// calls to this routine.  The cost of that consistency will be borne at run time, namely with the
// extra allocate/deallocate steps for the extra intermediate pointer cell.  We might in the future
// revisit this construction and allow ourselves to do such special-case optimization.
//
// Note that it will be up to the caller to dispose of (that is, free()) the pointer cell we allocate
// here and pass back, if we pass back a non-NULL pointer to that cell.
//
// FIX MAJOR:  Calling JSON_as_str() is perhaps a simplified implementation (but I don't understand
// yet why it's not working; that needs some diagnosis).  We could instead use our belief that we
// have in hand just a string-valued JSON object, and call either
// "const string str_ptr = json_string_value(json);" or do this:
/*
        string string;
        if (json_unpack(json, "s"
            , &string
        ) != 0) { 
	    ...
	} else {
	    ...
	}
*/
//
// FIX MAJOR:  Check what eventually happens to the memory allocated for the string itself.  Is that
// a pointer to memory controlled by the JSON object, that we never have to worry about free()ing
// separately ourselves?  Or do we need to keep track of it, and eventually run our own free() call?
//
string *JSON_as_string(json_t *json) {
    // string str_ptr = JSON_as_str(json, 0);
    const string str_ptr = json_string_value(json);
    string * string_ptr;
    if (str_ptr == NULL) {
	string_ptr = NULL;
    } else {
	string_ptr = (string *) malloc(sizeof(string));
	if (string_ptr != NULL) {
	    *(const string *)string_ptr = str_ptr;
	}
    }
    return string_ptr;
}

bool is_bool_zero_value(const bool *bool_ptr) {
    return (
        bool_ptr == NULL || *bool_ptr == false
    );
}

bool is_int_zero_value(const int *int_ptr) {
    return (
        int_ptr == NULL || *int_ptr == 0
    );
}

bool is_int32_zero_value(const int32 *int32_ptr) {
    return (
        int32_ptr == NULL || *int32_ptr == 0
    );
}

bool is_int64_zero_value(const int64 *int64_ptr) {
    return (
        int64_ptr == NULL || *int64_ptr == 0
    );
}

bool is_float64_zero_value(const float64 *float64_ptr) {
    return (
        float64_ptr == NULL || *float64_ptr == 0.0
    );
}

bool is_string_zero_value(string const *string_ptr) {
    return (
        string_ptr == NULL || *string_ptr == NULL || **string_ptr == '\0'
    );
}

bool is_struct_timespec_zero_value(const struct_timespec *struct_timespec_ptr) {
    return (
	struct_timespec_ptr == NULL || (
	    struct_timespec_ptr->tv_sec  == 0 &&
	    struct_timespec_ptr->tv_nsec == 0
	)
    );
}

json_t *struct_timespec_as_JSON(const struct_timespec *struct_timespec) {
    json_error_t error;
    size_t flags = 0;
    json_t *json;
    // FIX MAJOR:  when generating this code, we must special-case the field packing in this routine, based on the "struct_timespec" field type
    // FIX MAJOR:  make sure the "I" conversion can handle a 64-bit number
    json = json_pack_ex(&error, flags, "I"
	 , (json_int_t) (
	     (struct_timespec->tv_sec  * MILLISECONDS_PER_SECOND) +
	     (struct_timespec->tv_nsec / NANOSECONDS_PER_MILLISECOND)
	 )
    );
    if (json == NULL) {
	printf(FILE_LINE "ERROR:  text '%s', source '%s', line %d, column %d, position %d\n",
	    error.text, error.source, error.line, error.column, error.position);
    }
    return json;
}

struct_timespec *JSON_as_struct_timespec(json_t *json) {
    struct_timespec *timespec = (struct_timespec *)malloc(sizeof(struct_timespec));
    if (!timespec) {
	// FIX MAJOR:  invoke proper logging for error conditions
	fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_struct_timespec, %s\n", "malloc failed");
    } else {
	// FIX MAJOR:  when generating this code, special-case the field unpacking in this routine, based on the "struct_timespec" field type
	json_int_t pure_milliseconds;
	if (json_unpack(json, "I"
	    , &pure_milliseconds
	) != 0) {
	    // FIX MAJOR:  invoke proper logging for error conditions
	    fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_struct_timespec, %s\n", "JSON unpacking failed");
	    free(timespec);
	    timespec = NULL;
	} else {
	    timespec->tv_sec  = (time_t) (pure_milliseconds / MILLISECONDS_PER_SECOND);
	    timespec->tv_nsec = (long) (pure_milliseconds % MILLISECONDS_PER_SECOND) * NANOSECONDS_PER_MILLISECOND;
	}
    }
    return timespec;
}

void free_JSON(json_t *json) {
    if (json != NULL) {
        json_decref(json);
    }
}
