package nats

import (
	"container/ring"
	"sync"
)

// RBuf collects messages in ring buffer
type RBuf struct {
	mu   sync.Mutex
	once sync.Once
	ring *ring.Ring
	Size int
}

// Records returns collected items
func (buf *RBuf) Records() []BufItem {
	buf.mu.Lock()
	defer buf.mu.Unlock()
	buf.once.Do(func() {
		buf.ring = ring.New(buf.Size)
	})

	rec := []BufItem{}
	buf.ring.Do(func(p interface{}) {
		if p != nil {
			rec = append(rec, p.(BufItem))
		}
	})
	return rec
}

// Put collects item
func (buf *RBuf) Put(subj string, data []byte, headers []string) (int, error) {
	buf.mu.Lock()
	defer buf.mu.Unlock()
	buf.once.Do(func() {
		buf.ring = ring.New(buf.Size)
	})

	buf.ring.Value = BufItem{data, headers, subj}
	buf.ring = buf.ring.Next()

	return len(data), nil
}

// BufItem wraps payloads
type BufItem struct {
	data []byte
	hdrs []string
	subj string
}
