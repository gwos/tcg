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

// Global variable to be referenced internally in TCG conversion routines
// when they need to understand whether to convert strings to or from UTF-8.
extern int C_strings_use_utf8;

// Specify the character set encoding that strings on the C side use, at any
// time after initialize_data_conversions() is first called.  If this routine
// is not called, the effect will be as though the "ISO-8859-1" encoding was
// selected.  Argument values that are recognized as specifying that C strings
// already use UTF-8 encoding are "UTF-8" and "UTF8"; anything else will be
// treated as requiring an encoding conversion during data serialization and
// deserialization.
extern int set_C_string_encoding(char *encoding);

// Routine to scan a string and tell if it contains only 7-bit ASCII characters.
// This is the simplest means to bypass more-complex processing which will only
// be necessary if the upper half of the ISO-8859-1 character set is in use.
// Returns a true or false value, reflecting the content of the c_string.
extern int string_is_ascii(char *c_string);

// Conversion routines, to change character-set encodings.  The result strings
// are provided in static memory which will be overwritten on the next call to
// either routine, so these routines are neither reentrant (i.e., they cannot be
// called from separate threads), nor is their result data long-term persistent.
// They are intended for very short-term use in conversions where the resulting
// string will be used immediately and then this copy will be otherwise forgotten.
extern char *C_string_to_UTF_8(char *c_string);
extern char *UTF_8_to_C_string(char *utf_8_string);

// Like UTF_8_to_C_string(), but converts the string in-place.  This is safe
// because we will only at most reduce the number of bytes used in the string.
// Always converts as best it can, even if there were conversion errors.
// Returns the length of the resulting string if no errors were encountered,
// or -1 if some byte sequence was found that did not correspond to the target
// character set.
extern int UTF_8_to_C_string_in_place(char *utf_8_string);

// Global variables to be referenced internally in TCG conversion routines
// when they need to log errors.  Not for external use.
extern void (*external_logging_function)(void *arg, const char *format_string, ...);
extern void *external_logging_first_arg;

// A routine that an application can call to register a logging function to be
// used by TCG conversion routines to report errors.  When logging occurs, the
// actual_arg will be passed as the first argument to logging_function().  This
// registration routine returns zero on success, -1 on failure to register the
// logging function.  If no such logging function is registered, the effect will
// be as though register_logging_callback(&fprintf, stderr) had been called at
// the time that initialize_data_conversions() was called.  This means that it
// is the value of stderr at that moment that will rule by default (which may
// be important for applications that internally modify stderr at some point).
extern int register_logging_callback(void (*logging_function)(void *arg, const char *format_string, ...), void *actual_arg);

// A routine that should be called at application startup, before running any
// data-structure conversions.  This will take care of setting initial state
// which is necessary to make all the conversions operate safely.
extern void initialize_data_conversions();

// A routine that should be called at application end, after running all
// data-structure conversions, whether run directly by the application itself
// or by any callbacks triggered from outside the application.
extern void terminate_data_conversions();

#ifdef  __cplusplus
}
#endif

#endif // _CONVERT_GO_TO_C_H

