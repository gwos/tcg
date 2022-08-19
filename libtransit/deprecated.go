package main

/*
#include <stdbool.h>

typedef bool (*demandConfigHandler) ();

static bool invokeDemandConfigHandler(demandConfigHandler fn) {
	return fn();
}
*/
import "C"
import (
	"context"
	"encoding/json"

	"github.com/gwos/tcg/services"
)

// Deprecated: Use GetAgentId, GetAppName, GetAppType instead
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

// Deprecated: Use RegisterConfigHandler instead
// RegisterDemandConfigHandler is a C API for setting callback
//
//export RegisterDemandConfigHandler
func RegisterDemandConfigHandler(fn C.demandConfigHandler) {
	services.GetTransitService().RegisterConfigHandler(func([]byte) {
		C.invokeDemandConfigHandler(fn)
	})
}

// Deprecated: Use RemoveConfigHandler instead
// RemoveDemandConfigHandler is a C API for removing callback
//
//export RemoveDemandConfigHandler
func RemoveDemandConfigHandler() {
	services.GetTransitService().RemoveConfigHandler()
}

// Deprecated: Use Send instead
// ClearInDowntime is a C API for services.GetTransitService().ClearInDowntime
//
//export ClearInDowntime
func ClearInDowntime(payloadJSON, errBuf *C.char, errBufLen C.size_t) C.bool {
	if err := services.GetTransitService().
		ClearInDowntime(context.Background(), []byte(C.GoString(payloadJSON))); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// Deprecated: Use Send instead
// SetInDowntime is a C API for services.GetTransitService().SetInDowntime
//
//export SetInDowntime
func SetInDowntime(payloadJSON, errBuf *C.char, errBufLen C.size_t) C.bool {
	if err := services.GetTransitService().
		SetInDowntime(context.Background(), []byte(C.GoString(payloadJSON))); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// Deprecated: Use Send instead
// SendEvents is a C API for services.GetTransitService().SendEvents
//
//export SendEvents
func SendEvents(payloadJSON, errBuf *C.char, errBufLen C.size_t) C.bool {
	if err := services.GetTransitService().
		SendEvents(context.Background(), []byte(C.GoString(payloadJSON))); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// Deprecated: Use Send instead
// SendEventsAck is a C API for services.GetTransitService().SendEventsAck
//
//export SendEventsAck
func SendEventsAck(payloadJSON, errBuf *C.char, errBufLen C.size_t) C.bool {
	if err := services.GetTransitService().
		SendEventsAck(context.Background(), []byte(C.GoString(payloadJSON))); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// Deprecated: Use Send instead
// SendEventsUnack is a C API for services.GetTransitService().SendEventsUnack
//
//export SendEventsUnack
func SendEventsUnack(payloadJSON, errBuf *C.char, errBufLen C.size_t) C.bool {
	if err := services.GetTransitService().
		SendEventsUnack(context.Background(), []byte(C.GoString(payloadJSON))); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// Deprecated: Use Send instead
// SendResourcesWithMetrics is a C API for services.GetTransitService().SendResourceWithMetrics
//
//export SendResourcesWithMetrics
func SendResourcesWithMetrics(payloadJSON, errBuf *C.char, errBufLen C.size_t) C.bool {
	if err := services.GetTransitService().
		SendResourceWithMetrics(context.Background(), []byte(C.GoString(payloadJSON))); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// Deprecated: Use Send instead
// SynchronizeInventory is a C API for services.GetTransitService().SynchronizeInventory
//
//export SynchronizeInventory
func SynchronizeInventory(payloadJSON, errBuf *C.char, errBufLen C.size_t) C.bool {
	if err := services.GetTransitService().
		SynchronizeInventory(context.Background(), []byte(C.GoString(payloadJSON))); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}
