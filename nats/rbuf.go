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

// Records returns collected writes
func (buf *RBuf) Records() []BufMsg {
	buf.mu.Lock()
	defer buf.mu.Unlock()
	buf.once.Do(func() {
		buf.ring = ring.New(buf.Size)
	})

	rec := []BufMsg{}
	buf.ring.Do(func(p interface{}) {
		if p != nil {
			rec = append(rec, p.(BufMsg))
		}
	})
	return rec
}

func (buf *RBuf) WriteMsg(subj string, msg []byte) (int, error) {
	buf.mu.Lock()
	defer buf.mu.Unlock()
	buf.once.Do(func() {
		buf.ring = ring.New(buf.Size)
	})

	buf.ring.Value = BufMsg{msg, subj}
	buf.ring = buf.ring.Next()

	return len(msg), nil
}

// BufMsg wraps payloads
type BufMsg struct {
	msg  []byte
	subj string
}
