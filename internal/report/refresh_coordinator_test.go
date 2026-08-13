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

func TestRefreshCoordinator_StartupRefresh(t *testing.T) {
	discardLogs(t)

	var runs atomic.Int64
	coord := NewRefreshCoordinator(time.Second, func(ctx context.Context) error {
		runs.Add(1)
		return nil
	})
	coord.Start()
	defer coord.Shutdown()

	require.Eventually(t, func() bool { return runs.Load() >= 1 }, 2*time.Second, 10*time.Millisecond)
	assert.Equal(t, int64(1), runs.Load(), "coordinator must refresh immediately on startup")
}

func TestRefreshCoordinator_SingleFlight(t *testing.T) {
	discardLogs(t)

	var runs, concurrent, maxConcurrent atomic.Int64
	release := make(chan struct{})

	coord := NewRefreshCoordinator(time.Second, func(ctx context.Context) error {
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
			<-release
		}
		return nil
	})
	coord.Start()
	defer coord.Shutdown()

	require.Eventually(t, func() bool { return runs.Load() == 1 }, 2*time.Second, 5*time.Millisecond)
	require.Eventually(t, func() bool { return concurrent.Load() == 1 }, 2*time.Second, 5*time.Millisecond)

	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, int64(1), runs.Load(), "no second refresh may start while the first is running")
	assert.Equal(t, int64(1), maxConcurrent.Load(), "at most one refresh may run concurrently")

	close(release)
	require.Eventually(t, func() bool { return concurrent.Load() == 0 }, 2*time.Second, 10*time.Millisecond)
	assert.Equal(t, int64(1), maxConcurrent.Load(), "max concurrent refresh must remain 1")
}

func TestRefreshCoordinator_RetryAfterFailure(t *testing.T) {
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

	require.Eventually(t, func() bool { return runs.Load() >= 3 }, 3*time.Second, 10*time.Millisecond)
	assert.Equal(t, int64(2), coord.Metrics().Failed.Load())
	assert.GreaterOrEqual(t, coord.Metrics().Succeeded.Load(), int64(1))
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

	require.Eventually(t, func() bool { return runs.Load() == 1 }, 2*time.Second, 5*time.Millisecond)
	assert.Equal(t, int64(1), coord.Metrics().Failed.Load())

	require.Eventually(t, func() bool { return runs.Load() >= 2 }, 2*time.Second, 10*time.Millisecond)
	assert.Equal(t, int64(1), coord.Metrics().Succeeded.Load())
}

func TestRefreshCoordinator_Shutdown_NoLeak(t *testing.T) {
	discardLogs(t)

	coord := NewRefreshCoordinator(time.Second, func(ctx context.Context) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})
	coord.Start()

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
}

func TestRefreshCoordinator_ShutdownWithoutStart(t *testing.T) {
	coord := NewRefreshCoordinator(time.Second, func(ctx context.Context) error { return nil })
	coord.Shutdown()
}
