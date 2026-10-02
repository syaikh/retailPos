package pricing

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests read the grants that migrations actually produced and compare them
// to the role mapping the audit settled on. The route-level tests in
// routes_permission_test.go prove which code each route *demands*; nothing
// there proves who *holds* those codes, so a later baseline edit could quietly
// re-grant a sub-manager role and every existing test would stay green.
//
// This is the check that would have caught migration 056's problem earlier: the
// baseline only INSERTs into role_permissions, so dropping a code from a grant
// list stops it being added on a fresh install but never removes it from an
// already-migrated one.

// auditedRoleGrants is the expected grant set per role, restricted to the
// namespaces Wave 2 and Wave 4 decided. Anything not listed here is not
// covered, which is why the test asserts over exactly these three prefixes.
var auditedRoleGrants = map[string][]string{
	// Cashier sells. It can read prices so the till can price a cart, and it
	// owns its shift, but it must not be able to change or approve a rule.
	"cashier": {
		"pricing.view",
		"shift.cash_movement", "shift.create", "shift.view",
	},
	// Manager is the store-level pricing owner: full mutate, full approve, and
	// full supplier access (audit D2).
	"manager": {
		"pricing.approve", "pricing.create", "pricing.delete", "pricing.update", "pricing.view",
		"shift.audit", "shift.cash_movement", "shift.create", "shift.review", "shift.view",
		"supplier.create", "supplier.delete", "supplier.update", "supplier.view",
	},
	// Superadmin is unrestricted by design.
	"superadmin": {
		"pricing.approve", "pricing.create", "pricing.delete", "pricing.update", "pricing.view",
		"shift.audit", "shift.cash_movement", "shift.create", "shift.review", "shift.view",
		"supplier.create", "supplier.delete", "supplier.update", "supplier.view",
	},
	// Supervisor reads prices to supervise the floor and can audit a shift, but
	// it must NOT hold pricing.create/update/delete (migration 056 revoked
	// exactly that over-grant) and must NOT mutate suppliers (audit D2 leaves
	// supplier.view only).
	"supervisor": {
		"pricing.view",
		"shift.audit", "shift.cash_movement", "shift.create", "shift.review", "shift.view",
		"supplier.view",
	},
}

// loadGrants returns every code each role holds within the audited namespaces.
func loadGrants(t *testing.T) map[string][]string {
	t.Helper()
	rows, err := dbPool.Query(t.Context(), `
		SELECT r.name, p.code
		FROM role_permissions rp
		JOIN roles r ON r.id = rp.role_id
		JOIN permissions p ON p.id = rp.permission_id
		WHERE p.code LIKE 'pricing.%' OR p.code LIKE 'supplier.%' OR p.code LIKE 'shift.%'
	`)
	require.NoError(t, err)
	defer rows.Close()

	out := map[string][]string{}
	for rows.Next() {
		var role, code string
		require.NoError(t, rows.Scan(&role, &code))
		out[role] = append(out[role], code)
	}
	require.NoError(t, rows.Err())

	for role := range out {
		sort.Strings(out[role])
	}
	return out
}

func TestAuditedRoleGrants_MatchIntendedMapping(t *testing.T) {
	skipIfNoDB(t)
	got := loadGrants(t)

	for role, want := range auditedRoleGrants {
		t.Run(role, func(t *testing.T) {
			require.Contains(t, got, role, "role must exist in the baseline seed")
			assert.Equal(t, want, got[role],
				"grants drifted from the audited mapping: pricing mutation on a "+
					"sub-manager role, or a lost read, both land here")
		})
	}
}

// The individual assertions that the audit called out by name, so a failure
// names the specific regression instead of dumping two code lists.
func TestSupervisorCannotMutatePricing(t *testing.T) {
	skipIfNoDB(t)
	got := loadGrants(t)
	require.Contains(t, got, "supervisor")

	for _, forbidden := range []string{"pricing.create", "pricing.update", "pricing.delete"} {
		assert.NotContains(t, got["supervisor"], forbidden,
			"migration 056 revoked this grant; the baseline only INSERTs grants, so "+
				"nothing else will take it back")
	}
	assert.Contains(t, got["supervisor"], "pricing.view", "supervision still needs read access")
}

func TestSubManagerRolesCannotApprovePricing(t *testing.T) {
	skipIfNoDB(t)
	got := loadGrants(t)

	for _, role := range []string{"cashier", "supervisor"} {
		require.Contains(t, got, role)
		assert.NotContains(t, got[role], "pricing.approve",
			"approval must not sit below manager; it would make self-approval reachable")
	}
	for _, role := range []string{"manager", "superadmin"} {
		require.Contains(t, got, role)
		assert.Contains(t, got[role], "pricing.approve")
	}
}

// A cashier has no supplier.* code at all, which is what makes the audit's
// "cashier gets 403 on GET /suppliers" hold at the middleware.
func TestCashierHasNoSupplierAccess(t *testing.T) {
	skipIfNoDB(t)
	got := loadGrants(t)
	require.Contains(t, got, "cashier")

	for _, code := range got["cashier"] {
		assert.NotContains(t, code, "supplier.",
			"supplier data is commercially sensitive and must not reach the till")
	}
}
