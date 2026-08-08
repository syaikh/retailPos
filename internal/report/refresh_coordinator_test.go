package report

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func discardLogs(t *testing.T) {
	t.Helper()
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
}

func TestRefreshCoordinator_CoalescesEventsIntoOneRefresh(t *testing.T) {
	discardLogs(t)

	var runs atomic.Int64
	coord := NewRefreshCoordinator(50*time.Millisecond, func(ctx context.Context) error {
		runs.Add(1)
		return nil
	})
	coord.Start()
	defer coord.Shutdown()

	for i := 0; i < 4; i++ {
		coord.MarkDirty()
	}

	require.Eventually(t, func() bool { return runs.Load() >= 1 }, 2*time.Second, 10*time.Millisecond)
	// Give any incorrect extra refresh time to happen before asserting.
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, int64(1), runs.Load(), "events in one debounce window must coalesce into one refresh")
	assert.Equal(t, int64(3), coord.Metrics().Coalesced.Load(), "3 of 4 marks were coalesced")
	assert.False(t, coord.IsDirty(), "coordinator must return to clean after a successful refresh")
}

func TestRefreshCoordinator_SingleFlight_NoConcurrentRefresh(t *testing.T) {
	discardLogs(t)

	var runs, concurrent, maxConcurrent atomic.Int64
	release := make(chan struct{})

	coord := NewRefreshCoordinator(30*time.Millisecond, func(ctx context.Context) error {
		n := concurrent.Add(1)
		for {
			m := maxConcurrent.Load()
			if n <= m || maxConcurrent.CompareAndSwap(m, n) {
				break
			}
		}
		runs.Add(1)
		defer concurrent.Add(-1)
		if runs.Load() == 1 {
			<-release // keep refresh #1 in flight
		}
		return nil
	})
	coord.Start()
	defer coord.Shutdown()

	coord.MarkDirty()

	require.Eventually(t, func() bool { return runs.Load() == 1 }, 2*time.Second, 5*time.Millisecond)
	require.Eventually(t, func() bool { return concurrent.Load() == 1 }, 2*time.Second, 5*time.Millisecond)

	// Sales arrive while refresh #1 is running: they must not spawn refresh #2.
	coord.MarkDirty()
	coord.MarkDirty()
	coord.MarkDirty()

	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, int64(1), runs.Load(), "no second refresh may start while the first is running")
	assert.Equal(t, int64(1), maxConcurrent.Load(), "at most one refresh may run concurrently")

	// Release refresh #1; a follow-up refresh must be scheduled because the
	// coordinator remained dirty.
	close(release)
	require.Eventually(t, func() bool { return runs.Load() >= 2 }, 2*time.Second, 10*time.Millisecond)
	assert.Equal(t, int64(1), maxConcurrent.Load(), "max concurrent refresh must remain 1")
	assert.Equal(t, int64(3), coord.Metrics().Coalesced.Load())
	assert.False(t, coord.IsDirty())
}

func TestRefreshCoordinator_NoEvents_NoRefresh(t *testing.T) {
	discardLogs(t)

	var runs atomic.Int64
	coord := NewRefreshCoordinator(20*time.Millisecond, func(ctx context.Context) error {
		runs.Add(1)
		return nil
	})
	coord.Start()
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, int64(0), runs.Load(), "no SaleCreated events must never trigger a refresh")
	coord.Shutdown()
}

func TestRefreshCoordinator_Failure_IsolatedAndRetried(t *testing.T) {
	discardLogs(t)

	var runs atomic.Int64
	coord := NewRefreshCoordinator(20*time.Millisecond, func(ctx context.Context) error {
		if runs.Add(1) < 3 {
			return errors.New("refresh failed")
		}
		return nil
	})
	coord.Start()
	defer coord.Shutdown()

	// MarkDirty is the event-handler contract: it must never surface an error
	// even when the underlying refresh keeps failing.
	coord.MarkDirty()

	require.Eventually(t, func() bool { return runs.Load() >= 3 }, 3*time.Second, 10*time.Millisecond)
	assert.Equal(t, int64(2), coord.Metrics().Failed.Load())
	assert.GreaterOrEqual(t, coord.Metrics().Succeeded.Load(), int64(1))
	assert.False(t, coord.IsDirty(), "coordinator returns to clean once a retry succeeds")
}

