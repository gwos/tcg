#include <dlfcn.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

#include "libtransit.h" /* ERROR_LEN */
#include "util.h"

#ifndef NUL_TERM_LEN
/* Size of a NUL-termination byte. Generally useful for documenting the meaning
 * of +1 and -1 length adjustments having to do with such bytes. */
#define NUL_TERM_LEN 1 /*  sizeof('\0') */
#endif                 /* NUL_TERM_LEN */

/*
https://medium.com/@ben.mcclelland/an-adventure-into-cgo-calling-go-code-with-c-b20aa6637e75
https://medium.com/learning-the-go-programming-language/calling-go-functions-from-other-languages-4c7d8bcc69bf

The test supports environment variables:
    LIBTRANSIT=/path/to/libtransit.so
    TEST_ENDLESS - cycle run test_dlSendResourcesWithMetrics
And all environment variables supported by libtransit itself:
    TNG_CONFIG=/path/to/config.yml
    TNG_AGENTCONFIG_NATSSTORETYPE=MEMORY
For more info see package `config`
*/

char *listMetricsHandler() {
  char *payload = "{\"key\":\"value\"}";

  size_t bufLen = strlen(payload) + NUL_TERM_LEN;
  char *buf = malloc(bufLen);
  strcpy(buf, payload);

  printf("\nlistMetricsHandler: %s : %ld", buf, bufLen);
  return buf;
}

void test_dlRegisterListMetricsHandler() {
  void *handle;
  char *error;

  char *libtransit = getenv("LIBTRANSIT");
  if (!libtransit) {
    libtransit = "libtransit.so";
  }

  handle = dlopen(libtransit, RTLD_LAZY);
  if (!handle) {
    fail(dlerror());
  }

  void (*registerListMetricsHandler)(char *(*)()) =
      dlsym(handle, "RegisterListMetricsHandler");
  if ((error = dlerror()) != NULL) {
    fail(error);
  }

  registerListMetricsHandler(listMetricsHandler);

  dlclose(handle);
}

void test_dlSendResourcesWithMetrics() {
  void *handle;
  char *error;

  char *libtransit = getenv("LIBTRANSIT");
  if (!libtransit) {
    libtransit = "libtransit.so";
  }

  handle = dlopen(libtransit, RTLD_LAZY);
  if (!handle) {
    fail(dlerror());
  }

  bool (*sendResourcesWithMetrics)(char *, char *) =
      dlsym(handle, "SendResourcesWithMetrics");
  if ((error = dlerror()) != NULL) {
    fail(error);
  }

  /* TODO: should be serialized ResourceWithMetricsRequest */
  char *sample_request_json =
      "{\"name\": \"the-unique-name-of-the-instance-01\", \"status\": "
      "\"HOST_UP\", \"type\": \"gce_instance\"}";

  char err[ERROR_LEN] = "";
  bool res = sendResourcesWithMetrics(sample_request_json, (char *)&err);
  printf("\n_res:err: %d : %s", res, err);
  dlclose(handle);
}

int main(void) {
  test_dlRegisterListMetricsHandler();
  test_dlSendResourcesWithMetrics();

  fprintf(stdout, "\nall tests passed");

  if (getenv("TEST_ENDLESS") != NULL) {
    fprintf(stdout, "\nTEST_ENDLESS: press ctrl-c to exit");
    while (1) {
      sleep(3);
      test_dlSendResourcesWithMetrics();
    }
  }

  return 0;
}
