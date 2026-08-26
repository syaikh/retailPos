package receipt

import (
	"errors"
	"strconv"
	"strings"

	"print-agent/internal/escpos"
)

// ErrMissingInvoice is returned when a receipt has no invoice number.
var ErrMissingInvoice = errors.New("receipt missing invoice_number")

const divider = "--------------------------------"

// Render builds the ESC/POS byte stream for a 58mm receipt. The agent must treat
// every monetary field as precomputed display data and must never recompute tax,
// DPP, totals, or change. Indonesian PPN (11%) is carried by SubtotalDPP + Tax;
// split tenders are carried by Payments.
func Render(r Receipt, b Branding) ([]byte, error) {
	if r.InvoiceNumber == "" {
		return nil, ErrMissingInvoice
	}

	e := escpos.NewBuilder()
	e.Write(escpos.Init())
	e.Write(escpos.CodepageCP858())

	// --- Header (centered, store name emphasized) ---
	e.Write(escpos.Align(1))
	e.Write(escpos.Bold(true))
	e.Write(escpos.FontSize(1, 1))
	e.Write(escpos.Text(coalesce(b.StoreName, "RetailPOS")))
	e.Write(escpos.FontSize(0, 0))
	e.Write(escpos.Bold(false))
	if b.StoreAddress != "" {
		e.Write(escpos.Text(b.StoreAddress))
	}
	if b.StorePhone != "" {
		e.Write(escpos.Text("Telp: " + b.StorePhone))
	}
	if b.ReceiptHeader != "" {
		e.Write(escpos.Text(b.ReceiptHeader))
	}
	e.Write(escpos.Text(""))

	// --- Meta (left) ---
	e.Write(escpos.Align(0))
	e.Write(escpos.Text("Inv: " + r.InvoiceNumber))
	e.Write(escpos.Text("Waktu: " + r.CreatedAt))
	if r.CustomerName != "" {
		e.Write(escpos.Text("Cust: " + r.CustomerName))
	}
	e.Write(escpos.Text(divider))

	// --- Items ---
	for _, it := range r.Items {
		label := it.Name
		if it.Quantity > 1 {
			label = it.Name + " x" + strconv.FormatInt(it.Quantity, 10)
		}
		e.Write(escpos.Text(label))
		e.Write(escpos.Text("  " + formatRupiah(it.UnitPrice*it.Quantity)))
		if it.OriginalPrice > it.UnitPrice && it.OriginalPrice > 0 {
			saved := (it.OriginalPrice - it.UnitPrice) * it.Quantity
			e.Write(escpos.Text("  (-" + formatRupiah(saved) + " " + it.PricingRuleName + ")"))
		}
	}
	e.Write(escpos.Text(divider))

	// --- Tax / totals ---
	if r.Tax > 0 && r.SubtotalDPP > 0 {
		e.Write(escpos.Text("DPP  " + formatRupiah(r.SubtotalDPP)))
		e.Write(escpos.Text("PPN  " + formatRupiah(r.Tax)))
	}
	e.Write(escpos.Bold(true))
	e.Write(escpos.Text("TOTAL " + formatRupiah(r.TotalAmount)))
	e.Write(escpos.Bold(false))
	e.Write(escpos.Text(divider))

	// --- Payments ---
	payments := r.Payments
	if len(payments) == 0 {
		payments = []Payment{{Method: r.PaymentMethod, Amount: r.TotalAmount}}
	}
	for _, p := range payments {
		ref := ""
		if p.ReferenceNumber != "" {
			ref = " (" + p.ReferenceNumber + ")"
		}
		e.Write(escpos.Text(p.Method + ref + "  " + formatRupiah(p.Amount)))
	}
	if r.ChangeDue > 0 {
		e.Write(escpos.Text("Kembali " + formatRupiah(r.ChangeDue)))
	}
	e.Write(escpos.Text(divider))

	// --- Footer (centered) ---
	e.Write(escpos.Align(1))
	footer := b.ReceiptFooter
	if footer == "" {
		footer = "Terima kasih atas kunjungan Anda!"
	}
	for _, line := range strings.Split(footer, "\n") {
		e.Write(escpos.Text(line))
	}
	e.Write(escpos.Text(""))
	e.Write(escpos.LineFeed(3))
	e.Write(escpos.Cut())

	return e.Bytes(), nil
}

func coalesce(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// formatRupiah renders an integer amount of Indonesian Rupiah, e.g. 10000 -> "Rp 10.000".
func formatRupiah(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := groupDigits(int(n))
	if neg {
		return "-Rp " + s
	}
	return "Rp " + s
}

func groupDigits(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []rune(strconv.Itoa(n))
	var out []rune
	count := 0
	for i := len(digits) - 1; i >= 0; i-- {
		if count > 0 && count%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, digits[i])
		count++
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}
