#include <dlfcn.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

#include "libtransit.h" /* refs */
#include "transit.h"    /* consts */

#ifndef NUL_TERM_LEN
/* Size of a NUL-termination byte. Generally useful for documenting the meaning
 * of +1 and -1 length adjustments having to do with such bytes. */
#define NUL_TERM_LEN 1 /*  sizeof('\0') */
#endif                 /* NUL_TERM_LEN */

#ifndef ERR_BUF_LEN
#define ERR_BUF_LEN 250 /* buffer for error message */
#endif                  /* ERR_BUF_LEN */

#ifndef UTIL_H
#define UTIL_H
/* inspired by:
 * https://github.com/akheron/jansson/blob/master/test/suites/api/util.h */
#define failhdr \
  fprintf(stderr, "FAIL %s:%s:%d: ", __FILE__, __FUNCTION__, __LINE__)

#define fail(msg)                 \
  do {                            \
    failhdr;                      \
    fprintf(stderr, "%s\n", msg); \
    exit(1);                      \
  } while (0)
#endif /* UTIL_H */

/*
https://medium.com/@ben.mcclelland/an-adventure-into-cgo-calling-go-code-with-c-b20aa6637e75
https://medium.com/learning-the-go-programming-language/calling-go-functions-from-other-languages-4c7d8bcc69bf

Supported environment variables:
    LIBTRANSIT=/path/to/libtransit.so
And all environment variables supported by libtransit itself:
    TCG_CONFIG=/path/to/tcg_config.yaml
    TCG_CONNECTOR_NATSSTORETYPE=MEMORY
For more info see package `config`
*/

/* define handlers for extended libtransit functions */

void (*addMetric)(uintptr_t target, uintptr_t value) = NULL;
void (*addResource)(uintptr_t target, uintptr_t value) = NULL;
void (*addResourceGroup)(uintptr_t target, uintptr_t value) = NULL;
void (*addService)(uintptr_t target, uintptr_t value) = NULL;
void (*addThreshold)(uintptr_t target, uintptr_t value) = NULL;
void (*addThresholdDouble)(uintptr_t target, char *lbl, char *sType,
                           long long value) = NULL;
void (*addThresholdInt)(uintptr_t target, char *lbl, char *sType,
                        double value) = NULL;
void (*calcStatus)(uintptr_t target) = NULL;
uintptr_t (*createInventoryRequest)() = NULL;
uintptr_t (*createInventoryResource)(char *name, char *resType) = NULL;
uintptr_t (*createInventoryService)(char *name, char *resType) = NULL;
uintptr_t (*createMonitoredResource)(char *name, char *resType) = NULL;
uintptr_t (*createMonitoredService)(char *name, char *resType) = NULL;
uintptr_t (*createResourceGroup)(char *name, char *grType) = NULL;
uintptr_t (*createResourcesWithServicesRequest)() = NULL;
uintptr_t (*createThresholdValue)(char *lbl, char *sType) = NULL;
uintptr_t (*createTimeSeries)(char *name) = NULL;
void (*deleteHandle)(uintptr_t target) = NULL;
void (*setCategory)(uintptr_t target, char *value) = NULL;
void (*setContextTimestamp)(uintptr_t target, char *value) = NULL;
void (*setContextToken)(uintptr_t target, char *value) = NULL;
void (*setDescription)(uintptr_t target, char *value) = NULL;
void (*setDevice)(uintptr_t target, char *value) = NULL;
void (*setIntervalEnd)(uintptr_t target, long long sec, long long nsec) = NULL;
void (*setIntervalStart)(uintptr_t target, long long sec,
                         long long nsec) = NULL;
void (*setLastCheckTime)(uintptr_t target, long long sec,
                         long long nsec) = NULL;
void (*setLastPluginOutput)(uintptr_t target, char *value) = NULL;
void (*setName)(uintptr_t target, char *value) = NULL;
void (*setNextCheckTime)(uintptr_t target, long long sec,
                         long long nsec) = NULL;
