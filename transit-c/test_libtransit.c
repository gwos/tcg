#include <dlfcn.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

#include "libtransit.h" /* refs */
#include "util.h"

#ifndef NUL_TERM_LEN
/* Size of a NUL-termination byte. Generally useful for documenting the meaning
 * of +1 and -1 length adjustments having to do with such bytes. */
#define NUL_TERM_LEN 1 /*  sizeof('\0') */
#endif                 /* NUL_TERM_LEN */

#ifndef ERR_BUF_LEN
#define ERR_BUF_LEN 250 /* buffer for error message */
#endif                  /* ERR_BUF_LEN */

/*
https://medium.com/@ben.mcclelland/an-adventure-into-cgo-calling-go-code-with-c-b20aa6637e75
https://medium.com/learning-the-go-programming-language/calling-go-functions-from-other-languages-4c7d8bcc69bf

The test supports environment variables:
    LIBTRANSIT=/path/to/libtransit.so
    TEST_ENDLESS - cycle run test_dlSendResourcesWithMetrics
And all environment variables supported by libtransit itself:
    TCG_CONFIG=/path/to/tcg_config.yaml
    TCG_AGENTCONFIG_NATSSTORETYPE=MEMORY
For more info see package `config`
*/

/* define handlers for extended libtransit functions */
uintptr_t (*createInventoryRequest)() = NULL;
uintptr_t (*createInventoryResource)(char *name, char *resType) = NULL;
uintptr_t (*createInventoryService)(char *name, char *resType) = NULL;
uintptr_t (*createMonitoredResource)(char *name, char *resType) = NULL;
uintptr_t (*createMonitoredService)(char *name, char *resType) = NULL;
void (*addResource)(uintptr_t *pReq, uintptr_t pRes) = NULL;
void (*addService)(uintptr_t *pRes, uintptr_t pSvc) = NULL;
void (*setName)(uintptr_t *handle, char *s) = NULL;
void (*deleteHandle)(uintptr_t *handle) = NULL;
bool (*sendInventory)(uintptr_t pInvReq, char *errBuf, size_t errBufLen) = NULL;

/* define handlers for general libtransit functions */
bool (*getAgentIdentity)(char *buf, size_t bufLen, char *errBuf,
                         size_t errBufLen) = NULL;
bool (*goSetenv)(char *key, char *val, char *errBuf, size_t errBufLen) = NULL;
void (*registerListMetricsHandler)(char *(*)()) = NULL;
bool (*sendEvents)(char *payloadJSON, char *errBuf, size_t errBufLen) = NULL;
bool (*sendResourcesWithMetrics)(char *payloadJSON, char *errBuf,
                                 size_t errBufLen) = NULL;
bool (*synchronizeInventory)(char *payloadJSON, char *errBuf,
                             size_t errBufLen) = NULL;
void (*registerDemandConfigHandler)(bool (*)()) = NULL;

/* define handlers for other libtransit functions */
bool (*isControllerRunning)() = NULL;
bool (*isNatsRunning)() = NULL;
bool (*isTransportRunning)() = NULL;
bool (*startController)(char *errBuf, size_t errBufLen) = NULL;
bool (*startNats)(char *errBuf, size_t errBufLen) = NULL;
bool (*startTransport)(char *errBuf, size_t errBufLen) = NULL;
bool (*stopController)(char *errBuf, size_t errBufLen) = NULL;
bool (*stopNats)(char *errBuf, size_t errBufLen) = NULL;
bool (*stopTransport)(char *errBuf, size_t errBufLen) = NULL;

/* list_metrics_handler implements getTextHandlerType */
char *list_metrics_handler() {
  char *payload = "{\"key\":\"value\"}";

  size_t bufLen = strlen(payload) + NUL_TERM_LEN;
  char *buf = malloc(bufLen);
  strcpy(buf, payload);

  printf("\nlist_metrics_handler: %s : %ld", buf, bufLen);
  return buf;
}

