package salereturn

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// presentReturn reports the EFFECTIVE status: a pending return whose approval
// window elapsed displays as 'expired' even though no sweeper flipped the
// rows yet (lazy expiry, PRD §12.3 #16).
func TestPresentReturn_EffectiveStatus(t *testing.T) {
	past := time.Now().Add(-time.Minute)
	future := time.Now().Add(time.Minute)

	t.Run("pending past its window presents as expired", func(t *testing.T) {
		out := presentReturn(&Return{Status: ReturnStatusPendingApproval, ExpiresAt: &past})
		assert.Equal(t, ReturnStatusExpired, out["status"])
	})

	t.Run("pending inside its window stays pending", func(t *testing.T) {
		out := presentReturn(&Return{Status: ReturnStatusPendingApproval, ExpiresAt: &future})
		assert.Equal(t, ReturnStatusPendingApproval, out["status"])
	})

	t.Run("completed status is untouched", func(t *testing.T) {
		out := presentReturn(&Return{Status: ReturnStatusCompleted, ExpiresAt: &past})
		assert.Equal(t, ReturnStatusCompleted, out["status"])
	})

	t.Run("the approval row flips independently", func(t *testing.T) {
		out := presentReturn(&Return{
			Status:   ReturnStatusPendingApproval,
			Approval: &Approval{Status: "pending", ExpiresAt: past},
		})
		approval, ok := out["approval"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "expired", approval["status"])
	})

	t.Run("nil return presents as nil", func(t *testing.T) {
		assert.Nil(t, presentReturn(nil))
	})
}

func TestPresentReturns_EmptySliceIsNotNil(t *testing.T) {
	out := presentReturns(nil)
	assert.NotNil(t, out, "JSON consumers get [] rather than null")
	assert.Len(t, out, 0)
}
