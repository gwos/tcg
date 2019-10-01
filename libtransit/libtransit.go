package main

// #define ERROR_LEN 250 /* for strncpy error message */
// #include <string.h> /* for strncpy error message */
import "C"
import (
	"encoding/json"
	"github.com/gwos/tng/transit"
	"log"
)

func main() {
}

//export TestMonitoredResource
func TestMonitoredResource(str *C.char, error *C.char) *C.char {
	resource := transit.MonitoredResource{}
	if err := json.Unmarshal([]byte(C.GoString(str)), &resource); err != nil {
		C.strncpy((*C.char)(error), C.CString(err.Error()), C.ERROR_LEN)
		return nil
	}

	resource.Labels = map[string]string{"key1": "value1", "key02": "value02"}
	resource.Status = transit.SERVICE_PENDING
	buf, _ := json.Marshal(&resource)

	log.Printf("#TestMonitoredResource: %v, %s", resource, buf)

	/* https://github.com/golang/go/wiki/cgo#go-strings-and-c-strings */
	return C.CString(string(buf))
}
