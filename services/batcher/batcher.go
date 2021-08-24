package batcher

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"strconv"
	"time"

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
	if payload != nil && err == nil {
		return bt.rh(context.Background(), payload)
	}
	return err
}

type sendRequestHandler func(context.Context, []byte) error

// BatchBuilder implements builder
type BatchBuilder struct {
	r         transit.DynamicResourcesWithServicesRequest
	groupsMap map[string]transit.ResourceGroup
	resMap    map[string]transit.DynamicMonitoredResource
	resSvcMap map[string][]transit.DynamicMonitoredService
	svcMap    map[string]svcMapItem
}

// NewBuilder returns new instance
func NewBuilder() *BatchBuilder {
	return &BatchBuilder{
		r:         transit.DynamicResourcesWithServicesRequest{},
		groupsMap: make(map[string]transit.ResourceGroup),
		resMap:    make(map[string]transit.DynamicMonitoredResource),
		resSvcMap: make(map[string][]transit.DynamicMonitoredService),
		svcMap:    make(map[string]svcMapItem),
	}
}

// Add adds single transit.DynamicResourcesWithServicesRequest to batch
func (bld *BatchBuilder) Add(p []byte) {
	var r2 transit.DynamicResourcesWithServicesRequest
	if err := json.Unmarshal(p, &r2); err != nil {
		log.Err(err).
			RawJSON("payload", p).
			Msg("could not unmarshal metrics payload for batch")
	} else {
		if bld.r.Context == nil {
			bld.r.Context = r2.Context
		}
		for _, g := range r2.Groups {
			bld.groupsMap[string(g.Type)+":"+g.GroupName] = g
		}
		for _, res := range r2.Resources {
			resK := string(res.Type) + ":" + res.Name
			for _, svc := range res.Services {
				svcK := string(res.Type) + ":" + res.Name + "::" + string(svc.Type) + ":" + svc.Name
				bld.svcMap[svcK] = svcMapItem{
					resK: resK,
					svcK: svcK,
					svc:  svc,
				}
			}
			res.Services = []transit.DynamicMonitoredService{}
			bld.resMap[resK] = res
		}
	}
}

// Build builds the batch payload if not empty
func (bld *BatchBuilder) Build() ([]byte, error) {
	for _, g := range bld.groupsMap {
		bld.r.Groups = append(bld.r.Groups, g)
	}
	for _, svcItem := range bld.svcMap {
		bld.resSvcMap[svcItem.resK] = append(bld.resSvcMap[svcItem.resK], svcItem.svc)
	}
	for _, res := range bld.resMap {
		resK := string(res.Type) + ":" + res.Name
		res.Services = bld.resSvcMap[resK]
		bld.r.Resources = append(bld.r.Resources, res)
	}

	if len(bld.r.Resources) > 0 {
		log.Debug().Msgf("batched %d resources with %d services in %d groups",
			len(bld.r.Resources), len(bld.svcMap), len(bld.groupsMap))
		p, err := json.Marshal(bld.r)

		if err == nil && bytes.Contains(p, []byte(`:"-6`)) {
			log.Warn().
				Interface("resMap", bld.resMap).
				Interface("batch.Resources", bld.r.Resources).
				RawJSON("payload", p).
				Msg("zero time in batch payload")
		}

		return p, err
	}
	return nil, nil
}

type svcMapItem struct {
	resK string
	svcK string
	svc  transit.DynamicMonitoredService
}
