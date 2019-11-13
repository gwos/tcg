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
    for (enum_value = enum_string_count; --enum_value >= 0; ) {
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
char *JSON_as_str(json_t *json, size_t flags) {
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
