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
    TNG_CONFIG=/path/to/tng_config.yaml
    TNG_AGENTCONFIG_NATSSTORETYPE=MEMORY
For more info see package `config`
*/

/* define handlers for general libtransit functions */
int (*goSetenv)(char *, char *) = NULL;
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

void *libtransit_handle;

// Follow the full approved procedure for finding a symbol, including all the correct error detection.
// This allows us to immediately identify any specific symbol we're having trouble with.
void *find_symbol(char *symbol) {
  dlerror();  // clear any old error condition, before looking for the symbol
  void *address = dlsym( libtransit_handle, symbol );
  char *error = dlerror();
  if (error != NULL) {
    fail(error);
  }
  return address;
}

void load_libtransit() {
  char *error;

  char *libtransit = getenv("LIBTRANSIT");
  if (!libtransit) {
    libtransit = "../libtransit/libtransit.so";
  }

  libtransit_handle = dlopen(libtransit, RTLD_LAZY);
  if (!libtransit_handle) {
    fail(dlerror());
  }

  goSetenv                   = find_symbol( "GoSetenv" );
  registerListMetricsHandler = find_symbol( "RegisterListMetricsHandler" );
  sendResourcesWithMetrics   = find_symbol( "SendResourcesWithMetrics" );
  isControllerRunning        = find_symbol( "IsControllerRunning" );
  isNatsRunning              = find_symbol( "IsNatsRunning" );
  isTransportRunning         = find_symbol( "IsTransportRunning" );
  startController            = find_symbol( "StartController" );
  startNats                  = find_symbol( "StartNats" );
  startTransport             = find_symbol( "StartTransport" );
  stopController             = find_symbol( "StopController" );
  stopNats                   = find_symbol( "StopNats" );
  stopTransport              = find_symbol( "StopTransport" );
}

