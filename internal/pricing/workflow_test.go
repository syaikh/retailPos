package pricing

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	importexportshared "retail-pos-system/internal/shared/importexport"
)

// Every pricing rule must be created pending and inactive, whichever way it is
// created, and only Approve may make it reach the till. These tests pin the
// create paths that the API does not obviously cover: the service (API), the
// import adapter, and the two bulk SQL statements behind it.

func newWorkflowProduct(ctx context.Context, t *testing.T, tag string) int {
	return insertTestProduct(ctx, t, "WF-"+tag+"-"+time.Now().Format("0102150405.000"), "Workflow "+tag+" Product", 15000)
}

func uniqueRuleName(tag string) string {
	return "WF " + tag + " " + time.Now().Format("0102150405.000000")
}

func TestServiceCreateStartsPendingAndInactive(t *testing.T) {
	skipIfNoDB(t)
	repo := newWiredRepo()
	svc := NewService(repo)
	ctx := t.Context()
	productID := newWorkflowProduct(ctx, t, "CRT")

	tests := []struct {
		name     string
		status   RuleStatus
		isActive bool
	}{
		// The frontend create form omits status entirely, which is how a rule
		// used to land approved+active straight from the UI.
		{"status omitted", "", false},
		{"client asks for approved", StatusApproved, true},
		{"client asks for rejected", StatusRejected, true},
		{"client asks for draft", StatusDraft, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rule := &Rule{
				ProductID:       &productID,
				Type:            PricingTypePromotion,
				Method:          PricingMethodFixedPrice,
				PricingValue:    10000,
				Name:            uniqueRuleName(tc.name),
				MinimumQuantity: 1,
				Status:          tc.status,
				IsActive:        tc.isActive,
			}
			require.NoError(t, svc.Create(ctx, rule))

			stored, err := svc.GetByID(ctx, rule.ID)
			require.NoError(t, err)
			assert.Equal(t, StatusPending, stored.Status, "create must not let the caller choose the status")
			assert.False(t, stored.IsActive, "create must not let the caller activate a rule")
		})
	}
}

func TestApproveIsTheOnlyPathToActive(t *testing.T) {
	skipIfNoDB(t)
	repo := newWiredRepo()
	svc := NewService(repo)
	ctx := t.Context()
	productID := newWorkflowProduct(ctx, t, "APR")

	rule := &Rule{
		ProductID:       &productID,
		Type:            PricingTypePromotion,
		Method:          PricingMethodFixedPrice,
		PricingValue:    10000,
		Name:            uniqueRuleName("approve"),
		MinimumQuantity: 1,
	}
	require.NoError(t, svc.Create(ctx, rule))

	// Pending rules are invisible to the till even though the row exists.
	rules, err := repo.GetActiveRules(ctx, productID, nil, nil, time.Now(), nil, nil)
	require.NoError(t, err)
	assert.Empty(t, rules, "a pending rule must not resolve at checkout")

	require.NoError(t, svc.Approve(ctx, rule.ID))

	stored, err := svc.GetByID(ctx, rule.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusApproved, stored.Status)
	assert.True(t, stored.IsActive, "approval is what activates a rule")

	rules, err = repo.GetActiveRules(ctx, productID, nil, nil, time.Now(), nil, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, rules, "an approved+active rule must resolve at checkout")
}

func TestRejectLeavesRuleInactive(t *testing.T) {
	skipIfNoDB(t)
	repo := newWiredRepo()
	svc := NewService(repo)
	ctx := t.Context()
	productID := newWorkflowProduct(ctx, t, "REJ")

	rule := &Rule{
		ProductID:       &productID,
		Type:            PricingTypePromotion,
		Method:          PricingMethodFixedPrice,
		PricingValue:    10000,
		Name:            uniqueRuleName("reject"),
		MinimumQuantity: 1,
	}
	require.NoError(t, svc.Create(ctx, rule))
	require.NoError(t, svc.Reject(ctx, rule.ID))

	stored, err := svc.GetByID(ctx, rule.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusRejected, stored.Status)
	assert.False(t, stored.IsActive)

	rules, err := repo.GetActiveRules(ctx, productID, nil, nil, time.Now(), nil, nil)
	require.NoError(t, err)
	assert.Empty(t, rules, "a rejected rule must not resolve at checkout")
}

