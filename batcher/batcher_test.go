package batcher

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type intBuilder struct{}

func (intBuilder) Build(buf []Sized[int], _ int) [][]byte {
	payloads := make([][]byte, 0, len(buf))
	for _, it := range buf {
		payloads = append(payloads, []byte(strconv.Itoa(it.Value)))
	}
	return payloads
}

func TestBatcherOrder(t *testing.T) {
	var mu sync.Mutex
	handled := make([]int, 0)
	handler := func(_ context.Context, p []byte) error {
		v, _ := strconv.Atoi(string(p))
		// slow handler lets concurrent batches overlap
		time.Sleep(10 * time.Microsecond)
		mu.Lock()
		handled = append(handled, v)
		mu.Unlock()
		return nil
	}
	// every Add exceeds maxBytes and batches, the ticker batches concurrently
	bt := NewBatcher[int](intBuilder{}, handler, time.Millisecond, 1)
	expected := make([]int, 0)
	for i := 0; i < 1000; i++ {
		bt.Add(i, 2)
		expected = append(expected, i)
	}
	bt.Batch()
	bt.Exit()

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, expected, handled)
}
