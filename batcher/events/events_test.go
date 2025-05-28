package events

import (
	"encoding/json"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/gwos/tcg/sdk/transit"
	"github.com/stretchr/testify/assert"
)

func TestBuild(t *testing.T) {
	mbb := new(EventsBatchBuilder)
	buf := [][]byte{
		[]byte(`{"events":[
			{"host":"host1","device":"127.0.0.1","service":"http_alive","monitorStatus":"UP","severity":"SERIOUS","textMessage":"This is a serious Nagios Message on Device 127.0.0.1 - 0","lastInsertDate":"1370195732000","reportDate":"1579703726166","appType":"NAGIOS","monitorServer":"localhost"},
			{"host":"host2","device":"127.0.0.2","service":"test","monitorStatus":"UP","severity":"SERIOUS","textMessage":"This is a serious Nagios Message on Device 127.0.0.1 - 0","lastInsertDate":"1370195732000","reportDate":"1579703726166","appType":"NAGIOS","monitorServer":"localhost"},
			{"host":"host4","device":"127.0.0.3","service":"http_alive","monitorStatus":"UP","severity":"SERIOUS","textMessage":"This is a serious Nagios Message on Device 127.0.0.1 - 0","lastInsertDate":"1370195732000","reportDate":"1579703726166","appType":"NAGIOS","monitorServer":"localhost"},
			{"host":"host5","device":"127.0.0.4","service":"test","monitorStatus":"UP","severity":"SERIOUS","textMessage":"This is a serious Nagios Message on Device 127.0.0.1 - 0","lastInsertDate":"1370195732000","reportDate":"1579703726166","appType":"NAGIOS","monitorServer":"localhost"}
		]}`),
		[]byte(`{"events":[{"host":"host11","device":"new_device","service":"http_alive","monitorStatus":"UP","severity":"SERIOUS","textMessage":"This is a serious Nagios Message on Device 127.0.0.1 - 0","lastInsertDate":"1370195732000","reportDate":"1579703726166","appType":"NAGIOS","monitorServer":"localhost"}]}`),
		[]byte(`{"events":[{"host":"host12","device":"127.0.0.1","service":"test","monitorStatus":"UP","severity":"SERIOUS","textMessage":"This is a serious Nagios Message on Device 127.0.0.1 - 0","lastInsertDate":"1370195732000","reportDate":"1579703726166","appType":"NAGIOS","monitorServer":"localhost"}]}`),
	}

	printMemStats()
	mbb.Build(&buf, 1024)
	printMemStats()

	qq := make([]transit.GroundworkEventsRequest, 0)
	var q transit.GroundworkEventsRequest
	for _, p := range buf {
		q = transit.GroundworkEventsRequest{}
		assert.NoError(t, json.Unmarshal(p, &q))
		qq = append(qq, q)
	}
	assert.Equal(t, 3, len(qq))
	assert.Equal(t, 3, len(qq[0].Events))
	assert.Equal(t, "host2", qq[0].Events[1].Host)
	assert.Equal(t, 1, len(qq[1].Events))
	assert.Equal(t, 2, len(qq[2].Events))
	assert.Equal(t, "host11", qq[2].Events[0].Host)
}

// inspired by expvar.Handler() implementation
func memstats() any {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return stats
}
func printMemStats() {
	println("\n~", time.Now().Format(time.DateTime), "MEM_STATS", fmt.Sprintf("%+v", memstats()))
}
