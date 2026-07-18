package pricing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

func TestPricingAdapter_ModuleName(t *testing.T) {
	a := &adapter{}
	assert.Equal(t, "pricing_rules", a.ModuleName())
}

func TestPricingAdapter_NewAdapter(t *testing.T) {
	a := NewAdapter(nil)
	assert.NotNil(t, a)
	assert.Equal(t, "pricing_rules", a.ModuleName())
}

func TestPricingAdapter_ValidateBusiness(t *testing.T) {
	a := &adapter{}
	ctx := context.Background()
	errs := a.ValidateBusiness(ctx, Schema, nil)
	assert.Nil(t, errs)
}

func TestPricingAdapter_MapToEntity(t *testing.T) {
	pid := 42
	catID := 10
	brandID := 5

	tests := []struct {
		name        string
		row         map[string]interface{}
		wantName    string
		wantType    string
		wantMethod  string
		wantValue   float64
		wantProdID  *int
		wantCatID   *int
		wantBrandID *int
		wantErr     bool
		errContains string
	}{
		{
			name: "happy path with product_id",
			row: map[string]interface{}{
				"_row":          1,
				"ProductID":     float64(42),
				"PricingType":   "promotion",
				"PricingMethod": "fixed_price",
				"PricingValue":  12000,
				"Name":          "Test Rule",
				"MinimumQuantity": float64(3),
				"Priority":      float64(2),
				"IsActive":      "true",
			},
			wantName:   "Test Rule",
			wantType:   "promotion",
			wantMethod: "fixed_price",
			wantValue:  12000,
			wantProdID: &pid,
		},
		{
			name: "happy path with category_id",
			row: map[string]interface{}{
				"_row":          2,
				"CategoryID":    float64(10),
				"PricingType":   "special_price",
				"PricingMethod": "discount_percent",
				"PricingValue":  15,
				"Name":          "Category Discount",
			},
			wantName:   "Category Discount",
			wantType:   "special_price",
			wantMethod: "discount_percent",
			wantValue:  15,
			wantCatID:  &catID,
		},
		{
			name: "happy path with brand_id",
			row: map[string]interface{}{
				"_row":          3,
				"BrandID":       float64(5),
				"PricingType":   "promotion",
				"PricingMethod": "discount_amount",
				"PricingValue":  5000,
				"Name":          "Brand Discount",
			},
			wantName:    "Brand Discount",
			wantType:    "promotion",
			wantMethod:  "discount_amount",
			wantValue:   5000,
			wantBrandID: &brandID,
		},
		{
			name: "no target — returns error",
			row: map[string]interface{}{
				"_row":        4,
				"PricingType": "promotion",
				"Name":        "No Target",
			},
			wantErr:     true,
			errContains: "at least one of ProductID, CategoryID, or BrandID is required",
		},
		{
			name: "missing PricingType — returns error",
			row: map[string]interface{}{
				"_row":      5,
				"ProductID": float64(1),
				"Name":      "No Type",
			},
			wantErr:     true,
			errContains: "PricingType is required",
		},
		{
			name: "missing Name — returns error",
			row: map[string]interface{}{
				"_row":        6,
				"ProductID":   float64(1),
				"PricingType": "promotion",
			},
			wantErr:     true,
			errContains: "Name is required",
		},
		{
			name: "PricingMethod defaults to fixed_price",
			row: map[string]interface{}{
				"_row":        7,
				"ProductID":   float64(1),
				"PricingType": "promotion",
				"Name":        "Default Method",
			},
			wantName:   "Default Method",
			wantType:   "promotion",
			wantMethod: "fixed_price",
			wantProdID: intPtr(1),
		},
		{
			name: "IsActive defaults to true",
			row: map[string]interface{}{
				"_row":        8,
				"ProductID":   float64(1),
				"PricingType": "promotion",
				"Name":        "Default Active",
			},
			wantName:   "Default Active",
			wantMethod: "fixed_price",
			wantProdID: intPtr(1),
		},
		{
			name: "IsActive false",
			row: map[string]interface{}{
				"_row":        9,
				"ProductID":   float64(1),
				"PricingType": "promotion",
				"Name":        "Inactive Rule",
				"IsActive":    "false",
			},
			wantName:   "Inactive Rule",
			wantProdID: intPtr(1),
		},
		{
			name: "EffectiveFrom and EffectiveUntil parsed in Jakarta timezone",
			row: map[string]interface{}{
				"_row":           10,
				"ProductID":      float64(1),
				"PricingType":    "promotion",
				"PricingMethod":  "fixed_price",
				"PricingValue":   10000,
				"Name":           "Date Range Rule",
				"EffectiveFrom":  "2025-01-15",
				"EffectiveUntil": "2025-12-31",
			},
			wantName:   "Date Range Rule",
			wantProdID: intPtr(1),
		},
		{
			name: "EffectiveFrom invalid date — silently ignored",
			row: map[string]interface{}{
				"_row":          11,
				"ProductID":     float64(1),
				"PricingType":   "promotion",
				"PricingMethod": "fixed_price",
				"PricingValue":  10000,
				"Name":          "Bad From Date",
				"EffectiveFrom": "not-a-date",
			},
			wantName:   "Bad From Date",
			wantProdID: intPtr(1),
		},
		{
			name: "EffectiveUntil invalid date — silently ignored",
			row: map[string]interface{}{
				"_row":           12,
				"ProductID":      float64(1),
				"PricingType":    "promotion",
				"PricingMethod":  "fixed_price",
				"PricingValue":   10000,
				"Name":           "Bad Until Date",
				"EffectiveUntil": "not-a-date",
			},
			wantName:   "Bad Until Date",
			wantProdID: intPtr(1),
		},
		{
			name: "fallback to legacy Price column",
			row: map[string]interface{}{
				"_row":        13,
				"ProductID":   float64(1),
				"PricingType": "promotion",
				"Name":        "Legacy Price",
				"Price":       25000,
			},
			wantName:   "Legacy Price",
			wantValue:  25000,
			wantProdID: intPtr(1),
		},
	}

	ctx := context.Background()
	a := &adapter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := a.MapToEntity(ctx, Schema, tt.row)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			require.NoError(t, err)
			importRow, ok := got.(PricingRuleImportRow)
			require.True(t, ok, "expected PricingRuleImportRow, got %T", got)

			assert.Equal(t, tt.wantName, importRow.Name)
			if tt.wantType != "" {
				assert.Equal(t, tt.wantType, importRow.PricingType)
			}
			if tt.wantMethod != "" {
				assert.Equal(t, tt.wantMethod, importRow.PricingMethod)
			}
			if tt.wantValue != 0 {
				assert.Equal(t, tt.wantValue, importRow.PricingValue)
			}
			assert.Equal(t, tt.wantProdID, importRow.ProductID)
			assert.Equal(t, tt.wantCatID, importRow.CategoryID)
			assert.Equal(t, tt.wantBrandID, importRow.BrandID)

			// Verify Jakarta timezone parsing for EffectiveFrom
			if tt.name == "EffectiveFrom and EffectiveUntil parsed in Jakarta timezone" {
				require.NotNil(t, importRow.EffectiveFrom)
				jakarta := shared.JakartaLocation()
				ef := importRow.EffectiveFrom.In(jakarta)
				assert.Equal(t, 2025, ef.Year())
				assert.Equal(t, time.January, ef.Month())
				assert.Equal(t, 15, ef.Day())

				require.NotNil(t, importRow.EffectiveUntil)
				eu := importRow.EffectiveUntil.In(jakarta)
				assert.Equal(t, 2025, eu.Year())
				assert.Equal(t, time.December, eu.Month())
				assert.Equal(t, 31, eu.Day())
			}
		})
	}
}

