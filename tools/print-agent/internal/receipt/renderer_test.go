package receipt

import (
	"strings"
	"testing"
)

func sampleReceipt() Receipt {
	return Receipt{
		InvoiceNumber: "INV-1",
		CreatedAt:     "2026-08-26 10:00",
		Items: []Item{
			{Name: "Kopi", Quantity: 2, UnitPrice: 5000, OriginalPrice: 6000, PricingRuleName: "Promo"},
		},
		TotalAmount:  10000,
		SubtotalDPP:  9009,
		Tax:          991,
		PaymentMethod: "split",
		Payments: []Payment{
			{Method: "cash", Amount: 5000},
			{Method: "qris", Amount: 5000, ReferenceNumber: "QR1"},
		},
		CashReceived: 5000,
		ChangeDue:    0,
		CustomerName: "Jane",
		TotalSavings: 2000,
	}
}

func TestRenderContainsKeySections(t *testing.T) {
	out, err := Render(sampleReceipt(), Branding{StoreName: "Toko Test", ReceiptFooter: "Terima kasih"})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{
		"Toko Test", "INV-1", "Kopi", "DPP", "PPN", "TOTAL",
		"cash", "qris", "QR1", "Terima kasih",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("rendered receipt missing %q\n%s", want, s)
		}
	}
}

func TestRenderStartsWithInitEndsWithCut(t *testing.T) {
	out, err := Render(sampleReceipt(), Branding{})
	if err != nil {
		t.Fatal(err)
	}
	if out[0] != 0x1B || out[1] != '@' {
		t.Fatal("output must start with ESC @")
	}
	// trailing bytes: LineFeed(3) then Cut(3)
	if out[len(out)-3] != 0x1D || out[len(out)-2] != 'V' {
		t.Fatal("output must end with GS V cut")
	}
}

func TestRenderMissingInvoice(t *testing.T) {
	if _, err := Render(Receipt{}, Branding{}); err != ErrMissingInvoice {
		t.Fatal("expected ErrMissingInvoice")
	}
}
