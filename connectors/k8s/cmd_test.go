package k8s

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

// transportStub replaces the transport control seam and records the calls made through it.
type transportStub struct {
	running bool
	stops   int
	starts  int
	stopErr error
}

func (s *transportStub) install(t *testing.T) {
	t.Helper()
	origRunning, origStop, origStart := transportRunning, stopTransport, startTransport
	origFailures, origStopped, origMax := failures, stoppedByUs, maxFailures
	failures, stoppedByUs, maxFailures = 0, false, defaultFailuresToStop

	transportRunning = func() bool { return s.running }
	stopTransport = func() error {
		s.stops++
		if s.stopErr != nil {
			return s.stopErr
		}
		s.running = false
		return nil
	}
	startTransport = func() error {
		s.starts++
		s.running = true
		return nil
	}

	t.Cleanup(func() {
		transportRunning, stopTransport, startTransport = origRunning, origStop, origStart
		failures, stoppedByUs, maxFailures = origFailures, origStopped, origMax
	})
}

func authErr() error {
	return fmt.Errorf("%w: %w", ErrKAPI, k8serrors.NewUnauthorized("token expired"))
}

func otherErr() error {
	return fmt.Errorf("%w: %w", ErrKAPI, k8serrors.NewTimeoutError("slow", 1))
}

func TestMarkCollectAuthErrorStopsImmediately(t *testing.T) {
	stub := &transportStub{running: true}
	stub.install(t)

	markCollectFailed(authErr())
	assert.Equal(t, 1, stub.stops, "an expired token must turn the status red at once")
	assert.False(t, stub.running)
	assert.True(t, stoppedByUs)
}

// A transport brought back up behind our back — by the Start button or by a config
// push — must be marked stopped again while collection is still failing. Trusting a
// remembered "stopped by us" flag here left the connector Running forever.
func TestMarkCollectFailedReStopsExternallyRestartedTransport(t *testing.T) {
	stub := &transportStub{running: true}
	stub.install(t)

	markCollectFailed(authErr())
	assert.Equal(t, 1, stub.stops)
	assert.False(t, stub.running)

	/* the Start button restarts the transport while the token is still expired */
	stub.running = true

	markCollectFailed(authErr())
	assert.Equal(t, 2, stub.stops, "a still-failing connector must not stay Running")
	assert.False(t, stub.running)
}

func TestMarkCollectFailedStopsTransportOncePerOutage(t *testing.T) {
	stub := &transportStub{running: true}
	stub.install(t)

	for range 5 {
		markCollectFailed(authErr())
	}
	assert.Equal(t, 1, stub.stops, "retries in an already red state must not touch the transport")
}

func TestApplyRetryConfigUsesConfiguredRetries(t *testing.T) {
	stub := &transportStub{running: true}
	stub.install(t)

	applyRetryConfig(2) // UI "Retries" = 2

	markCollectFailed(otherErr())
	assert.Equal(t, 0, stub.stops, "one failure is below the configured threshold")

	markCollectFailed(otherErr())
	assert.Equal(t, 1, stub.stops, "the second failure reaches the configured threshold")
}

func TestApplyRetryConfigFallsBackToDefault(t *testing.T) {
	stub := &transportStub{running: true}
	stub.install(t)

	applyRetryConfig(0) // setting absent from the config

	for range defaultFailuresToStop - 1 {
		markCollectFailed(otherErr())
	}
	assert.Equal(t, 0, stub.stops)

	markCollectFailed(otherErr())
	assert.Equal(t, 1, stub.stops)
}

func TestApplyRetryConfigKeepsRedStatus(t *testing.T) {
	stub := &transportStub{running: true}
	stub.install(t)

	markCollectFailed(authErr())
	assert.Equal(t, 1, stub.stops)

	applyRetryConfig(2)
	assert.Equal(t, 0, failures, "a reconfigured connector gets a full set of attempts again")
	assert.True(t, stoppedByUs, "but it stays red until a cycle actually succeeds")
}

func TestMarkCollectOkRestoresStatus(t *testing.T) {
	stub := &transportStub{running: true}
	stub.install(t)

	markCollectFailed(authErr())
	assert.False(t, stub.running)

	markCollectOk()
	assert.Equal(t, 1, stub.starts)
	assert.True(t, stub.running)
	assert.False(t, stoppedByUs)
	assert.Equal(t, 0, failures)
}

func TestMarkCollectOkKeepsTransportUntouchedWhenNotStoppedByUs(t *testing.T) {
	stub := &transportStub{running: true}
	stub.install(t)

	markCollectFailed(otherErr()) // counted, but not enough to stop
	markCollectOk()

	assert.Equal(t, 0, stub.starts, "a successful cycle must not start a transport we did not stop")
	assert.Equal(t, 0, failures)
}

// A connector stopped from the UI or by the dispatcher must stay stopped: taking it
// over would let a successful collection cycle silently start it again.
func TestMarkCollectDoesNotTakeOverAnAlreadyStoppedTransport(t *testing.T) {
	stub := &transportStub{running: false}
	stub.install(t)

	markCollectFailed(authErr())
	assert.Equal(t, 0, stub.stops)
	assert.False(t, stoppedByUs)

	markCollectOk()
	assert.Equal(t, 0, stub.starts)
	assert.False(t, stub.running)
}

func TestMarkCollectFailedKeepsRetryingWhenStopFails(t *testing.T) {
	stub := &transportStub{running: true, stopErr: errors.New("task queue is full")}
	stub.install(t)

	markCollectFailed(authErr())
	assert.Equal(t, 1, stub.stops)
	assert.False(t, stoppedByUs, "a failed stop must not be recorded as ours")

	stub.stopErr = nil
	markCollectFailed(authErr())
	assert.Equal(t, 2, stub.stops, "the next cycle must retry the stop")
	assert.True(t, stoppedByUs)
}
