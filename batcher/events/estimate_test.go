package events

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gwos/tcg/sdk/transit"
	"github.com/stretchr/testify/assert"
)

func TestEstimateSize(t *testing.T) {
	rd := transit.NewTimestamp()
	rd.Time = time.Now()
	q := &transit.GroundworkEventsRequest{}
	for range 5 {
		q.Events = append(q.Events, transit.GroundworkEvent{
			AppType: "NAGIOS", Host: "host-1", Service: "local_load", Device: "127.0.0.1",
			MonitorStatus: "WARNING", Severity: "WARNING", TypeRule: "NAGIOS",
			TextMessage: "WARNING - load average: 5.12, 4.34, 3.56",
			ErrorType:   "SERVICE NOTIFICATION", ApplicationName: "NAGIOS", LoggerName: "NAGIOS",
			ConsolidationName: "NAGIOSEVENT", MonitorServer: "localhost", ReportDate: rd,
		})
	}
	bb, err := json.Marshal(q)
	assert.NoError(t, err)
	est := EstimateSize(q)
	ratio := float64(est) / float64(len(bb))
	t.Logf("estimate/actual %d/%d = %.2f", est, len(bb), ratio)
	assert.InDelta(t, 1.0, ratio, 0.25)
}
