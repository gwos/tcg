package batcher

import (
	"context"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog/log"
)

// BatchBuilder defines builder interface
type BatchBuilder interface {
	// Add adds single payload to batch
	Add(p []byte)
	// Build builds the batch payloads
	// it's possible that not all single payloads can be combined into one
	Build() [][]byte
}

// BatchBuilderConstructor defines constructor
type BatchBuilderConstructor func() BatchBuilder

// BatchHandler defines handler
type BatchHandler func(context.Context, []byte) error

// Batcher implements buffered batcher
type Batcher struct {
	mu sync.Mutex

	cache      *cache.Cache
	cacheSize  int
	maxBytes   int
	ticker     *time.Ticker
	tickerExit chan bool

	batchHandler BatchHandler
	newBBuilder  BatchBuilderConstructor
}

// NewBatcher returns new instance
func NewBatcher(
	newBB BatchBuilderConstructor,
	bh BatchHandler,
	d time.Duration,
	maxBytes int) *Batcher {
	if d == 0 {
		d = math.MaxInt64
	}
	bt := Batcher{
		cache:      cache.New(-1, -1),
		cacheSize:  0,
		maxBytes:   maxBytes,
		ticker:     time.NewTicker(d),
		tickerExit: make(chan bool, 1),

		batchHandler: bh,
		newBBuilder:  newBB,
	}
	bt.cache.SetDefault("nextCK", uint64(0))

	/* handle ticker */
	go func() {
		for {
			select {
			case <-bt.ticker.C:
				bt.Batch()
			case <-bt.tickerExit:
				bt.Batch()
				return
			}
		}
	}()

	return &bt
}

// Add adds single payload to batch buffer
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
		bt.Batch()
	}
	return err
}

// Batch processes buffered payloads
func (bt *Batcher) Batch() {
	bt.mu.Lock()

	bld := bt.newBBuilder()
	for ck, c := range bt.cache.Items() {
		if ck == "nextCK" {
			continue
		}
		p := c.Object.([]byte)
		bld.Add(p)
		bt.cacheSize -= len(p)
		bt.cache.Delete(ck)
	}

	bt.mu.Unlock()

	payloads := bld.Build()
	if len(payloads) > 0 {
		for _, p := range payloads {
			if len(p) > 0 {
				bt.batchHandler(context.Background(), p)
			}
		}
	}
}

// Exit stops the internal ticker
func (bt *Batcher) Exit() {
	bt.tickerExit <- true
}

// Reset applyes configuration
func (bt *Batcher) Reset(d time.Duration, maxBytes int) {
	bt.Batch()
	bt.maxBytes = maxBytes
	if d == 0 {
		d = math.MaxInt64
	}
	bt.ticker.Reset(d)
}