// status is the gate, independent of is_active. Both rows below are active in
// the database; only one is approved.
func TestGetActiveRulesRequiresApprovedNotJustActive(t *testing.T) {
	skipIfNoDB(t)
	repo := newWiredRepo()
	svc := NewService(repo)
	ctx := t.Context()
	productID := newWorkflowProduct(ctx, t, "GATE")

	pending := &Rule{
		ProductID:       &productID,
		Type:            PricingTypePromotion,
		Method:          PricingMethodFixedPrice,
		PricingValue:    10000,
		Name:            uniqueRuleName("gate-pending"),
		MinimumQuantity: 1,
	}
	require.NoError(t, svc.Create(ctx, pending))
	// Force is_active on behind the service's back: the point is that an active
	// pending rule still must not resolve.
	stored, err := svc.GetByID(ctx, pending.ID)
	require.NoError(t, err)
	stored.IsActive = true
	require.NoError(t, repo.Update(ctx, stored))

	rules, err := repo.GetActiveRules(ctx, productID, nil, nil, time.Now(), nil, nil)
	require.NoError(t, err)
	assert.Empty(t, rules, "is_active alone must not make a pending rule reach the till")
}

// The schema defaults are the backstop. Every Go path pins status and is_active
// explicitly, so the column defaults are unreachable from the application — but an
// INSERT that omits them (a future service, a psql fix, a script) used to produce
// an approved and active rule on insert, which is the immediate-active creation
// the workflow exists to prevent. Migration 055 realigns the defaults.
func TestPricingRulesSchemaDefaultsStartPending(t *testing.T) {
	skipIfNoDB(t)
	ctx := t.Context()
	productID := newWorkflowProduct(ctx, t, "DEF")
	ruleName := uniqueRuleName("default")

	// status and is_active are deliberately omitted, which is what makes this a
	// test of the column defaults rather than of the service.
	_, err := dbPool.Exec(ctx, `
		INSERT INTO pricing_rules (product_id, name, pricing_type, pricing_method,
		       pricing_value, minimum_quantity)
		VALUES ($1, $2, 'promotion', 'fixed_price', 10000, 1)
	`, productID, ruleName)
	require.NoError(t, err)

	var got Rule
	require.NoError(t, dbPool.QueryRow(ctx,
		`SELECT status, is_active FROM pricing_rules WHERE name = $1`, ruleName,
	).Scan(&got.Status, &got.IsActive))
	assert.Equal(t, StatusPending, got.Status, "the status column default must not hand out approval")
	assert.False(t, got.IsActive, "the is_active column default must not hand out activation")
}

// --- import path ---------------------------------------------------------

func TestMapToEntityTakesStoreFromInjectedClaim(t *testing.T) {
	a := &adapter{}
	ctx := context.Background()

	row := map[string]interface{}{
		"_row":            2,
		"Type":            string(PricingTypePromotion),
		"Method":          string(PricingMethodFixedPrice),
		"PricingValue":    float64(10000),
		"Name":            "Imported Rule",
		"MinimumQuantity": float64(1),
		"IsActive":        true,
		"ProductID":       float64(1),
		"_store_id":       7,
	}

	entity, err := a.MapToEntity(ctx, importexportshared.ModuleSchema{}, row)
	require.NoError(t, err)
	imported := entity.(RuleImportRow)
	require.NotNil(t, imported.StoreID, "import must not drop the importer's store")
	assert.Equal(t, 7, *imported.StoreID)

	// A superadmin has no store claim, so nothing is injected and the rule stays
	// global. That is the only way an imported rule may be global.
	delete(row, "_store_id")
	entity, err = a.MapToEntity(ctx, importexportshared.ModuleSchema{}, row)
	require.NoError(t, err)
	imported = entity.(RuleImportRow)
	assert.Nil(t, imported.StoreID)
}