func TestRefreshCoordinator_EventsDuringFailedRefresh_Coalesce(t *testing.T) {
	discardLogs(t)

	var runs atomic.Int64
	release := make(chan struct{})
	coord := NewRefreshCoordinator(20*time.Millisecond, func(ctx context.Context) error {
		if runs.Add(1) == 1 {
			<-release
			return errors.New("refresh failed")
		}
		return nil
	})
	coord.Start()
	defer coord.Shutdown()

	coord.MarkDirty()
	require.Eventually(t, func() bool { return runs.Load() == 1 }, 2*time.Second, 5*time.Millisecond)

	// Events during the failed refresh must fold into the existing dirty state.
	coord.MarkDirty()
	coord.MarkDirty()
	coord.MarkDirty()

	time.Sleep(120 * time.Millisecond)
	assert.Equal(t, int64(1), runs.Load(), "events during a failed refresh must coalesce, not spawn parallel retries")

	close(release)
	require.Eventually(t, func() bool { return runs.Load() >= 2 }, 2*time.Second, 10*time.Millisecond)
	assert.False(t, coord.IsDirty())
}

func TestRefreshCoordinator_RecoveryAfterFailure(t *testing.T) {
	discardLogs(t)

	var runs atomic.Int64
	coord := NewRefreshCoordinator(15*time.Millisecond, func(ctx context.Context) error {
		if runs.Add(1) == 1 {
			return errors.New("boom")
		}
		return nil
	})
	coord.Start()
	defer coord.Shutdown()

	coord.MarkDirty()
	require.Eventually(t, func() bool { return runs.Load() == 1 }, 2*time.Second, 5*time.Millisecond)
	assert.True(t, coord.IsDirty(), "coordinator must remain dirty after a failed refresh")

	require.Eventually(t, func() bool { return runs.Load() >= 2 }, 2*time.Second, 10*time.Millisecond)
	assert.Equal(t, int64(1), coord.Metrics().Failed.Load())
	assert.Equal(t, int64(1), coord.Metrics().Succeeded.Load())
	assert.False(t, coord.IsDirty(), "coordinator must return to the clean state after a successful retry")
}

func TestRefreshCoordinator_Shutdown_NoLeak(t *testing.T) {
	discardLogs(t)

	coord := NewRefreshCoordinator(10*time.Millisecond, func(ctx context.Context) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})
	coord.Start()
	coord.MarkDirty()

	done := make(chan struct{})
	go func() {
		coord.Shutdown()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown did not complete")
	}

	// MarkDirty and Shutdown after shutdown must be safe and non-blocking.
	coord.MarkDirty()
	coord.Shutdown()
}

func TestRefreshCoordinator_ShutdownWithoutStart(t *testing.T) {
	coord := NewRefreshCoordinator(time.Second, func(ctx context.Context) error { return nil })
	coord.Shutdown()
}

func TestRefreshCoordinator_MarkDirtyBeforeStart(t *testing.T) {
	discardLogs(t)

	var runs atomic.Int64
	coord := NewRefreshCoordinator(20*time.Millisecond, func(ctx context.Context) error {
		runs.Add(1)
		return nil
	})

	coord.MarkDirty()
	coord.MarkDirty()
	assert.Equal(t, int64(0), runs.Load())

	coord.Start()
	defer coord.Shutdown()
	require.Eventually(t, func() bool { return runs.Load() == 1 }, 2*time.Second, 10*time.Millisecond)
}