bool demand_config_handler() {
  printf("DemandConfig was called by the TCG\n");
  return true;
}

void *libtransit_handle;

// Follow the full approved procedure for finding a symbol, including all the
// correct error detection. This allows us to immediately identify any specific
// symbol we're having trouble with.
void *find_symbol(char *symbol) {
  dlerror();  // clear any old error condition, before looking for the symbol
  void *address = dlsym(libtransit_handle, symbol);
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

  createInventoryRequest = find_symbol("CreateInventoryRequest");
  createInventoryResource = find_symbol("CreateInventoryResource");
  createInventoryService = find_symbol("CreateInventoryService");
  createMonitoredResource = find_symbol("CreateMonitoredResource");
  createMonitoredService = find_symbol("CreateMonitoredService");
  addResource = find_symbol("AddResource");
  addService = find_symbol("AddService");
  setName = find_symbol("SetName");
  deleteHandle = find_symbol("DeleteHandle");
  sendInventory = find_symbol("SendInventory");

  getAgentIdentity = find_symbol("GetAgentIdentity");
  goSetenv = find_symbol("GoSetenv");
  registerListMetricsHandler = find_symbol("RegisterListMetricsHandler");
  sendEvents = find_symbol("SendEvents");
  sendResourcesWithMetrics = find_symbol("SendResourcesWithMetrics");
  isControllerRunning = find_symbol("IsControllerRunning");
  isNatsRunning = find_symbol("IsNatsRunning");
  isTransportRunning = find_symbol("IsTransportRunning");
  startController = find_symbol("StartController");
  startNats = find_symbol("StartNats");
  startTransport = find_symbol("StartTransport");
  stopController = find_symbol("StopController");
  stopNats = find_symbol("StopNats");
  stopTransport = find_symbol("StopTransport");
  registerDemandConfigHandler = find_symbol("RegisterDemandConfigHandler");
}