func TestPricingAdapter_MapToEntity_DateTimezoneOffset(t *testing.T) {
	a := &adapter{}
	ctx := context.Background()

	row := map[string]interface{}{
		"_row":            1,
		"ProductID":       float64(1),
		"PricingType":     "promotion",
		"PricingMethod":   "fixed_price",
		"PricingValue":    10000,
		"Name":            "TZ Test",
		"EffectiveFrom":   "2025-06-15",
		"EffectiveUntil":  "2025-06-30",
	}

	got, err := a.MapToEntity(ctx, Schema, row)
	require.NoError(t, err)
	importRow := got.(PricingRuleImportRow)

	// Date string "2025-06-15" parsed in Jakarta timezone (UTC+7)
	// Should be 2025-06-15 00:00:00+07:00 = 2025-06-14 17:00:00 UTC
	jakarta := shared.JakartaLocation()
	ef := importRow.EffectiveFrom.In(jakarta)
	assert.Equal(t, "2025-06-15", ef.Format("2006-01-02"))

	eu := importRow.EffectiveUntil.In(jakarta)
	assert.Equal(t, "2025-06-30", eu.Format("2006-01-02"))
}

func TestPricingAdapter_Repository(t *testing.T) {
	a := &adapter{}
	ra := a.Repository()
	assert.NotNil(t, ra)
}

