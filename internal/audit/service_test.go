package audit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/eventbus"
)

func TestAuditService_CreatePublishesEvent(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	published := make(chan struct{}, 1)
	bus.Subscribe(eventbus.NewListenerFunc(
		[]eventbus.EventType{"audit.log_created"},
		func(ctx context.Context, event eventbus.Event) error {
			published <- struct{}{}
			return nil
		},
	))

	al := &AuditLog{
		Role:       "admin",
		Action:     "test_action_service_create",
		EntityType: "product",
	}
	err := svc.CreateAuditLog(ctx, al)
	require.NoError(t, err)

	select {
	case <-published:
	case <-ctx.Done():
		t.Fatal("timeout waiting for audit.log_created event")
	}
}

func TestAuditService_GetAuditLogs(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	al := &AuditLog{
		Role:       "cashier",
		Action:     "test_action_service_list",
		EntityType: "product",
	}
	require.NoError(t, svc.CreateAuditLog(ctx, al))

	t.Run("list without filters", func(t *testing.T) {
		logs, total, err := svc.GetAuditLogs(ctx, 10, 0, nil, "", "", "", "", "")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		assert.GreaterOrEqual(t, len(logs), 1)
	})

	t.Run("filter by action", func(t *testing.T) {
		logs, total, err := svc.GetAuditLogs(ctx, 10, 0, nil, "", "test_action_service_list", "", "", "")
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		require.Len(t, logs, 1)
		assert.Equal(t, "test_action_service_list", logs[0].Action)
	})
}
