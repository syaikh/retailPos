package supplier

import (
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/ownership"
	"retail-pos-system/internal/permissions"

	"github.com/gin-gonic/gin"
)

// canViewUnitCost reports whether the caller may read the purchase cost of a
// product-supplier link. It uses the same product.cost.view permission that
// internal/product gates a product's own cost on, because the number is the same
// number: a link's unit_cost is a cost of goods figure, and supplier.view says
// nothing about costing.
func canViewUnitCost(c *gin.Context) bool {
	return ownership.CanAccessAll(middleware.GetPermissions(c), permissions.ProductCostView)
}

// presentProductSupplier returns the wire representation of a single link for a
// caller. unit_cost is omitted rather than nulled when the caller lacks
// product.cost.view, so a consumer cannot tell a hidden cost from a zero one.
func presentProductSupplier(ps ProductSupplier, canViewUnitCost bool) any {
	if canViewUnitCost {
		return ps
	}
	return productSupplierWithoutCost{ProductSupplier: ps}
}

// presentProductSuppliers applies presentProductSupplier across a list.
func presentProductSuppliers(list []ProductSupplier, canViewUnitCost bool) []any {
	out := make([]any, 0, len(list))
	for _, ps := range list {
		out = append(out, presentProductSupplier(ps, canViewUnitCost))
	}
	return out
}

// productSupplierWithoutCost embeds ProductSupplier and shadows the promoted
// UnitCost field with a nil + omitempty field so encoding/json drops
// `unit_cost` from the payload. This mirrors productWithoutCost in
// internal/product.
type productSupplierWithoutCost struct {
	ProductSupplier
	UnitCost *int `json:"unit_cost,omitempty"`
}
