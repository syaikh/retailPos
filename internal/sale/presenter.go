package sale

import (
	"github.com/gin-gonic/gin"

	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/ownership"
	"retail-pos-system/internal/permissions"
)

// canViewCost reports whether the caller may read sensitive cost data,
// governed by the product.cost.view permission (see
// docs/audits/sensitive-data-audit.md).
func canViewCost(c *gin.Context) bool {
	return ownership.CanAccessAll(middleware.GetPermissions(c), permissions.ProductCostView)
}

// presentSale returns the wire representation of s for a caller. Cost on items
// is sensitive (product.cost.view); when the caller lacks the permission the
// field is omitted instead of null so consumers cannot distinguish a missing
// cost from a zero cost.
func presentSale(s *Sale, canViewCost bool) any {
	if canViewCost {
		return s
	}
	ps := saleWithoutCost{Sale: *s}
	if s.Items != nil {
		ps.Items = make([]saleItemWithoutCost, 0, len(s.Items))
		for _, it := range s.Items {
			ps.Items = append(ps.Items, saleItemWithoutCost{Item: it})
		}
	}
	return ps
}

// presentCart returns the wire representation of a cart for a caller,
// omitting item cost for non-holders of product.cost.view.
func presentCart(cart *CartSession, canViewCost bool) any {
	if canViewCost {
		return cart
	}
	pc := cartSessionWithoutCost{CartSession: *cart}
	if cart.Items != nil {
		pc.Items = make([]cartItemWithoutCost, 0, len(cart.Items))
		for _, it := range cart.Items {
			pc.Items = append(pc.Items, cartItemWithoutCost{CartItem: it})
		}
	}
	return pc
}

// presentCarts returns the wire representation of a cart list for a caller.
func presentCarts(carts []CartSession, canViewCost bool) any {
	if canViewCost {
		return carts
	}
	presented := make([]any, 0, len(carts))
	for _, c := range carts {
		presented = append(presented, presentCart(&c, false))
	}
	return presented
}

// saleWithoutCost embeds Sale and shadows the promoted Items field with items
// that carry no cost.
type saleWithoutCost struct {
	Sale
	Items []saleItemWithoutCost `json:"items,omitempty"`
}

// saleItemWithoutCost embeds Item and shadows the promoted Cost field with
// a nil + omitempty field so encoding/json omits `cost` from the payload.
type saleItemWithoutCost struct {
	Item
	Cost *int `json:"cost,omitempty"`
}

// cartSessionWithoutCost embeds CartSession and shadows the promoted Items
// field with items that carry no cost.
type cartSessionWithoutCost struct {
	CartSession
	Items []cartItemWithoutCost `json:"items,omitempty"`
}

// cartItemWithoutCost embeds CartItem and shadows the promoted Cost field with
// a nil + omitempty field so encoding/json omits `cost` from the payload.
type cartItemWithoutCost struct {
	CartItem
	Cost *int `json:"cost,omitempty"`
}
