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

/* define handlers for general libtransit functions */
void (*registerListMetricsHandler)(char *(*)()) = NULL;
bool (*sendResourcesWithMetrics)(char *requestJSON, char *errorBuf) = NULL;
bool (*synchronizeInventory)(char *requestJSON, char *errorBuf) = NULL;
/* define handlers for other libtransit functions */
bool (*isControllerRunning)() = NULL;
bool (*isNatsRunning)() = NULL;
bool (*isTransportRunning)() = NULL;
bool (*startController)(char *errorBuf) = NULL;
bool (*startNats)(char *errorBuf) = NULL;
bool (*startTransport)(char *errorBuf) = NULL;
bool (*stopController)(char *errorBuf) = NULL;
bool (*stopNats)(char *errorBuf) = NULL;
bool (*stopTransport)(char *errorBuf) = NULL;

/* listMetricsHandler implements getTextHandlerType */
char *listMetricsHandler() {
  char *payload = "{\"key\":\"value\"}";

  size_t bufLen = strlen(payload) + NUL_TERM_LEN;
  char *buf = malloc(bufLen);
  strcpy(buf, payload);

  printf("\nlistMetricsHandler: %s : %ld", buf, bufLen);
  return buf;
}

void test_dl_libtransit() {
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

  registerListMetricsHandler = dlsym(handle, "RegisterListMetricsHandler");
  sendResourcesWithMetrics = dlsym(handle, "SendResourcesWithMetrics");
  isControllerRunning = dlsym(handle, "IsControllerRunning");
  isNatsRunning = dlsym(handle, "IsNatsRunning");
  isTransportRunning = dlsym(handle, "IsTransportRunning");
  startController = dlsym(handle, "StartController");
  startNats = dlsym(handle, "StartNats");
  startTransport = dlsym(handle, "StartTransport");
  stopController = dlsym(handle, "StopController");
  stopNats = dlsym(handle, "StopNats");
  stopTransport = dlsym(handle, "StopTransport");

  if ((error = dlerror()) != NULL) {
    fail(error);
  }

  dlclose(handle);
}

void test_dl_libtransit_control() {
  char errorBuf[ERROR_LEN] = "";
  bool res = false;

  res = stopController(errorBuf);
  if (!res) {
    fail(errorBuf);
  }
  res = stopNats(errorBuf);
  if (!res) {
    fail(errorBuf);
  }

  printf("\n_sleep 5 sec");
  sleep(5);

  res = isControllerRunning();
  if (res) {
    fail("Controller still running");
  }
  res = isNatsRunning();
  if (res) {
    fail("Nats still running");
  }
  res = isTransportRunning();
  if (res) {
    fail("Transport still running");
  }
  res = startController(errorBuf);
  if (!res) {
    fail(errorBuf);
  }
  res = startNats(errorBuf);
  if (!res) {
    fail(errorBuf);
  }
  res = startTransport(errorBuf);
  if (!res) {
    fail(errorBuf);
  }

  printf("\n_sleep 5 sec");
  sleep(5);

  res = isControllerRunning();
  if (!res) {
    fail("Controller still not running");
  }
  res = isNatsRunning();
  if (!res) {
    fail("Nats still not running");
  }
  res = isTransportRunning();
  if (!res) {
    fail("Transport still not running");
  }

  registerListMetricsHandler(listMetricsHandler);
}

void test_dlSendResourcesWithMetrics() {
  /* TODO: should be serialized ResourceWithMetricsRequest */
  char *requestJSON =
      "{\"name\": \"the-unique-name-of-the-instance-01\", \"status\": "
      "\"HOST_UP\", \"type\": \"gce_instance\"}";

  char errorBuf[ERROR_LEN] = "";
  bool res = sendResourcesWithMetrics(requestJSON, errorBuf);
  printf("\n_res:errorBuf: %d : %s", res, errorBuf);
}

int main(void) {
  test_dl_libtransit();
  test_dl_libtransit_control();
  test_dlSendResourcesWithMetrics();

  fprintf(stdout, "\nall tests passed");

  if (getenv("TEST_ENDLESS") != NULL) {
    fprintf(stderr, "\n\nTEST_ENDLESS: press ctrl-c to exit\n\n");
    while (1) {
      sleep(3);
      test_dlSendResourcesWithMetrics();
    }
  }

  return 0;
}
