package events

import (
	"encoding/json"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/gwos/tcg/batcher"
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
	buf = mbb.Build(toSized(t, buf), 1024)
	printMemStats()

	qq := make([]transit.GroundworkEventsRequest, 0, len(buf))
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

func TestBuildOrder(t *testing.T) {
	ev := func(hosts ...string) *transit.GroundworkEventsRequest {
		q := &transit.GroundworkEventsRequest{}
		for _, h := range hosts {
			q.Events = append(q.Events, transit.GroundworkEvent{Host: h})
		}
		return q
	}
	buf := []batcher.Sized[*transit.GroundworkEventsRequest]{
		{Value: ev("before"), Size: 100},
		{Value: ev("big-1", "big-2", "big-3", "big-4"), Size: 3000},
		{Value: ev("after"), Size: 100},
	}
	hosts := make([]string, 0)
	for _, p := range new(EventsBatchBuilder).Build(buf, 1024) {
		q := transit.GroundworkEventsRequest{}
		assert.NoError(t, json.Unmarshal(p, &q))
		for _, e := range q.Events {
			hosts = append(hosts, e.Host)
		}
	}
	assert.Equal(t, []string{"before", "big-1", "big-2", "big-3", "big-4", "after"}, hosts)
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

// toSized converts serialized payloads into batcher input with exact sizes
func toSized(t testing.TB, buf [][]byte) []batcher.Sized[*transit.GroundworkEventsRequest] {
	items := make([]batcher.Sized[*transit.GroundworkEventsRequest], 0, len(buf))
	for _, p := range buf {
		q := new(transit.GroundworkEventsRequest)
		assert.NoError(t, json.Unmarshal(p, q))
		items = append(items, batcher.Sized[*transit.GroundworkEventsRequest]{Value: q, Size: len(p)})
	}
	return items
}
