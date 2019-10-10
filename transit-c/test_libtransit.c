#include <dlfcn.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "libtransit.h" /* ERROR_LEN */
#include "transit.h"
#include "transit_json.h"
#include "util.h"

void test_dlSendResourcesWithMetrics() {
  /*
  https://medium.com/@ben.mcclelland/an-adventure-into-cgo-calling-go-code-with-c-b20aa6637e75
  https://medium.com/learning-the-go-programming-language/calling-go-functions-from-other-languages-4c7d8bcc69bf
   */
  void *handle;
  char *error;

  handle = dlopen("./libtransit.so", RTLD_LAZY);
  if (!handle) {
    fprintf(stderr, "\nlibtransit error: %s\n", dlerror());
    exit(1);
  }

  bool (*sendResourcesWithMetrics)(char *, char *) =
      dlsym(handle, "SendResourcesWithMetrics");
  if ((error = dlerror()) != NULL) {
    fprintf(stderr, "\nlibtransit error: %s\n", error);
    exit(1);
  }

  /* TODO: should be serialized ResourceWithMetricsRequest */
  char *wrong_request_json =
      "{\"name\": \"the-unique-name-of-the-instance-01\", \"status\": "
      "\"HOST_UP\", \"type\": \"gce_instance\"}";

  char err[ERROR_LEN] = "";
  bool res = sendResourcesWithMetrics(wrong_request_json, (char *)&err);
  printf("\n_res:err: %d : %s", res, err);
  dlclose(handle);
}

int main(void) {
  test_dlSendResourcesWithMetrics();

  fprintf(stdout, "\nall tests passed");
  return 0;
}
