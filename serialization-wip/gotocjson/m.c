#include "convert_go_to_c.h"
#include "milliseconds.h"

json_t *milliseconds_MillisecondTimestamp_as_JSON(const milliseconds_MillisecondTimestamp *milliseconds_MillisecondTimestamp) {
    json_error_t error;
    size_t flags = 0;
    json_t *json;
    // FIX MAJOR:  when generating this code, we must special-case the field packing in this routine, based on the "struct_timespec" field type
    // FIX MAJOR:  make sure the "I" conversion can handle a 64-bit number
    json = json_pack_ex(&error, flags, "I"
         // struct_timespec time_Time_;  // go:  time.Time
	 , (json_int_t) (
	     (milliseconds_MillisecondTimestamp->time_Time_.tv_sec  * MILLISECONDS_PER_SECOND) +
	     (milliseconds_MillisecondTimestamp->time_Time_.tv_nsec / NANOSECONDS_PER_MILLISECOND)
	 )
    );
    if (json == NULL) {
	printf(FILE_LINE "ERROR:  text '%s', source '%s', line %d, column %d, position %d\n",
	    error.text, error.source, error.line, error.column, error.position);
    }
    return json;
}

/*
typedef struct _milliseconds_MillisecondTimestamp_ {
    struct_timespec time_Time_;  // go:  time.Time
} milliseconds_MillisecondTimestamp;
*/
milliseconds_MillisecondTimestamp *JSON_as_milliseconds_MillisecondTimestamp(json_t *json) {
    milliseconds_MillisecondTimestamp *MillisecondTimestamp = (milliseconds_MillisecondTimestamp *)malloc(sizeof(milliseconds_MillisecondTimestamp));
    if (!MillisecondTimestamp) {
	// FIX MAJOR:  invoke proper logging for error conditions
	fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_milliseconds_MillisecondTimestamp, %s\n", "malloc failed");
    } else {
	// FIX MAJOR:  when generating this code, special-case the field unpacking in this routine, based on the "struct_timespec" field type
	json_int_t pure_milliseconds;
	if (json_unpack(json, "I"
	    , &pure_milliseconds
	) != 0) {
	    // FIX MAJOR:  invoke proper logging for error conditions
	    fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_milliseconds_MillisecondTimestamp, %s\n", "JSON unpacking failed");
	    free(MillisecondTimestamp);
	    MillisecondTimestamp = NULL;
	} else {
	    MillisecondTimestamp->time_Time_.tv_sec  = (time_t) (pure_milliseconds / MILLISECONDS_PER_SECOND);
	    MillisecondTimestamp->time_Time_.tv_nsec = (long) (pure_milliseconds % MILLISECONDS_PER_SECOND) * NANOSECONDS_PER_MILLISECOND;
	}
    }
    return MillisecondTimestamp;
}
