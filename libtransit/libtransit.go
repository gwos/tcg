package main

//#define ERROR_LEN 250 /* buffer for error message */
import "C"
import (
	"github.com/gwos/tng/services"
	"log"
	"unsafe"
)

func init() {
	transitService := services.GetTransitService()
	if transitService.AgentConfig.StartController {
		if err := transitService.StartController(); err != nil {
			log.Println(err)
		}
	}
	if transitService.AgentConfig.StartNATS {
		if err := transitService.StartNATS(); err != nil {
			log.Println(err)
		}
	}
	if transitService.AgentConfig.StartTransport {
		if err := transitService.StartTransport(); err != nil {
			log.Println(err)
		}
	}
	log.Println("libtransit:", transitService.Status())
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
	if err := services.GetTransitService().
		SendResourceWithMetrics([]byte(C.GoString(resourcesWithMetricsRequestJSON))); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

//export SynchronizeInventory
func SynchronizeInventory(sendInventoryRequestJSON, errorBuf *C.char) bool {
	if err := services.GetTransitService().
		SynchronizeInventory([]byte(C.GoString(sendInventoryRequestJSON))); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

//export StartController
func StartController(errorBuf *C.char) bool {
	if err := services.GetTransitService().StartController(); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

//export StopController
func StopController(errorBuf *C.char) bool {
	if err := services.GetTransitService().StopController(); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

//export StartNATS
func StartNATS(errorBuf *C.char) bool {
	if err := services.GetTransitService().StartNATS(); err != nil {
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
	if err := services.GetTransitService().StartTransport(); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

//export StopTransport
func StopTransport(errorBuf *C.char) bool {
	if err := services.GetTransitService().StopTransport(); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}