void test_libtransit_control() {
  char errorBuf[ERROR_LEN] = "";
  bool res = false;

  char *tng_config = getenv("TNG_CONFIG");
  if (!tng_config) {
    if (goSetenv("TNG_CONFIG", "../tng_config.yaml") != 0) {
      fail("Could not set TNG_CONFIG");
    }
  }

  // We allow our forced selection here to be externally overridden, but in general
  // for casual test purposes, we don't want the FILE type (as is specified by our
  // ../tng_config.yaml file) to be in operation during this testing, as that will
  // cause a buildup of queued items as this test is run and re-run.
  char *nats_stor_type = getenv("TNG_AGENTCONFIG_NATSSTORETYPE");
  if (!nats_stor_type) {
    if (goSetenv("TNG_AGENTCONFIG_NATSSTORETYPE", "MEMORY") != 0) {
      fail("Could not set TNG_AGENTCONFIG_NATSSTORETYPE");
    }
  }

  // If true, force a test of StopNats(), to see if it will generate a segfault if
  // NATS has not previously been started.  If false, skip testing the StopNats()
  // routine ut the beginning, before NATS has been started.
  bool test_StopNats = false;

  // If true, force a test of StopController(), to see if it will generate a segfault
  // if the controller has not previously been started. If false, skip testing the
  // StopController() routine at the beginning, before the controller has been started.
  bool test_StopController = false;

  // Since we start the controller before starting NATS, it seems to make sense that
  // we should stop NATS before stopping the controller.

  // At least in our stack as of this writing, StopNats() will segfault inside the
  // github.com/nats-io/nats-streaming-server/server.(*StanServer).Shutdown code if
  // NATS has not previously been started.  The github.com/gwos/tng/nats.StopServer()
  // code should be revised to not pass stuff to NATS that causes it to segfault.
  if (test_StopNats) {
    printf("Testing StopNats ...\n");
    res = stopNats(errorBuf);
    if (!res) {
      fail(errorBuf);
    }
  }
  else {
    printf("Skipping test of StopNats.\n");
  }

  // At least in our stack as of this writing, StopController() will segfauult inside
  // the net/http.(*Server).Shutdown() code if the controller has not previously been
  // started.  The github.com/gwos/tng/services.(*Controller).StopController() code
  // should be revised to not pass stuff to net/http that causes it to segfault.
  if (test_StopController) {
    printf("Testing StopController ...\n");
    res = stopController(errorBuf);
    if (!res) {
      fail(errorBuf);
    }
  }
  else {
    printf("Skipping test of StopController.\n");
  }

  printf("sleeping for 5 seconds ...\n");
  sleep(5);

  printf("Testing IsControllerRunning ...\n");
  res = isControllerRunning();
  if (res) {
    fail("Controller still running");
  }

  printf("Testing IsNatsRunning ...\n");
  res = isNatsRunning();
  if (res) {
    fail("Nats still running");
  }

  printf("Testing IsTransportRunning ...\n");
  res = isTransportRunning();
  if (res) {
    fail("Transport still running");
  }

  printf("Testing StartController ...\n");
  res = startController(errorBuf);
  if (!res) {
    fail(errorBuf);
  }

  printf("Testing StartNats ...\n");
  res = startNats(errorBuf);
  if (!res) {
    fail(errorBuf);
  }

  // StartNats() should have already started the transport,
  // so this call should be safely idempotent.
  printf("Testing StartTransport ...\n");
  res = startTransport(errorBuf);
  if (!res) {
    fail(errorBuf);
  }

  printf("sleeping for 5 seconds ...\n");
  sleep(5);

  printf("Testing IsControllerRunning ...\n");
  res = isControllerRunning();
  if (!res) {
    fail("Controller still not running");
  }

  printf("Testing IsNatsRunning ...\n");
  res = isNatsRunning();
  if (!res) {
    fail("Nats still not running");
  }

  printf("Testing IsTransportRunning ...\n");
  res = isTransportRunning();
  if (!res) {
    fail("Transport still not running");
  }

  printf("Testing RegisterListMetricsHandler ...\n");
  registerListMetricsHandler(listMetricsHandler);

  printf("Testing StopNats ...\n");
  res = stopNats(errorBuf);
  if (!res) {
    fail(errorBuf);
  }

  printf("Testing StopController ...\n");
  res = stopController(errorBuf);
  if (!res) {
    fail(errorBuf);
  }

  // We want to leave Transit running, for later tests outside of this routine.
  // So we restart it here.

  printf("Testing StartController ...\n");
  res = startController(errorBuf);
  if (!res) {
    fail(errorBuf);
  }

  printf("Testing StartNats ...\n");
  res = startNats(errorBuf);
  if (!res) {
    fail(errorBuf);
  }
}

void test_SendResourcesWithMetrics() {
  /* TODO: should be serialized ResourceWithMetricsRequest */
  char *requestJSON =
      "{\"name\": \"the-unique-name-of-the-instance-01\", \"status\": "
      "\"HOST_UP\", \"type\": \"gce_instance\"}";

  char errorBuf[ERROR_LEN] = "";
  bool res = sendResourcesWithMetrics(requestJSON, errorBuf);
  printf("sendResourcesWithMetrics: boolean result %d, error '%s'\n", res, errorBuf);
}

int main(void) {
  load_libtransit();
  test_libtransit_control();
  test_SendResourcesWithMetrics();

  printf("\n");
  printf("all tests passed\n");

  if (getenv("TEST_ENDLESS") != NULL) {
    fprintf(stderr, "\n\nTEST_ENDLESS: press ctrl-c to exit\n\n");
    while (1) {
      sleep(3);
      test_SendResourcesWithMetrics();
    }
  }

  // We can't call this until we're actually done with the dynamically loaded
  // code, as when we do so the reference count will drop to zero, no other
  // loaded library will use symbols in it, and the dynamically loaded library
  // will be unloaded.
  dlclose(libtransit_handle);

  return 0;
}
