package taskqueue

import (
	"container/ring"
	"fmt"
	"math"
	"sync"
	"time"
)

const defaultCapacity = 8

var (
	ErrTaskQueue          = fmt.Errorf("task queue error")
	ErrTaskQueueCapacity  = fmt.Errorf("%w: capacity is exhausted", ErrTaskQueue)
	ErrTaskQueueUndefined = fmt.Errorf("%w: undefined", ErrTaskQueue)
)

// Task defines queued task
type Task struct {
	done    chan error
	Args    []interface{}
	Idx     uint8
	Subject Subject
}

// Done returns channel for result
func (task Task) Done() chan error {
	return task.done
}

// Handler defines task handler
type Handler func(*Task) error

// Subject defines task subject
// used as key in map and requires comparable type
type Subject interface{}

// TaskQueue defines task queue
type TaskQueue struct {
	mu           sync.Mutex
	alarm        time.Duration
	alarmHandler Handler
	capacity     uint8
	debugger     func([]Task)
	handlers     map[Subject]Handler
	idx          uint8
	queue        chan *Task
	ring         *ring.Ring
}

// PushAsync adds task into queue and returns immediately
func (q *TaskQueue) PushAsync(subj Subject, args ...interface{}) (*Task, error) {
	if _, ok := q.handlers[subj]; !ok {
		return nil, fmt.Errorf("%w: %v", ErrTaskQueueUndefined, subj)
	}
	done := make(chan error, 1)
	task := &Task{done: done, Args: args, Idx: q.idx + 1, Subject: subj}
	/* put task into queue */
	q.mu.Lock()
	defer q.mu.Unlock()
	select {
	case q.queue <- task:
		q.idx = task.Idx
		if q.idx > math.MaxUint8-1 {
			q.idx = 0
		}
		/* put task into ring buffer for debug */
		q.ring.Value = *task
		q.ring = q.ring.Next()
		return task, nil
	default:
		if q.debugger != nil {
			lastTasks := []Task{}
			q.ring.Do(func(p interface{}) {
				if p != nil {
					lastTasks = append(lastTasks, p.(Task))
				}
			})
			q.debugger(lastTasks)
		}
		return nil, fmt.Errorf("%w: %v", ErrTaskQueueCapacity, q.capacity)
	}
}

// PushSync adds task into queue and returns after task processing
func (q *TaskQueue) PushSync(subj Subject, args ...interface{}) error {
	if task, err := q.PushAsync(subj, args...); err != nil {
		return err
	} else {
		return <-task.Done()
	}
}

func (q *TaskQueue) runQueue() {
	for task := range q.queue {
		var alarmTimer *time.Timer
		if q.alarm != 0 && q.alarmHandler != nil {
			/* create closure to prevent the race condition
			as loop var can be updated in time between timer trigger and handler call */
			func(task *Task) {
				alarmTimer = time.AfterFunc(q.alarm, func() {
					q.alarmHandler(task)
				})
			}(task)
		}
		handler := q.handlers[task.Subject]
		err := handler(task)
		if alarmTimer != nil {
			alarmTimer.Stop()
		}
		task.done <- err
		close(task.done)
	}
}

// TaskQueueOption defines task queue option
type TaskQueueOption func(*TaskQueue)

// NewTaskQueue creates task queue
func NewTaskQueue(opts ...TaskQueueOption) *TaskQueue {
	q := &TaskQueue{capacity: defaultCapacity}
	for _, optFn := range opts {
		optFn(q)
	}
	q.ring = ring.New(int(q.capacity))
	q.queue = make(chan *Task, q.capacity)
	go q.runQueue()
	return q
}

// WithAlarm defines handler invoked on timeout after task start
// useful for log the long executed task
func WithAlarm(d time.Duration, h Handler) TaskQueueOption {
	return func(q *TaskQueue) {
		q.alarm = d
		q.alarmHandler = h
	}
}

// WithCapacity defines capacity of task queue
func WithCapacity(c uint8) TaskQueueOption {
	return func(q *TaskQueue) {
		q.capacity = c
	}
}

// WithHandlers defines tasks
func WithHandlers(m map[Subject]Handler) TaskQueueOption {
	return func(q *TaskQueue) {
		q.handlers = m
	}
}

// WithDebugger defines debug
func WithDebugger(fn func([]Task)) TaskQueueOption {
	return func(q *TaskQueue) {
		q.debugger = fn
	}
}