void test_libtransit_control() {
  char errBuf[ERR_BUF_LEN] = "";
  bool res = false;

  char *tcg_config = getenv("TCG_CONFIG");
  if (!tcg_config) {
    res = goSetenv("TCG_CONFIG", "../tcg_config.yaml", errBuf, ERR_BUF_LEN);
    if (!res) {
      fail(errBuf);
    }
  }

  // We allow our forced selection here to be externally overridden, but in
  // general for casual test purposes, we don't want the FILE type (as is
  // specified by our ../tcg_config.yaml file) to be in operation during this
  // testing, as that will cause a buildup of queued items as this test is run
  // and re-run.
  char *nats_stor_type = getenv("TCG_CONNECTOR_NATSSTORETYPE");
  if (!nats_stor_type) {
    res =
        goSetenv("TCG_CONNECTOR_NATSSTORETYPE", "MEMORY", errBuf, ERR_BUF_LEN);
    if (!res) {
      fail(errBuf);
    }
  }

  // Check config
  printf("Testing GetAgentIdentity ...\n");
  char buf[1024] = "";
  res = getAgentIdentity(buf, 1024, errBuf, ERR_BUF_LEN);
  if (!res) {
    fail(errBuf);
  }
  printf("\n%s\n\n", buf);

  // If true, force a test of StopNats(), to see if it will generate a segfault
  // if NATS has not previously been started.  If false, skip testing the
  // StopNats() routine ut the beginning, before NATS has been started.
  bool test_StopNats = false;

  // If true, force a test of StopController(), to see if it will generate a
  // segfault if the controller has not previously been started. If false, skip
  // testing the StopController() routine at the beginning, before the
  // controller has been started.
  bool test_StopController = false;

  // Since we start the controller before starting NATS, it seems to make sense
  // that we should stop NATS before stopping the controller.

  // At least in our stack as of this writing, StopNats() will segfault inside
  // the github.com/nats-io/nats-streaming-server/server.(*StanServer).Shutdown
  // code if NATS has not previously been started.  The
  // github.com/gwos/tcg/nats.StopServer() code should be revised to not pass
  // stuff to NATS that causes it to segfault.
  if (test_StopNats) {
    printf("Testing StopNats ...\n");
    res = stopNats(errBuf, ERR_BUF_LEN);
    if (!res) {
      fail(errBuf);
    }
  } else {
    printf("Skipping test of StopNats.\n");
  }

  // At least in our stack as of this writing, StopController() will segfault
  // inside the net/http. (*Server).Shutdown() code if the controller has not
  // previously been started.  The github.com/gwos/tcg/services.
  // (*Controller).StopController() code should be revised to not pass stuff to
  // net/http that causes it to segfault.
  if (test_StopController) {
    printf("Testing StopController ...\n");
    res = stopController(errBuf, ERR_BUF_LEN);
    if (!res) {
      fail(errBuf);
    }
  } else {
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
  res = startController(errBuf, ERR_BUF_LEN);
  if (!res) {
    fail(errBuf);
  }

  printf("Testing StartNats ...\n");
  res = startNats(errBuf, ERR_BUF_LEN);
  if (!res) {
    fail(errBuf);
  }

  // StartNats() should have already started the transport,
  // so this call should be safely idempotent.
  printf("Testing StartTransport ...\n");
  res = startTransport(errBuf, ERR_BUF_LEN);
  if (!res) {
    fail(errBuf);
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
  registerListMetricsHandler(list_metrics_handler);

  printf("Testing StopNats ...\n");
  res = stopNats(errBuf, ERR_BUF_LEN);
  if (!res) {
    fail(errBuf);
  }

  printf("Testing StopController ...\n");
  res = stopController(errBuf, ERR_BUF_LEN);
  if (!res) {
    fail(errBuf);
  }

  // We want to leave Transit running, for later tests outside of this routine.
  // So we restart it here.

  printf("Testing StartController ...\n");
  res = startController(errBuf, ERR_BUF_LEN);
  if (!res) {
    fail(errBuf);
  }

  printf("Testing StartNats ...\n");
  res = startNats(errBuf, ERR_BUF_LEN);
  if (!res) {
    fail(errBuf);
  }

  printf("Testing startTransport ...\n");
  res = startTransport(errBuf, ERR_BUF_LEN);
  if (!res) {
    fail(errBuf);
  }

  printf("Testing registerDemandConfigHandler ...\n");
  registerDemandConfigHandler(demand_config_handler);

  system("sh ./transit-c/send_config_script.sh");
}

void test_SendResourcesWithMetrics() {
  /* TODO: should be serialized ResourceWithMetricsRequest */
  char *payloadJSON =
      "{\"name\": \"the-unique-name-of-the-instance-01\", \"status\": "
      "\"HOST_UP\", \"type\": \"gce_instance\"}";

  char errBuf[ERR_BUF_LEN] = "";
  bool res = sendResourcesWithMetrics(payloadJSON, errBuf, ERR_BUF_LEN);
  if (!res) {
    fail(errBuf);
  }
}

void test_SendInventory() {
  uintptr_t invReq = createInventoryRequest();
  uintptr_t invRes = createInventoryResource("invRes", "host");
  uintptr_t invSvc = createInventoryService("invSvc", "service");
  uintptr_t monSvc = createMonitoredService("monSvc", "service");

  setName(&invRes, "resource-01");
  setName(&invSvc, "service-01");
  setName(&monSvc, "service-02");

  addService(&invRes, invSvc);
  addService(&invRes, monSvc);
  addResource(&invReq, invRes);

  char errBuf[ERR_BUF_LEN] = "";
  bool res = sendInventory(invReq, errBuf, ERR_BUF_LEN);

  deleteHandle(&invReq);
  deleteHandle(&invRes);
  deleteHandle(&invSvc);
  deleteHandle(&monSvc);

  if (!res) {
    fail(errBuf);
  }
}

int main(void) {
  load_libtransit();
  test_SendInventory();

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
