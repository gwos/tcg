package events

import (
	"github.com/gwos/tcg/sdk/transit"
)

// Approximate JSON overhead of fixed parts, in bytes.
// Keys, punctuation and formatted numbers are counted here,
// variable-length strings are added separately.
const (
	sizeRequest    = 12 // events key and brackets
	sizeEvent      = 50 // appType, host, monitorStatus, reportDate keys
	sizeTimestamp  = 35 // key and quoted millis
	sizeFieldQuote = 20 // key and quotes for an optional string field
)

// EstimateSize returns the approximate size of the JSON-serialized request.
// It is cheap enough to be called on the caller's thread, no serialization is done.
func EstimateSize(q *transit.GroundworkEventsRequest) int {
	if q == nil {
		return 0
	}
	n := sizeRequest
	for i := range q.Events {
		e := &q.Events[i]
		n += sizeEvent + len(e.AppType) + len(e.Host) + len(e.MonitorStatus) +
			optStr(e.Device) + optStr(e.Service) + optStr(e.OperationStatus) +
			optStr(e.Severity) + optStr(e.ApplicationSeverity) +
			optStr(e.Component) + optStr(e.SubComponent) + optStr(e.Priority) +
			optStr(e.TypeRule) + optStr(e.TextMessage) +
			optStr(e.ApplicationName) + optStr(e.ConsolidationName) +
			optStr(e.ErrorType) + optStr(e.LoggerName) + optStr(e.LogType) +
			optStr(e.MonitorServer)
		if e.ReportDate != nil {
			n += sizeTimestamp
		}
		if e.LastInsertDate != nil {
			n += sizeTimestamp
		}
	}
	return n
}

func optStr(s string) int {
	if s == "" {
		return 0
	}
	return sizeFieldQuote + len(s)
}