void (*setOwner)(uintptr_t target, char *value) = NULL;
void (*setPropertyBool)(uintptr_t target, char *key, _Bool value) = NULL;
void (*setPropertyDouble)(uintptr_t target, char *key, double value) = NULL;
void (*setPropertyInt)(uintptr_t target, char *key, long long int value) = NULL;
void (*setPropertyStr)(uintptr_t target, char *key, char *value) = NULL;
void (*setPropertyTime)(uintptr_t target, char *key, long long sec,
                        long long nsec) = NULL;
void (*setSampleType)(uintptr_t target, char *value) = NULL;
void (*setStatus)(uintptr_t target, char *value) = NULL;
void (*setTag)(uintptr_t target, char *key, char *value) = NULL;
void (*setType)(uintptr_t target, char *value) = NULL;
void (*setUnit)(uintptr_t target, char *value) = NULL;
void (*setValueBool)(uintptr_t target, _Bool value) = NULL;
void (*setValueDouble)(uintptr_t target, double value) = NULL;
void (*setValueInt)(uintptr_t target, long long int value) = NULL;
void (*setValueStr)(uintptr_t target, char *value) = NULL;
void (*setValueTime)(uintptr_t target, long long sec, long long nsec) = NULL;
bool (*marshallIndentJSON)(uintptr_t target, char *prefix, char *indent,
                           char *buf, size_t bufLen, char *errBuf,
                           size_t errBufLen) = NULL;
bool (*sendInventory)(uintptr_t req, char *errBuf, size_t errBufLen) = NULL;
bool (*sendMetrics)(uintptr_t req, char *errBuf, size_t errBufLen) = NULL;

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
  printf("\n\n  list_metrics_handler called\n\n");

  char *payload = "{\"key\":\"value\"}";
  size_t bufLen = strlen(payload) + NUL_TERM_LEN;
  char *buf = malloc(bufLen);
  strcpy(buf, payload);
  return buf;
}

bool demand_config_handler() {
  printf("\n\n  demand_config_handler called\n\n");
  return true;
}

// Follow the full approved procedure for finding a symbol, including all the
// correct error detection. This allows us to immediately identify any specific
// symbol we're having trouble with.
void *find_symbol(void *handle, char *symbol) {
  dlerror();  // clear any old error condition, before looking for the symbol
  void *address = dlsym(handle, symbol);
  char *error = dlerror();
  if (error != NULL) {
    fail(error);
  }
  return address;
}