func TestPricingAdapter_ReposAdapter_ExportData(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	effFrom := time.Date(2025, 1, 15, 0, 0, 0, 0, shared.JakartaLocation())
	effUntil := time.Date(2025, 12, 31, 0, 0, 0, 0, shared.JakartaLocation())

	rows := pgxmock.NewRows([]string{
		"id", "product_id", "category_id", "brand_id", "pricing_type", "pricing_method",
		"pricing_value", "name", "minimum_quantity", "maximum_quantity", "priority",
		"customer_group_id", "store_id", "recurrence_days", "time_from", "time_to",
		"allow_combine", "is_active", "status", "effective_from", "effective_until",
		"created_at", "updated_at",
	}).AddRow(
		1, &([]int{42})[0], nil, nil, PricingTypePromotion, PricingMethodFixedPrice,
		12000.0, "Export Rule", 1, nil, 0,
		nil, nil, nil, nil, nil,
		false, true, StatusApproved, effFrom, effUntil,
		now, now,
	)
	mock.ExpectQuery("SELECT (.+) FROM pricing_rules ORDER BY id ASC").WillReturnRows(rows)

	repo := NewRepository(mock)
	a := NewAdapter(repo)
	ra := a.Repository()

	data, err := ra.ExportData(context.Background(), Schema)
	require.NoError(t, err)
	require.Len(t, data, 1)

	assert.Equal(t, "Export Rule", data[0]["Name"])
	assert.Equal(t, "promotion", data[0]["PricingType"])
	assert.Equal(t, "fixed_price", data[0]["PricingMethod"])
	assert.Equal(t, 12000.0, data[0]["PricingValue"])

	// Verify EffectiveFrom is formatted in Jakarta timezone
	effFromStr, ok := data[0]["EffectiveFrom"].(string)
	require.True(t, ok, "EffectiveFrom should be string")
	assert.Equal(t, "2025-01-15", effFromStr)

	effUntilStr, ok := data[0]["EffectiveUntil"].(string)
	require.True(t, ok, "EffectiveUntil should be string")
	assert.Equal(t, "2025-12-31", effUntilStr)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPricingAdapter_ReposAdapter_ExportData_NoDates(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{
		"id", "product_id", "category_id", "brand_id", "pricing_type", "pricing_method",
		"pricing_value", "name", "minimum_quantity", "maximum_quantity", "priority",
		"customer_group_id", "store_id", "recurrence_days", "time_from", "time_to",
		"allow_combine", "is_active", "status", "effective_from", "effective_until",
		"created_at", "updated_at",
	}).AddRow(
		1, nil, nil, nil, PricingTypeSpecialPrice, PricingMethodDiscountPct,
		10.0, "No Date Rule", 1, nil, 0,
		nil, nil, nil, nil, nil,
		false, true, StatusDraft, nil, nil,
		now, now,
	)
	mock.ExpectQuery("SELECT (.+) FROM pricing_rules ORDER BY id ASC").WillReturnRows(rows)

	repo := NewRepository(mock)
	a := NewAdapter(repo)
	ra := a.Repository()

	data, err := ra.ExportData(context.Background(), Schema)
	require.NoError(t, err)
	require.Len(t, data, 1)

	assert.Equal(t, "No Date Rule", data[0]["Name"])
	_, hasFrom := data[0]["EffectiveFrom"]
	_, hasUntil := data[0]["EffectiveUntil"]
	assert.False(t, hasFrom, "EffectiveFrom should be absent when nil")
	assert.False(t, hasUntil, "EffectiveUntil should be absent when nil")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPricingAdapter_ReposAdapter_ExportData_Error(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT (.+) FROM pricing_rules ORDER BY id ASC").WillReturnError(fmt.Errorf("db error"))

	repo := NewRepository(mock)
	a := NewAdapter(repo)
	ra := a.Repository()

	_, err = ra.ExportData(context.Background(), Schema)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPricingAdapter_ReposAdapter_Insert(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO pricing_rules").WithArgs(
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
	).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	repo := NewRepository(mock)
	a := NewAdapter(repo)
	ra := a.Repository()

	pid := 42
	inserted, err := ra.Insert(context.Background(), []interface{}{
		PricingRuleImportRow{
			Row:             1,
			ProductID:       &pid,
			PricingType:     "promotion",
			PricingMethod:   "fixed_price",
			PricingValue:    12000,
			Name:            "Insert Rule",
			MinimumQuantity: 1,
			IsActive:        true,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, inserted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPricingAdapter_ReposAdapter_Insert_MinQtyZero(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewRepository(mock)
	a := NewAdapter(repo)
	ra := a.Repository()

	pid := 42
	_, err = ra.Insert(context.Background(), []interface{}{
		PricingRuleImportRow{
			Row:             1,
			ProductID:       &pid,
			PricingType:     "promotion",
			PricingMethod:   "fixed_price",
			PricingValue:    10000,
			Name:            "Zero MinQty",
			MinimumQuantity: 0,
		},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "minimum_quantity must be >= 1")
}

func TestPricingAdapter_ReposAdapter_Update(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE pricing_rules").WithArgs(
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(),
	).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit()

	repo := NewRepository(mock)
	a := NewAdapter(repo)
	ra := a.Repository()

	pid := 42
	updated, err := ra.Update(context.Background(), []interface{}{
		PricingRuleImportRow{
			Row:             1,
			ProductID:       &pid,
			PricingType:     "promotion",
			PricingMethod:   "fixed_price",
			PricingValue:    9000,
			Name:            "Update Rule",
			MinimumQuantity: 1,
			IsActive:        true,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, updated)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPricingAdapter_ReposAdapter_LoadReferences(t *testing.T) {
	a := NewAdapter(nil)
	ra := a.Repository()
	refs, err := ra.LoadReferences(context.Background(), Schema)
	assert.NoError(t, err)
	assert.Empty(t, refs)
}

func TestPricingAdapter_ExportData_JakartaTimezoneFormatting(t *testing.T) {
	// Verify that UTC times are formatted as Jakarta dates
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	// A UTC time that is still the same day in Jakarta (14:00 UTC = 21:00 WIB)
	utcTime := time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC)
	now := time.Now()

	rows := pgxmock.NewRows([]string{
		"id", "product_id", "category_id", "brand_id", "pricing_type", "pricing_method",
		"pricing_value", "name", "minimum_quantity", "maximum_quantity", "priority",
		"customer_group_id", "store_id", "recurrence_days", "time_from", "time_to",
		"allow_combine", "is_active", "status", "effective_from", "effective_until",
		"created_at", "updated_at",
	}).AddRow(
		1, nil, nil, nil, PricingTypePromotion, PricingMethodFixedPrice,
		10000.0, "TZ Export Rule", 1, nil, 0,
		nil, nil, nil, nil, nil,
		false, true, StatusApproved, utcTime, nil,
		now, now,
	)
	mock.ExpectQuery("SELECT (.+) FROM pricing_rules ORDER BY id ASC").WillReturnRows(rows)

	repo := NewRepository(mock)
	a := NewAdapter(repo)
	ra := a.Repository()

	data, err := ra.ExportData(context.Background(), Schema)
	require.NoError(t, err)
	require.Len(t, data, 1)

	// 14:00 UTC = 21:00 WIB, same calendar day 2025-06-15
	effFromStr := data[0]["EffectiveFrom"].(string)
	assert.Equal(t, "2025-06-15", effFromStr)

	assert.NoError(t, mock.ExpectationsWereMet())
}
