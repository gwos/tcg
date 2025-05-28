package main

/*
#include <stdbool.h>

typedef const char cchar_t;
*/
import "C"
import (
	"context"
	"encoding/json"

	"github.com/gwos/tcg/services"
)

// C API for JSON serialized data

// GetAgentIdentity is a C API for getting AgentIdentity
//
//export GetAgentIdentity
func GetAgentIdentity(buf *C.char, bufLen C.size_t, errBuf *C.char, errBufLen C.size_t) C.bool {
	res, err := json.Marshal(services.GetTransitService().Connector.AgentIdentity)
	if err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	cStrLen := len(res) + 1
	if cStrLen > int(bufLen) {
		bufStr(errBuf, errBufLen, msgfBufTooSmall(cStrLen))
		return false
	}
	bufStr(buf, bufLen, string(res))
	return true
}

// ClearInDowntime is a C API for services.GetTransitService().ClearInDowntime
//
//export ClearInDowntime
func ClearInDowntime(payloadJSON *C.cchar_t, errBuf *C.char, errBufLen C.size_t) C.bool {
	if err := services.GetTransitService().
		ClearInDowntime(context.Background(), []byte(C.GoString(payloadJSON))); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// SetInDowntime is a C API for services.GetTransitService().SetInDowntime
//
//export SetInDowntime
func SetInDowntime(payloadJSON *C.cchar_t, errBuf *C.char, errBufLen C.size_t) C.bool {
	if err := services.GetTransitService().
		SetInDowntime(context.Background(), []byte(C.GoString(payloadJSON))); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// SendEvents is a C API for services.GetTransitService().SendEvents
//
//export SendEvents
func SendEvents(payloadJSON *C.cchar_t, errBuf *C.char, errBufLen C.size_t) C.bool {
	if err := services.GetTransitService().
		SendEvents(context.Background(), []byte(C.GoString(payloadJSON))); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// SendEventsAck is a C API for services.GetTransitService().SendEventsAck
//
//export SendEventsAck
func SendEventsAck(payloadJSON *C.cchar_t, errBuf *C.char, errBufLen C.size_t) C.bool {
	if err := services.GetTransitService().
		SendEventsAck(context.Background(), []byte(C.GoString(payloadJSON))); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// SendEventsUnack is a C API for services.GetTransitService().SendEventsUnack
//
//export SendEventsUnack
func SendEventsUnack(payloadJSON *C.cchar_t, errBuf *C.char, errBufLen C.size_t) C.bool {
	if err := services.GetTransitService().
		SendEventsUnack(context.Background(), []byte(C.GoString(payloadJSON))); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// SendResourcesWithMetrics is a C API for services.GetTransitService().SendResourceWithMetrics
//
//export SendResourcesWithMetrics
func SendResourcesWithMetrics(payloadJSON *C.cchar_t, errBuf *C.char, errBufLen C.size_t) C.bool {
	if err := services.GetTransitService().
		SendResourceWithMetrics(context.Background(), []byte(C.GoString(payloadJSON))); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// SynchronizeInventory is a C API for services.GetTransitService().SynchronizeInventory
//
//export SynchronizeInventory
func SynchronizeInventory(payloadJSON *C.cchar_t, errBuf *C.char, errBufLen C.size_t) C.bool {
	if err := services.GetTransitService().
		SynchronizeInventory(context.Background(), []byte(C.GoString(payloadJSON))); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// SynchronizeInventoryExt is a C API for services.GetTransitService().SynchronizeInventoryExt
//
//export SynchronizeInventoryExt
func SynchronizeInventoryExt(payloadJSON *C.cchar_t, errBuf *C.char, errBufLen C.size_t) C.bool {
	if err := services.GetTransitService().
		SynchronizeInventoryExt(context.Background(), []byte(C.GoString(payloadJSON))); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}
