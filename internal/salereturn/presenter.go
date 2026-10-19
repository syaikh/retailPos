package salereturn

import "time"

// presentReturn shapes a return for the API (PRD §9 responses). Status is the
// EFFECTIVE status: a pending return whose approval window elapsed without a
// decision displays as 'expired' even though no sweeper has flipped the rows
// yet (lazy expiry, PRD §12.3 #16).
func presentReturn(ret *Return) map[string]interface{} {
	if ret == nil {
		return nil
	}
	status := ret.Status
	if status == ReturnStatusPendingApproval && ret.ExpiresAt != nil && time.Now().After(*ret.ExpiresAt) {
		status = ReturnStatusExpired
	}

	items := make([]map[string]interface{}, 0, len(ret.Items))
	for _, it := range ret.Items {
		reps := make([]map[string]interface{}, 0, len(it.Replacements))
		for _, rep := range it.Replacements {
			reps = append(reps, map[string]interface{}{
				"id":         rep.ID,
				"product_id": rep.ProductID,
				"quantity":   rep.Quantity,
			})
		}
		items = append(items, map[string]interface{}{
			"id":           it.ID,
			"sale_item_id": it.SaleItemID,
			"product_id":   it.ProductID,
			"product_name": it.ProductName,
			"quantity":     it.Quantity,
			"unit_price":   it.UnitPrice,
			"subtotal":     it.Subtotal,
			"dpp_amount":   it.DppAmount,
			"tax_amount":   it.TaxAmount,
			"tax_rate":     it.TaxRate,
			"condition":    it.Condition,
			"outcome":      it.Outcome,
			"notes":        it.Notes,
			"replacements": reps,
		})
	}

	payments := make([]map[string]interface{}, 0, len(ret.Payments))
	for _, p := range ret.Payments {
		payments = append(payments, map[string]interface{}{
			"id":                  p.ID,
			"payment_method_id":   p.PaymentMethodID,
			"payment_method_code": p.PaymentMethodCode,
			"amount":              p.Amount,
		})
	}

	out := map[string]interface{}{
		"id":                  ret.ID,
		"return_number":       ret.ReturnNumber,
		"original_sale_id":    ret.OriginalSaleID,
		"cashier_id":          ret.CashierID,
		"store_id":            ret.StoreID,
		"customer_id":         ret.CustomerID,
		"shift_id":            ret.ShiftID,
		"return_type":         ret.ReturnType,
		"reason":              ret.Reason,
		"condition":           ret.Condition,
		"notes":               ret.Notes,
		"subtotal":            ret.Subtotal,
		"discount":            ret.Discount,
		"restocking_fee":      ret.RestockingFee,
		"tax_reversal":        ret.TaxReversal,
		"total_refund_amount": ret.TotalRefundAmount,
		"status":              status,
		"change_due":          ret.ChangeDue,
		"created_at":          ret.CreatedAt,
		"updated_at":          ret.UpdatedAt,
		"expires_at":          ret.ExpiresAt,
		"items":               items,
		"payments":            payments,
	}

	if a := ret.Approval; a != nil {
		approvalStatus := a.Status
		if approvalStatus == "pending" && time.Now().After(a.ExpiresAt) {
			approvalStatus = "expired"
		}
		out["approval"] = map[string]interface{}{
			"id":              a.ID,
			"requested_by":    a.RequestedBy,
			"approved_by":     a.ApprovedBy,
			"status":          approvalStatus,
			"reason":          a.Reason,
			"rejected_reason": a.RejectedReason,
			"expires_at":      a.ExpiresAt,
		}
	} else {
		out["approval"] = nil
	}
	return out
}

func presentReturns(returns []Return) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(returns))
	for i := range returns {
		out = append(out, presentReturn(&returns[i]))
	}
	return out
}
