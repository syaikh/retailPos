package product

import (
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/ownership"
	"retail-pos-system/internal/permissions"

	"github.com/gin-gonic/gin"
)

// canViewCost reports whether the caller may read sensitive cost data,
// governed by the product.cost.view permission (see
// docs/audits/permission-additions-sprint1.md).
func canViewCost(c *gin.Context) bool {
	return ownership.CanAccessAll(middleware.GetPermissions(c), permissions.ProductCostView)
}

// presentProduct returns the wire representation of p for a caller. Cost is
// sensitive (product.cost.view); when the caller lacks the permission the
// field is omitted instead of null so consumers cannot distinguish a missing
// cost from a zero cost.
func presentProduct(p Product, canViewCost bool) any {
	if canViewCost {
		return p
	}
	return productWithoutCost{Product: p}
}

// productWithoutCost embeds Product and shadows the promoted Cost field with
// a nil + omitempty field so encoding/json omits `cost` from the payload.
type productWithoutCost struct {
	Product
	Cost *int `json:"cost,omitempty"`
}