void *example_load_libtransit() {
  char *error;

  char *lib_path = getenv("LIBTRANSIT");
  if (!lib_path) {
    lib_path = "./libtransit.so";
  }

  void *lib_handle = dlopen(lib_path, RTLD_LAZY);
  if (!lib_handle) {
    fail(dlerror());
  }

  addMetric = find_symbol(lib_handle, "AddMetric");
  addResource = find_symbol(lib_handle, "AddResource");
  addResourceGroup = find_symbol(lib_handle, "AddResourceGroup");
  addService = find_symbol(lib_handle, "AddService");
  addThreshold = find_symbol(lib_handle, "AddThreshold");
  addThresholdDouble = find_symbol(lib_handle, "AddThresholdDouble");
  addThresholdInt = find_symbol(lib_handle, "AddThresholdInt");
  calcStatus = find_symbol(lib_handle, "CalcStatus");
  createInventoryRequest = find_symbol(lib_handle, "CreateInventoryRequest");
  createInventoryResource = find_symbol(lib_handle, "CreateInventoryResource");
  createInventoryService = find_symbol(lib_handle, "CreateInventoryService");
  createMonitoredResource = find_symbol(lib_handle, "CreateMonitoredResource");
  createMonitoredService = find_symbol(lib_handle, "CreateMonitoredService");
  createResourceGroup = find_symbol(lib_handle, "CreateResourceGroup");
  createResourcesWithServicesRequest =
      find_symbol(lib_handle, "CreateResourcesWithServicesRequest");
  createThresholdValue = find_symbol(lib_handle, "CreateThresholdValue");
  createTimeSeries = find_symbol(lib_handle, "CreateTimeSeries");
  deleteHandle = find_symbol(lib_handle, "DeleteHandle");
  setCategory = find_symbol(lib_handle, "SetCategory");
  setContextTimestamp = find_symbol(lib_handle, "SetContextTimestamp");
  setContextToken = find_symbol(lib_handle, "SetContextToken");
  setDescription = find_symbol(lib_handle, "SetDescription");
  setDevice = find_symbol(lib_handle, "SetDevice");
  setIntervalEnd = find_symbol(lib_handle, "SetIntervalEnd");
  setIntervalStart = find_symbol(lib_handle, "SetIntervalStart");
  setLastCheckTime = find_symbol(lib_handle, "SetLastCheckTime");
  setLastPluginOutput = find_symbol(lib_handle, "SetLastPluginOutput");
  setName = find_symbol(lib_handle, "SetName");
  setNextCheckTime = find_symbol(lib_handle, "SetNextCheckTime");
  setOwner = find_symbol(lib_handle, "SetOwner");
  setPropertyBool = find_symbol(lib_handle, "SetPropertyBool");
  setPropertyDouble = find_symbol(lib_handle, "SetPropertyDouble");
  setPropertyInt = find_symbol(lib_handle, "SetPropertyInt");
  setPropertyStr = find_symbol(lib_handle, "SetPropertyStr");
  setPropertyTime = find_symbol(lib_handle, "SetPropertyTime");
  setSampleType = find_symbol(lib_handle, "SetSampleType");
  setStatus = find_symbol(lib_handle, "SetStatus");
  setTag = find_symbol(lib_handle, "SetTag");
  setType = find_symbol(lib_handle, "SetType");
  setUnit = find_symbol(lib_handle, "SetUnit");
  setValueBool = find_symbol(lib_handle, "SetValueBool");
  setValueDouble = find_symbol(lib_handle, "SetValueDouble");
  setValueInt = find_symbol(lib_handle, "SetValueInt");
  setValueStr = find_symbol(lib_handle, "SetValueStr");
  setValueTime = find_symbol(lib_handle, "SetValueTime");
  marshallIndentJSON = find_symbol(lib_handle, "MarshallIndentJSON");
  sendInventory = find_symbol(lib_handle, "SendInventory");
  sendMetrics = find_symbol(lib_handle, "SendMetrics");

  getAgentIdentity = find_symbol(lib_handle, "GetAgentIdentity");
  goSetenv = find_symbol(lib_handle, "GoSetenv");
  registerListMetricsHandler =
      find_symbol(lib_handle, "RegisterListMetricsHandler");
  sendEvents = find_symbol(lib_handle, "SendEvents");
  sendResourcesWithMetrics =
      find_symbol(lib_handle, "SendResourcesWithMetrics");
  isControllerRunning = find_symbol(lib_handle, "IsControllerRunning");
  isNatsRunning = find_symbol(lib_handle, "IsNatsRunning");
  isTransportRunning = find_symbol(lib_handle, "IsTransportRunning");
  startController = find_symbol(lib_handle, "StartController");
  startNats = find_symbol(lib_handle, "StartNats");
  startTransport = find_symbol(lib_handle, "StartTransport");
  stopController = find_symbol(lib_handle, "StopController");
  stopNats = find_symbol(lib_handle, "StopNats");
  stopTransport = find_symbol(lib_handle, "StopTransport");
  registerDemandConfigHandler =
      find_symbol(lib_handle, "RegisterDemandConfigHandler");

  return lib_handle;
}

