package main

//#define ERROR_LEN 250 /* buffer for error message */
import "C"
import (
	"log"
	"unsafe"

	"github.com/gwos/tng/controller"
	"github.com/gwos/tng/services"
)

func init() {
	var err error
	service := services.GetTransitService()

	if service.AgentConfig.StartController {
		err = controller.StartServer(service.AgentConfig.SSL, service.AgentConfig.Port)
		if err != nil {
			log.Println(err)
		}
	}

	if service.AgentConfig.StartNATS {
		err = service.StartNATS()
		if err != nil {
			log.Println(err)
		}
	}

	if service.AgentConfig.StartTransport {
		err = service.StartTransport()
		if err != nil {
			log.Println(err)
		}
	}
}

func main() {
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func putError(errorBuf *C.char, err error) {
	buf := (*[int(C.ERROR_LEN)]byte)(unsafe.Pointer(errorBuf))
	buf[min(copy(buf[:], err.Error()), C.ERROR_LEN-1)] = 0
}

//export SendResourcesWithMetrics
func SendResourcesWithMetrics(resourcesWithMetricsRequestJSON, errorBuf *C.char) bool {
	err := services.GetTransitService().
		SendResourceWithMetrics([]byte(C.GoString(resourcesWithMetricsRequestJSON)))
	if err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

//export SynchronizeInventory
func SynchronizeInventory(sendInventoryRequestJSON, errorBuf *C.char) bool {
	err := services.GetTransitService().
		SynchronizeInventory([]byte(C.GoString(sendInventoryRequestJSON)))
	if err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

//export StartNATS
func StartNATS(errorBuf *C.char) bool {
	err := services.GetTransitService().StartNATS()
	if err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

//export StopNATS
func StopNATS() {
	services.GetTransitService().StopNATS()
}

//export StartTransport
func StartTransport(errorBuf *C.char) bool {
	err := services.GetTransitService().StartTransport()
	if err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

//export StopTransport
func StopTransport(errorBuf *C.char) bool {
	err := services.GetTransitService().StopTransport()
	if err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

//export StartController
func StartController(errorBuf *C.char) bool {
	service := services.GetTransitService()
	err := controller.StartServer(service.AgentConfig.SSL, service.AgentConfig.Port)
	if err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

//export StopController
func StopController(errorBuf *C.char) bool {
	err := controller.StopServer()
	if err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

//TODO:
func ListInventory() {
}

////export ListMetrics
// func ListMetrics(errorBuf *C.char) *C.char {
// 	monitorDescriptor, err := transitService.ListMetrics()
// 	if err != nil {
// 		putError(errorBuf, err)
// 		return nil
// 	}
//
// 	bytes, err := json.Marshal(monitorDescriptor)
// 	if err != nil {
// 		putError(errorBuf, err)
// 		return nil
// 	}
//
// 	return C.CString(string(bytes))
// }
