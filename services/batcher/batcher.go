package batcher

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"time"

	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/transit"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog/log"
)

// Batcher implements buffered batcher
type Batcher struct {
	cache      *cache.Cache
	cacheSize  int
	maxBytes   int
	ticker     *time.Ticker
	tickerExit chan bool

	rh sendRequestHandler
}

// NewBatcher returns new instance
func NewBatcher(fn sendRequestHandler, d time.Duration, maxBytes int) *Batcher {
	if d == 0 {
		d = math.MaxInt64
	}
	bt := Batcher{
		cache:      cache.New(-1, -1),
		cacheSize:  0,
		maxBytes:   maxBytes,
		ticker:     time.NewTicker(d),
		tickerExit: make(chan bool, 1),

		rh: fn,
	}
	bt.cache.SetDefault("nextCK", uint64(0))

	/* handle ticker */
	go func() {
		for {
			select {
			case <-bt.ticker.C:
				bt.batchBufferedMetrics()
			case <-bt.tickerExit:
				bt.batchBufferedMetrics()
				return
			}
		}
	}()

	return &bt
}

// Add adds single transit.DynamicResourcesWithServicesRequest to batch
func (bt *Batcher) Add(p []byte) error {
	nextCK, err := bt.cache.IncrementUint64("nextCK", 1)
	if err != nil {
		return err
	}
	if nextCK == math.MaxUint64 {
		nextCK = 0
		bt.cache.SetDefault("nextCK", 0)
	}
	bt.cache.SetDefault(strconv.FormatUint(nextCK, 10), p)
	bt.cacheSize += len(p)
	if bt.cacheSize > bt.maxBytes {
		log.Debug().Msgf("batch buffer size %dKB exceeded the soft limit %dKB",
			bt.cacheSize/1024, bt.maxBytes/1024)
		return bt.batchBufferedMetrics()
	}
	return err
}

// Exit stops the internal ticker
func (bt *Batcher) Exit() {
	bt.tickerExit <- true
}

// Reset applyes configuration
func (bt *Batcher) Reset(d time.Duration, maxBytes int) {
	bt.batchBufferedMetrics()
	bt.maxBytes = maxBytes
	if d == 0 {
		d = math.MaxInt64
	}
	bt.ticker.Reset(d)
}

func (bt *Batcher) batchBufferedMetrics() error {
	bld := NewBuilder()
	for ck, c := range bt.cache.Items() {
		if ck == "nextCK" {
			continue
		}
		p := c.Object.([]byte)
		bld.Add(p)
		bt.cacheSize -= len(p)
		bt.cache.Delete(ck)
	}
	payload, err := bld.Build()
	if err == nil && len(payload) > 0 {
		return bt.rh(context.Background(), payload)
	}
	return err
}

type sendRequestHandler func(context.Context, []byte) error

// BatchBuilder implements builder
type BatchBuilder struct {
	tContexts []transit.TracerContext
	groupsMap map[string]transit.ResourceGroup
	resMap    map[string]transit.DynamicMonitoredResource
	svcMap    map[string]svcMapItem
}

// NewBuilder returns new instance
func NewBuilder() *BatchBuilder {
	return &BatchBuilder{
		tContexts: make([]transit.TracerContext, 0),
		groupsMap: make(map[string]transit.ResourceGroup),
		resMap:    make(map[string]transit.DynamicMonitoredResource),
		svcMap:    make(map[string]svcMapItem),
	}
}

// Add adds single transit.DynamicResourcesWithServicesRequest to batch
func (bld *BatchBuilder) Add(p []byte) {
	r := transit.DynamicResourcesWithServicesRequest{}
	if err := json.Unmarshal(p, &r); err != nil {
		log.Err(err).
			RawJSON("payload", p).
			Msg("could not unmarshal metrics payload for batch")
	} else {
		bld.tContexts = append(bld.tContexts, *r.Context)
		for _, g := range r.Groups {
			bld.groupsMap[string(g.Type)+":"+g.GroupName] = g
		}
		for _, res := range r.Resources {
			resK := string(res.Type) + ":" + res.Name
			for _, svc := range res.Services {
				applyTime(&res, &svc, r.Context.TimeStamp) // ensure time fields
				svcK := string(res.Type) + ":" + res.Name + "::" + string(svc.Type) + ":" + svc.Name
				bld.svcMap[svcK] = svcMapItem{
					resK: resK,
					svcK: svcK,
					svc:  svc,
				}
			}
			applyTime(&res, &transit.DynamicMonitoredService{}, r.Context.TimeStamp) // ensure resource time fields in case of empty services
			res.Services = []transit.DynamicMonitoredService{}
			bld.resMap[resK] = res
		}
	}
}

// Build builds the batch payload if not empty
func (bld *BatchBuilder) Build() ([]byte, error) {
	r := transit.DynamicResourcesWithServicesRequest{}
	if len(bld.tContexts) > 0 {
		r.Context = &bld.tContexts[0]
	}
	for _, g := range bld.groupsMap {
		r.Groups = append(r.Groups, g)
	}
	for _, svcItem := range bld.svcMap {
		res := bld.resMap[svcItem.resK]
		res.Services = append(res.Services, svcItem.svc)
		bld.resMap[svcItem.resK] = res
	}
	for _, res := range bld.resMap {
		r.Resources = append(r.Resources, res)
	}

	if len(r.Resources) > 0 {
		log.Debug().Msgf("batched %d resources with %d services in %d groups",
			len(r.Resources), len(bld.svcMap), len(bld.groupsMap))
		return json.Marshal(r)
	}
	return nil, nil
}

type svcMapItem struct {
	resK string
	svcK string
	svc  transit.DynamicMonitoredService
}

func applyTime(res *transit.DynamicMonitoredResource,
	svc *transit.DynamicMonitoredService,
	ts milliseconds.MillisecondTimestamp) {

	if ts.IsZero() {
		ts = milliseconds.MillisecondTimestamp{Time: time.Now()}
	}
	switch {
	case res.LastCheckTime.IsZero() && !svc.LastCheckTime.IsZero():
		res.LastCheckTime = svc.LastCheckTime
	case !res.LastCheckTime.IsZero() && svc.LastCheckTime.IsZero():
		svc.LastCheckTime = res.LastCheckTime
	case res.LastCheckTime.IsZero() && svc.LastCheckTime.IsZero():
		res.LastCheckTime = ts
		svc.LastCheckTime = ts
	}
	switch {
	case res.NextCheckTime.IsZero() && !svc.NextCheckTime.IsZero():
		res.NextCheckTime = svc.NextCheckTime
	case !res.NextCheckTime.IsZero() && svc.NextCheckTime.IsZero():
		svc.NextCheckTime = res.NextCheckTime
	case res.NextCheckTime.IsZero() && svc.NextCheckTime.IsZero():
		res.NextCheckTime = ts
		svc.NextCheckTime = ts
	}
}
