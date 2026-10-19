package sale

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// ReturnSaleReader implements salereturn.SaleReader (structural typing —
// salereturn's port method signature matches, so no import of
// internal/salereturn is needed; internal/wiring registers
// sale.ReturnSaleReader{} into the salereturn service).
//
// The sale is read ON THE CALLER'S TRANSACTION, so the header, lines and
// payments cannot change between the return's validation and its posting —
// the write-then-read race the plan review flagged (TOCTOU) is closed by the
// transaction plus salereturn's per-sale advisory lock.
type ReturnSaleReader struct{}

// SaleForReturn returns nil, nil when no sale with that id exists; the caller
// maps that to its own not-found domain error.
func (ReturnSaleReader) SaleForReturn(ctx context.Context, tx pgx.Tx, saleID int) (*shared.ReturnSale, error) {
	var sale shared.ReturnSale
	err := tx.QueryRow(ctx, `
		SELECT id, invoice_number, cashier_id, customer_id,
		       COALESCE(store_id, 0), COALESCE(status, ''),
		       total_amount, change_due, created_at
		FROM sales
		WHERE id = $1
	`, saleID).Scan(
		&sale.ID, &sale.InvoiceNumber, &sale.CashierID, &sale.CustomerID,
		&sale.StoreID, &sale.Status, &sale.TotalAmount, &sale.ChangeDue,
		&sale.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load sale %d for return: %w", saleID, err)
	}

	itemRows, err := tx.Query(ctx, `
		SELECT id, product_id, COALESCE(product_name, ''), quantity, unit_price, subtotal,
		       COALESCE(dpp_amount, 0), COALESCE(tax_amount, 0), COALESCE(tax_rate, 0)
		FROM sale_items
		WHERE sale_id = $1
		ORDER BY id
	`, saleID)
	if err != nil {
		return nil, fmt.Errorf("failed to load sale %d items for return: %w", saleID, err)
	}
	defer itemRows.Close()
	for itemRows.Next() {
		var item shared.ReturnSaleItem
		if err := itemRows.Scan(&item.ID, &item.ProductID, &item.ProductName,
			&item.Quantity, &item.UnitPrice, &item.Subtotal, &item.DppAmount,
			&item.TaxAmount, &item.TaxRate); err != nil {
			return nil, fmt.Errorf("failed to scan sale %d item for return: %w", saleID, err)
		}
		sale.Items = append(sale.Items, item)
	}
	if err := itemRows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate sale %d items for return: %w", saleID, err)
	}

	payRows, err := tx.Query(ctx, `
		SELECT payment_method_id, payment_method_code, amount
		FROM sale_payments
		WHERE sale_id = $1
		ORDER BY id
	`, saleID)
	if err != nil {
		return nil, fmt.Errorf("failed to load sale %d payments for return: %w", saleID, err)
	}
	defer payRows.Close()
	for payRows.Next() {
		var p shared.ReturnSalePayment
		if err := payRows.Scan(&p.PaymentMethodID, &p.PaymentMethodCode, &p.Amount); err != nil {
			return nil, fmt.Errorf("failed to scan sale %d payment for return: %w", saleID, err)
		}
		sale.Payments = append(sale.Payments, p)
	}
	if err := payRows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate sale %d payments for return: %w", saleID, err)
	}

	return &sale, nil
}
