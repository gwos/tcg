#ifndef TRANSIT_JSON_H
#define TRANSIT_JSON_H

#ifndef NUL_TERM_LEN
/* Size of a NUL-termination byte. Generally useful for documenting the meaning
 * of +1 and -1 length adjustments having to do with such bytes. */
#define NUL_TERM_LEN 1 /*  sizeof('\0') */
#endif                 /* NUL_TERM_LEN */

#include <stdlib.h>

#include "transit.h"

/* Returns new reference or NULL on error.
 The return value must be freed by the caller using free(). */
MonitoredResource *decodeMonitoredResource(const char *str);

/* Returns the JSON representation of json as a string, or NULL on error.
The return value must be freed by the caller using free().
flags is described in `char *json_dumps(const json_t *json, size_t flags)`
The JSON_SORT_KEYS is used by default. */
char *encodeMonitoredResource(const MonitoredResource *resource, size_t flags);

#endif /* TRANSIT_JSON_H */
