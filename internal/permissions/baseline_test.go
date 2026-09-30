package permissions

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// baselinePath is 000_baseline.sql, which is the authoritative seed for the
// permissions table and the role grants. The Go catalog in permissions.go and
// that file are edited separately, so without this test a code can be added to
// All() but never granted, or granted by id in SQL while the registry rejects
// it, and nothing notices until a permission check 403s at runtime.
const baselinePath = "../../database/migrations/000_baseline.sql"

func readBaseline(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.FromSlash(baselinePath))
	if err != nil {
		t.Fatalf("read %s: %v", baselinePath, err)
	}
	return string(data)
}

// permissionRowRe captures the code out of each seeded permissions row. The code
// is dotted and may have more than one dot (product.cost.view), hence the
// repeating group rather than a single [a-z_]+\.[a-z_]+.
var permissionRowRe = regexp.MustCompile(`VALUES \(\d+, '([a-z_]+(?:\.[a-z_]+)+)',`)

func TestBaselineSeedsEveryRegisteredCode(t *testing.T) {
	sql := readBaseline(t)
	seeded := make(map[string]bool)
	for _, m := range permissionRowRe.FindAllStringSubmatch(sql, -1) {
		seeded[m[1]] = true
	}
	if len(seeded) == 0 {
		t.Fatal("no permission rows parsed from 000_baseline.sql")
	}
	for _, c := range All() {
		if !seeded[c.String()] {
			t.Errorf("code %q is in All() but has no INSERT row in 000_baseline.sql", c)
		}
	}
	for code := range seeded {
		if !Exists(Code(code)) {
			t.Errorf("000_baseline.sql seeds %q but it is not registered in permissions.go", code)
		}
	}
}

// roleGrantRe captures the ARRAY[...] payload of one role's grant list, keyed by
// the "-- <role>: N permissions" comment that precedes it. (?s) lets the
// .*? span the newlines between the comment and the INSERT.
var roleGrantRe = regexp.MustCompile(
	`(?s)-- (\w+): (\d+) permissions\nINSERT INTO public\.role_permissions.*?ARRAY\[(.*?)\]\)`,
)

func baselineRoleGrants(t *testing.T, sql string) map[string][]string {
	t.Helper()
	grants := make(map[string][]string)
	for _, m := range roleGrantRe.FindAllStringSubmatch(sql, -1) {
		role, declared := m[1], m[2]
		var codes []string
		for _, part := range strings.Split(m[3], ",") {
			if p := strings.Trim(strings.TrimSpace(part), "'"); p != "" {
				codes = append(codes, p)
			}
		}
		if len(codes) != atoiOrZero(declared) {
			t.Errorf("role %s: comment says %s permissions, ARRAY has %d", role, declared, len(codes))
		}
		if !sort.StringsAreSorted(codes) {
			t.Errorf("role %s: grants are not in sorted order, which makes diffs noisy", role)
		}
		grants[role] = codes
	}
	if len(grants) == 0 {
		t.Fatal("no role grant lists parsed from 000_baseline.sql")
	}
	return grants
}

func atoiOrZero(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func granted(t *testing.T, grants map[string][]string, role string, codes ...Code) {
	t.Helper()
	have := make(map[string]bool, len(grants[role]))
	for _, c := range grants[role] {
		have[c] = true
	}
	for _, c := range codes {
		if !have[c.String()] {
			t.Errorf("role %s should hold %q but the baseline does not grant it", role, c)
		}
	}
}

func notGranted(t *testing.T, grants map[string][]string, role string, codes ...Code) {
	t.Helper()
	have := make(map[string]bool, len(grants[role]))
	for _, c := range grants[role] {
		have[c] = true
	}
	for _, c := range codes {
		if have[c.String()] {
			t.Errorf("role %s should not hold %q but the baseline grants it", role, c)
		}
	}
}

// TestBaselineRoleGrantsMatchPolicy pins the store-boundary decisions from
// docs/design/master-data-store-boundary-audit.md (waves 2c and 2d) into the
// seed data, so a later edit that re-grants pricing.write to a shift lead fails
// here rather than in production.
func TestBaselineRoleGrantsMatchPolicy(t *testing.T) {
	grants := baselineRoleGrants(t, readBaseline(t))

	// A shift lead can read pricing but must not author, edit, approve, or
	// delete it — that was the audit's sharpest finding.
	granted(t, grants, "supervisor", PricingView, SupplierView)
	notGranted(t, grants, "supervisor", PricingCreate, PricingUpdate, PricingDelete, PricingApprove)
	notGranted(t, grants, "supervisor", SupplierCreate, SupplierUpdate, SupplierDelete)

	granted(t, grants, "manager", SupplierView, SupplierCreate, SupplierUpdate, SupplierDelete, PricingApprove)

	// Superadmin holds every registered code except one. RequirePermission does
	// not bypass for superadmin (internal/middleware/auth.go), so a code absent
	// from this grant list is a 403 for the most privileged role in the system.
	//
	// sale.lookup is the single deliberate exception, not a defect. Migration
	// 031_revoke_sale_lookup_manager.sql removed it from manager on the recorded
	// decision that cross-cashier "Find Transaction" is a cashier-only
	// capability: roles with report.view already see every cashier's sales in
	// "My Transactions" (ownership.CanAccessAll), so the lookup tab is a weaker
	// redacted subset and TransactionsPage.svelte hides the tab bar for them
	// entirely. Granting it to superadmin would widen the API surface against
	// that decision, and web/src/shared/composables/__tests__/useRBAC.test.ts
	// already asserts superadmin = ALL_PERMISSIONS minus sale.lookup. Do not add
	// it here without revisiting that decision on both sides.
	superadminExcludedByDesign := map[Code]string{
		SaleLookup: "cashier-only by design (031_revoke_sale_lookup_manager.sql)",
	}
	for _, c := range All() {
		if _, byDesign := superadminExcludedByDesign[c]; byDesign {
			notGranted(t, grants, "superadmin", c)
			continue
		}
		granted(t, grants, "superadmin", c)
	}

	// Front-of-house and back-office roles have no business seeing suppliers,
	// and in particular no path to unit cost.
	for _, role := range []string{"cashier", "finance", "inventory_staff"} {
		notGranted(t, grants, role, SupplierView, SupplierCreate, SupplierUpdate, SupplierDelete, PricingApprove)
	}
}
