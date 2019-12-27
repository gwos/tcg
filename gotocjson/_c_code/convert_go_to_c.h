// Code that is used during conversion of Go types, enumerations, and
// data structures, but is not specific to any one generated package.

#ifndef _CONVERT_GO_TO_C_H
#define _CONVERT_GO_TO_C_H

#ifdef  __cplusplus
extern "C" {
#endif

#include <stdbool.h>    // as of C99, provides the "bool" datatype, along with "true" and "false" macros
#include <stdint.h>     // as or C99, provides "int32_t" and "int64_t" datatypes
#include <time.h>       // to supply "struct timespec", with time_t tv_sec (seconds) and long tv_nsec (nanoseconds) members 
#include <sys/time.h>   // to supply "struct timeval",  with time_t tv_sec (seconds) and suseconds_t tv_usec (microseconds) members

#include "jansson.h"

#if !(JSON_INTEGER_IS_LONG_LONG)
// In addition to millisecond timestamps, 64-bit integers are presumed in some of the other Go structures we convert.
#error The Jansson JSON integer type may not hold 64 bits on this platform; 64 bits are needed for the milliseconds_MillisecondTimestamp type.
#endif

#ifndef NUL_TERM_LEN
// Size of a NUL-termination byte.  Generally useful for documenting the
// meaning of +1 and -1 length adjustments having to do with such bytes.
#define NUL_TERM_LEN 1  // sizeof('\0')
#endif  // NUL_TERM_LEN

// typedef int int;    // Go's "int" type is at least 32 bits in size; that might or might not be identical to C's "int" type
typedef int64_t int64;
typedef double  float64;
typedef int32_t int32;
typedef struct timespec struct_timespec;
typedef struct_timespec time_Time;

#ifndef string
// Make a simple global substitution using the C preprocessor, so we don't
// have to deal with this ad infinitum in the language-conversion code.
#define string char *
#endif  // string

#define stringify(x)                    #x
#define expand_and_stringify(x)         stringify(x)

// FILE_LINE is defined so you can just say:
// log_message (APP_FATAL, FILE_LINE "Insufficient memory for %s; exiting!", foobar);
// (Notice the lack of a comma after the FILE_LINE invocation.)
//
#define FILE_LINE       __FILE__ "[" expand_and_stringify(__LINE__) "] "

#define arraysize(array) (sizeof(array) / sizeof(array[0]))

// FIX MAJOR:  output these definitions in the boilerplate for the header file that defines the "struct_timespec" conversions
#define MILLISECONDS_PER_SECOND         1000
#define MICROSECONDS_PER_MILLISECOND    1000    
#define NANOSECONDS_PER_MICROSECOND     1000    
#define NANOSECONDS_PER_MILLISECOND     (NANOSECONDS_PER_MICROSECOND * MICROSECONDS_PER_MILLISECOND)

// If this routine fails to find any matching string within the array, it returns a negative result. 
// It doesn't log anything in that situation, both because the caller is going to need to check the
// result anyway for such an out-of-bound result, and because the calling code has a much better
// idea of the full context of what ought to be included in a log message.
extern int enumeration_value(const string const enum_string[], int enum_string_count, const char *enum_value_as_string);

extern char *typeof_json_item(const json_t *json);

// This routine decrements the reference count on the "json" object.  In many calling
// contexts, that will be the last reference to that object, so it will be destroyed.
// If you do want to keep the object around, call json_incref(json) before calling
// the JSON_as_string_ptr() routine.
extern string JSON_as_str(json_t *json, size_t flags);

#define string_ptr_as_JSON_ptr(string_ptr) json_string(*(string_ptr))

// FIX MAJOR:  This #define definition is tentative, and it's causing us some build problems.
// It is likely to be either revised or replaced with an actual separate routine.
// #define JSON_as_string_ptr(json) JSON_as_str(json, 0)
extern string *JSON_as_string_ptr(json_t *json);

extern bool is_bool_ptr_zero_value(const bool *bool_ptr);
extern bool is_int_ptr_zero_value(const int *int_ptr);
extern bool is_int32_ptr_zero_value(const int32 *int32_ptr);
extern bool is_int64_ptr_zero_value(const int64 *int64_ptr);
extern bool is_float64_ptr_zero_value(const float64 *float64_ptr);
extern bool is_string_ptr_zero_value(string const *string_ptr);
extern bool is_struct_timespec_ptr_zero_value(const struct_timespec *struct_timespec_ptr);

static inline struct_timespec time_t_to_struct_timespec(time_t moment) 
{
    struct_timespec clock = {moment, 0};
    return clock;
}

extern struct_timespec timeval_to_timespec(struct timeval timeval_timestamp);
extern json_t *struct_timespec_ptr_as_JSON_ptr(const struct_timespec *milliseconds_MillisecondTimestamp);
extern struct_timespec *JSON_as_struct_timespec(json_t *json);
#define time_Time_ptr_as_JSON_ptr struct_timespec_ptr_as_JSON_ptr
#define JSON_as_time_Time JSON_as_struct_timespec

// A routine that an application must eventually call to dispose of whatever JSON object
// got returned by conversion from a JSON string to Go-as-C data structures.  This call
// must be made only after the application is done with the returned C data structures.
extern void free_JSON(json_t *json);

#ifdef  __cplusplus
}
#endif

#endif // _CONVERT_GO_TO_C_H