void example_libtransit_control() {
  char errBuf[ERR_BUF_LEN] = "";
  bool res = false;

  char *tcg_config = getenv("TCG_CONFIG");
  if (!tcg_config) {
    res = goSetenv("TCG_CONFIG", "/dev/null", errBuf, ERR_BUF_LEN);
    if (!res) {
      fail(errBuf);
    }
  }

  // We allow our forced selection here to be externally overridden, but in
  // general for casual test purposes, we don't want the FILE type (as is
  // specified by our ../tcg_config.yaml file) to be in operation during this
  // testing, as that will cause a buildup of queued items as this test is run
  // and re-run.
  char *nats_store_type = getenv("TCG_CONNECTOR_NATSSTORETYPE");
  if (!nats_store_type) {
    res =
        goSetenv("TCG_CONNECTOR_NATSSTORETYPE", "MEMORY", errBuf, ERR_BUF_LEN);
    if (!res) {
      fail(errBuf);
    }
  }

  char *controller_addr = getenv("TCG_CONNECTOR_CONTROLLERADDR");
  if (!controller_addr) {
    controller_addr = "0.0.0.0:9999";
    res = goSetenv("TCG_CONNECTOR_CONTROLLERADDR", controller_addr, errBuf,
                   ERR_BUF_LEN);
    if (!res) {
      fail(errBuf);
    }
  }

  char *controller_pin = getenv("TCG_CONNECTOR_CONTROLLERPIN");
  if (!controller_pin) {
    controller_pin = "9999";
    res = goSetenv("TCG_CONNECTOR_CONTROLLERPIN", controller_pin, errBuf,
                   ERR_BUF_LEN);
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

  printf("sleeping for 2 seconds ...\n");
  sleep(2);

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

  printf("Testing StartTransport ...\n");
  res = startTransport(errBuf, ERR_BUF_LEN);
  if (!res) {
    fail(errBuf);
  }

  printf("sleeping for 2 seconds ...\n");
  sleep(2);

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
    // expected on empty config
    fprintf(stderr, "%s\n", "Transport still not running");
  }

  printf("Testing RegisterListMetricsHandler ...\n");
  registerListMetricsHandler(list_metrics_handler);

  printf("Testing registerDemandConfigHandler ...\n");
  registerDemandConfigHandler(demand_config_handler);

  printf("Testing curl ...\n");
  if (system("curl --version") == 0) {
    char sh[1024] = "";
    printf("Testing metrics entrypoint ...\n");
    snprintf(sh, 1024,
             "curl -v -w'\n\n' -H'X-PIN:%s' 'http://%s/api/v1/metrics'",
             controller_pin, controller_addr);
    system(sh);
    printf("Testing config entrypoint ...\n");
    snprintf(sh, 1024,
             "curl -v -w'\n\n' -H'X-PIN:%s' -d '{}' 'http://%s/api/v1/config'",
             controller_pin, controller_addr);
    system(sh);
  }
}

void example_send_inventory() {
  uintptr_t invReq = createInventoryRequest();
  uintptr_t invRes =
      createInventoryResource("invRes", TRANSIT_RESOURCE_TYPE_HOST);
  uintptr_t invSvc =
      createInventoryService("invSvc", TRANSIT_RESOURCE_TYPE_SERVICE);
  uintptr_t monSvc =
      createMonitoredService("monSvc", TRANSIT_RESOURCE_TYPE_SERVICE);
  uintptr_t resGroup = createResourceGroup("group-01", TRANSIT_HOST_GROUP);

  setName(invRes, "resource-01");
  setName(invSvc, "service-01");
  setName(monSvc, "service-02");
  setPropertyBool(invRes, "prop-bool1", true);
  setPropertyBool(invRes, "prop-bool2", false);
  setPropertyDouble(invRes, "prop-double", 0.11);
  setPropertyInt(invRes, "prop-int", 11);
  setPropertyStr(invRes, "prop-str", "str-33");
  setPropertyTime(invRes, "prop-time", 1609372800, 0);

  addService(invRes, invSvc);
  addService(invRes, monSvc);  // do noting
  addResource(invReq, invRes);
  addResource(resGroup, invRes);
  addResourceGroup(invReq, resGroup);

  char errBuf[ERR_BUF_LEN] = "";
  bool res = sendInventory(invReq, errBuf, ERR_BUF_LEN);

  deleteHandle(invReq);
  deleteHandle(invRes);
  deleteHandle(invSvc);
  deleteHandle(monSvc);
  deleteHandle(resGroup);

  if (!res) {
    fail(errBuf);
  }
}

