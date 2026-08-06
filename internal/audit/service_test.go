package audit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditService_CreateAuditLog(t *testing.T) {
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	al := &Log{
		Role:       "admin",
		Action:     "test_action_service_create",
		EntityType: "product",
	}
	err := svc.CreateAuditLog(ctx, al)
	require.NoError(t, err)
	assert.Greater(t, al.ID, 0)
}

func TestAuditService_GetAuditLogs(t *testing.T) {
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	al := &Log{
		Role:       "cashier",
		Action:     "test_action_service_list",
		EntityType: "product",
	}
	require.NoError(t, svc.CreateAuditLog(ctx, al))

	t.Run("list without filters", func(t *testing.T) {
		logs, total, err := svc.GetAuditLogs(ctx, 10, 0, nil, "", "", "", nil, "", "")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		assert.GreaterOrEqual(t, len(logs), 1)
	})

	t.Run("filter by action", func(t *testing.T) {
		logs, total, err := svc.GetAuditLogs(ctx, 10, 0, nil, "", "test_action_service_list", "", nil, "", "")
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		require.Len(t, logs, 1)
		assert.Equal(t, "test_action_service_list", logs[0].Action)
	})
}
