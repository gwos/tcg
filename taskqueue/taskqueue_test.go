package taskqueue

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWithAlarm(t *testing.T) {
	mu := sync.Mutex{}
	res := map[Subject]int{
		"task1": 0,
		"task2": 0,
	}
	hAlarm := func(task *Task) error {
		mu.Lock()
		res[task.Subject] += 1
		mu.Unlock()
		return nil
	}
	hTask := func(task *Task) error {
		time.Sleep(time.Millisecond * 2)
		return nil
	}
	handlers := map[Subject]Handler{
		"task1": hTask,
		"task2": hTask,
	}
	q := NewTaskQueue(
		WithHandlers(handlers),
		WithAlarm(time.Millisecond, hAlarm),
	)

	var err error
	_, err = q.PushAsync("task1")
	assert.NoError(t, err)
	err = q.PushSync("task2")
	assert.NoError(t, err)

	/* it's a rare condition when the Alarm handler still running at the moment,
	caused by extreme timings */
	time.Sleep(time.Millisecond)
	mu.Lock()
	defer mu.Unlock()

	assert.Equal(
		t,
		map[Subject]int{
			"task1": 1,
			"task2": 1,
		},
		res,
	)
}

func TestWithCapacity(t *testing.T) {
	hTask := func(task *Task) error {
		time.Sleep(time.Millisecond)
		return nil
	}
	handlers := map[Subject]Handler{
		"task1": hTask,
		"task2": hTask,
	}
	q := NewTaskQueue(WithCapacity(2), WithHandlers(handlers))

	var err error
	_, err = q.PushAsync("task1")
	assert.NoError(t, err)
	_, err = q.PushAsync("task2")
	assert.NoError(t, err)
	_, err = q.PushAsync("task1")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrTaskQueue))
	assert.True(t, errors.Is(err, ErrTaskQueueCapacity))
}

func TestWithHandlers(t *testing.T) {
	type subj int
	const (
		task0 subj = iota
		task1
		task2
	)

	res := map[Subject]int{}
	hTask := func(task *Task) error {
		res[task.Subject] += 1
		return nil
	}
	handlers := map[Subject]Handler{
		task1: hTask,
		task2: hTask,
	}
	q := NewTaskQueue(WithHandlers(handlers))

	var (
		err  error
		task *Task
	)
	task, err = q.PushAsync(task1)
	assert.NoError(t, err)
	assert.Equal(t, uint8(1), task.Idx)
	assert.Equal(t, Subject(task1), task.Subject)
	assert.Equal(t, []any(nil), task.Args)
	task, err = q.PushAsync(task2, "data_arg", 1, true, []byte("data arg"))
	assert.NoError(t, err)
	assert.Equal(t, uint8(2), task.Idx)
	assert.Equal(t, Subject(task2), task.Subject)
	assert.Equal(t, []any{"data_arg", 1, true, []byte("data arg")}, task.Args)

	/* use sync to complete queue and check totals */
	err = q.PushSync(task2)
	assert.NoError(t, err)
	assert.Equal(
		t,
		map[Subject]int{
			task1: 1,
			task2: 2,
		},
		res,
	)
	/* test for an error */
	err = q.PushSync(task0)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrTaskQueue))
	assert.True(t, errors.Is(err, ErrTaskQueueUndefined))
}