void example_send_metrics() {
  uintptr_t monReq = createResourcesWithServicesRequest();
  uintptr_t monRes =
      createMonitoredResource("monRes", TRANSIT_RESOURCE_TYPE_HOST);
  uintptr_t monSvc =
      createMonitoredService("monSvc", TRANSIT_RESOURCE_TYPE_SERVICE);
  uintptr_t invSvc =
      createInventoryService("invSvc", TRANSIT_RESOURCE_TYPE_SERVICE);
  uintptr_t resGroup = createResourceGroup("group-01", TRANSIT_HOST_GROUP);

  setName(monRes, "resource-01");
  setName(monSvc, "service-01");
  setName(invSvc, "service-02");
  setLastPluginOutput(monSvc, "last-plugin-output");
  setLastCheckTime(monSvc, 1609372800, 0);

  uintptr_t crit = createThresholdValue("lbl-crit", TRANSIT_CRITICAL);
  uintptr_t warn = createThresholdValue("lbl-warn", TRANSIT_WARNING);
  setValueInt(crit, 90);
  setValueInt(warn, 70);
  uintptr_t metric1 = createTimeSeries("metric-1");
  uintptr_t metric2 = createTimeSeries("metric-2");
  uintptr_t metric3 = createTimeSeries("metric-3");
  setIntervalEnd(metric1, 1609372800, 0);
  setIntervalEnd(metric2, 1609372800, 0);
  setIntervalEnd(metric3, 1609372800, 0);
  setValueInt(metric1, 10);
  setValueInt(metric2, 20);
  setValueInt(metric3, 30);

  addThreshold(metric1, crit);
  addThreshold(metric1, warn);
  addThreshold(metric2, crit);
  addThreshold(metric2, warn);
  addThreshold(metric3, crit);
  addThreshold(metric3, warn);

  addMetric(monSvc, metric1);
  addMetric(monSvc, metric2);
  addMetric(monSvc, metric3);
  deleteHandle(crit);
  deleteHandle(warn);
  deleteHandle(metric1);
  deleteHandle(metric2);
  deleteHandle(metric3);

  addService(monRes, invSvc);  // do noting
  addService(monRes, monSvc);

  calcStatus(monRes);

  addResource(monReq, monRes);
  addResource(resGroup, monRes);
  addResourceGroup(monReq, resGroup);

  char msg[1024 * 100] = "";
  char errBuf[ERR_BUF_LEN] = "";

  // print out json example
  marshallIndentJSON(monReq, "", "  ", msg, 1024 * 100, errBuf, ERR_BUF_LEN);
  printf("%s\n", msg);

  bool res = sendMetrics(monReq, errBuf, ERR_BUF_LEN);

  deleteHandle(monReq);
  deleteHandle(monRes);
  deleteHandle(invSvc);
  deleteHandle(monSvc);
  deleteHandle(resGroup);

  if (!res) {
    fail(errBuf);
  }
}

int main(void) {
  printf("\n example_load_libtransit ...\n");
  void *libtransit_handle = example_load_libtransit();

  printf("\n example_libtransit_control ...\n");
  example_libtransit_control();

  startNats(NULL, 0);
  printf("\n example_send_inventory ...\n");
  example_send_inventory();
  printf("\n example_send_metrics ...\n");
  example_send_metrics();
  stopNats(NULL, 0);

  // We can't call this until we're actually done with the dynamically loaded
  // code, as when we do so the reference count will drop to zero, no other
  // loaded library will use symbols in it, and the dynamically loaded library
  // will be unloaded.
  dlclose(libtransit_handle);

  return 0;
}
