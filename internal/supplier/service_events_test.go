package supplier

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/events"
)

type recordingBus struct {
	mu     sync.Mutex
	topics []string
	events []interface{}
}

func (b *recordingBus) Publish(_ context.Context, topic string, event interface{}) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.topics = append(b.topics, topic)
	b.events = append(b.events, event)
	return nil
}

func (b *recordingBus) snapshot() ([]string, []interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.topics...), append([]interface{}(nil), b.events...)
}

func TestService_PublishesSupplierChangedOnDeactivate(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)
	bus := &recordingBus{}
	svc := NewService(repo)
	svc.SetEventBus(bus)

	s := createGovSupplier(ctx, t, repo, "EVT-DEACT-"+govSuffix(), true)
	s.IsActive = false
	require.NoError(t, svc.Update(ctx, s))

	topics, evts := bus.snapshot()
	require.Len(t, evts, 1)
	assert.Equal(t, events.TopicSupplierChanged, topics[0])
	ev, ok := evts[0].(*events.SupplierChanged)
	require.True(t, ok)
	assert.Equal(t, events.SupplierActionDeactivated, ev.Action)
	assert.Equal(t, s.ID, ev.SupplierID)
	assert.Equal(t, s.Code, ev.Code)
	assert.Greater(t, ev.Version, 1, "the deactivation event carries the bumped version")
}

func TestService_PublishesSupplierChangedOnDelete(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)
	bus := &recordingBus{}
	svc := NewService(repo)
	svc.SetEventBus(bus)

	s := createGovSupplier(ctx, t, repo, "EVT-DEL-"+govSuffix(), true)
	require.NoError(t, svc.Delete(ctx, s.ID))

	topics, evts := bus.snapshot()
	require.Len(t, evts, 1)
	assert.Equal(t, events.TopicSupplierChanged, topics[0])
	ev, ok := evts[0].(*events.SupplierChanged)
	require.True(t, ok)
	assert.Equal(t, events.SupplierActionDeleted, ev.Action)
	assert.Equal(t, s.ID, ev.SupplierID)
	assert.Equal(t, s.Code, ev.Code)
}

func TestService_NoEventWhenAvailabilityUnchanged(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)
	bus := &recordingBus{}
	svc := NewService(repo)
	svc.SetEventBus(bus)

	t.Run("ordinary field edit on an active supplier", func(t *testing.T) {
		s := createGovSupplier(ctx, t, repo, "EVT-EDIT-"+govSuffix(), true)
		s.Name = s.Name + " renamed"
		require.NoError(t, svc.Update(ctx, s))
	})

	t.Run("reactivation", func(t *testing.T) {
		s := createGovSupplier(ctx, t, repo, "EVT-REACT-"+govSuffix(), false)
		s.IsActive = true
		require.NoError(t, svc.Update(ctx, s))
	})

	_, evts := bus.snapshot()
	assert.Empty(t, evts, "only active -> inactive and delete announce a change")
}

func TestService_NilEventBusIsSafe(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)
	svc := NewService(repo)

	s := createGovSupplier(ctx, t, repo, "EVT-NIL-"+govSuffix(), true)
	s.IsActive = false
	require.NoError(t, svc.Update(ctx, s))
}
