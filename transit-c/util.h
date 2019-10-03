#ifndef UTIL_H
#define UTIL_H

/* inspired by:
 * https://github.com/akheron/jansson/blob/master/test/suites/api/util.h */
#define failhdr fprintf(stderr, "FAIL %s:%s:%d: ", __FILE__, __FUNCTION__, __LINE__)

#define fail(msg)                 \
  do {                            \
    failhdr;                      \
    fprintf(stderr, "%s\n", msg); \
    exit(1);                      \
  } while (0)

#endif /* UTIL_H */
