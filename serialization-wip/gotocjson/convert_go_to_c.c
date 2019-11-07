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

char *JSON_as_string(json_t *json, size_t flags) {
    char *result;
    if (!flags) {
        // FIX MAJOR:  These settings are for initial development use.
        // We probably want to change them for production use.
        // flags = JSON_SORT_KEYS | JSON_INDENT(4) | JSON_ENSURE_ASCII;
        flags = JSON_INDENT(4) | JSON_ENSURE_ASCII;
    }
    if (json == NULL) {
        // FIX MAJOR:  this message is just for development use, to track down the true source of failure
        printf(FILE_LINE "in JSON_as_string, received a NULL pointer\n");
        result = NULL;
    }
    else {
        result = json_dumps(json, flags);
        json_decref(json);
    }
    return result;
}

void free_JSON(json_t *json) {
    if (json != NULL) {
        json_decref(json);
    }
}