func TestBulkInsertPricingRulesPinsPendingAndInactive(t *testing.T) {
	skipIfNoDB(t)
	repo := newWiredRepo()
	ctx := t.Context()
	productID := newWorkflowProduct(ctx, t, "IMP")

	storeID := insertTestStore(ctx, t, "WF-IMPORT-STORE")
	ruleName := uniqueRuleName("import")
	n, err := repo.BulkInsertPricingRules(ctx, []RuleImportPayload{{
		ProductID:       &productID,
		Type:            string(PricingTypePromotion),
		Method:          string(PricingMethodFixedPrice),
		PricingValue:    10000,
		Name:            ruleName,
		MinimumQuantity: 1,
		// A CSV may carry these columns. Neither may reach the database.
		IsActive: true,
		StoreID:  &storeID,
	}})
	require.NoError(t, err)
	require.Equal(t, 1, n)

	// GetAll's search matches product/category/brand names, not the rule name,
	// so read the row back directly.
	var got Rule
	var gotStore *int
	err = dbPool.QueryRow(ctx,
		`SELECT status, is_active, store_id FROM pricing_rules WHERE name = $1`,
		ruleName,
	).Scan(&got.Status, &got.IsActive, &gotStore)
	require.NoError(t, err)
	assert.Equal(t, StatusPending, got.Status, "an import must not create an approved rule")
	assert.False(t, got.IsActive, "an import must not create an active rule")
	require.NotNil(t, gotStore, "an import must inherit the importer's store")
	assert.Equal(t, storeID, *gotStore)
}

func TestBulkUpdatePricingRulesPreservesStatusAndScope(t *testing.T) {
	skipIfNoDB(t)
	repo := newWiredRepo()
	svc := NewService(repo)
	ctx := t.Context()
	productID := newWorkflowProduct(ctx, t, "IMPUPD")

	ownStore := insertTestStore(ctx, t, "WF-UPD-OWN")
	rule := &Rule{
		ProductID:       &productID,
		Type:            PricingTypePromotion,
		Method:          PricingMethodFixedPrice,
		PricingValue:    10000,
		Name:            uniqueRuleName("import-update"),
		MinimumQuantity: 1,
		StoreID:         &ownStore,
	}
	require.NoError(t, svc.Create(ctx, rule))
	require.NoError(t, svc.Approve(ctx, rule.ID))

	// An import update tries to promote the rule to global and to a different
	// store. The name/product/type triple is unchanged, so without the store_id
	// guard in the WHERE this would silently re-home an approved rule.
	otherStore := insertTestStore(ctx, t, "WF-UPD-OTHER")
	n, err := repo.BulkUpdatePricingRules(ctx, []RuleImportPayload{{
		ProductID:       &productID,
		Type:            string(PricingTypePromotion),
		Name:            rule.Name,
		MinimumQuantity: 1,
		PricingValue:    20000,
		IsActive:        true,
		StoreID:         &otherStore,
	}})
	require.NoError(t, err)
	assert.Equal(t, 0, n, "an import must not update a rule outside the importer's store")

	stored, err := svc.GetByID(ctx, rule.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusApproved, stored.Status)
	require.NotNil(t, stored.StoreID)
	assert.Equal(t, ownStore, *stored.StoreID, "an import must not re-home a rule to another store")

	// The same payload scoped to the rule's own store updates the commercial
	// terms but leaves the approval state alone.
	n, err = repo.BulkUpdatePricingRules(ctx, []RuleImportPayload{{
		ProductID:       &productID,
		Type:            string(PricingTypePromotion),
		Name:            rule.Name,
		MinimumQuantity: 1,
		PricingValue:    20000,
		IsActive:        true,
		StoreID:         &ownStore,
	}})
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	stored, err = svc.GetByID(ctx, rule.ID)
	require.NoError(t, err)
	assert.Equal(t, 20000.0, stored.PricingValue)
	assert.Equal(t, StatusApproved, stored.Status)
}
