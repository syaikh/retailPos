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
		TotalAmount:   10000,
		SubtotalDPP:   9009,
		Tax:           991,
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

func sampleReturnRefund() ReturnReceipt {
	return ReturnReceipt{
		InvoiceNumber:         "RETUR-0001",
		CreatedAt:             "2026-08-26 11:00",
		ReturnType:            "refund",
		OriginalInvoiceNumber: "INV-1",
		RefundAmount:          10000,
		Payments:              []Payment{{Method: "CASH", Amount: 10000}},
		Reason:                "Barang rusak",
		Condition:             "Rusak",
		Items: []Item{
			{Name: "Kopi", Quantity: 2, UnitPrice: 5000},
		},
	}
}

func TestRenderReturnContainsKeySections(t *testing.T) {
	out, err := RenderReturn(sampleReturnRefund(), Branding{StoreName: "Toko Test", ReceiptFooter: "Terima kasih"})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{
		"Toko Test", "RETUR", "No retur: RETUR-0001", "Inv asal: INV-1",
		"Kopi", "Alasan: Barang rusak", "Kondisi: Rusak",
		"Total dikembalikan", "Rp 10.000", "CASH", "Terima kasih",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("rendered return receipt missing %q\n%s", want, s)
		}
	}
	// A refund is money out: no replacement section.
	if strings.Contains(s, "Penggantian") {
		t.Fatalf("refund receipt must not contain the replacement section\n%s", s)
	}
}

func TestRenderReturnReplacementShowsGoodsNotMoney(t *testing.T) {
	r := sampleReturnRefund()
	r.ReturnType = "replacement"
	r.RefundAmount = 0
	r.Payments = nil
	r.Replacements = []ReplacementItem{{Name: "Kopi", Quantity: 2}}
	out, err := RenderReturn(r, Branding{})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{"Penggantian", "Kopi x2"} {
		if !strings.Contains(s, want) {
			t.Fatalf("replacement return receipt missing %q\n%s", want, s)
		}
	}
	// A replacement hands over goods: no money line at all.
	for _, bad := range []string{"Total dikembalikan", "CASH"} {
		if strings.Contains(s, bad) {
			t.Fatalf("replacement receipt must not contain %q\n%s", bad, s)
		}
	}
}

func TestRenderReturnNotesOptional(t *testing.T) {
	r := sampleReturnRefund()
	r.Notes = "catatan pelanggan"
	out, err := RenderReturn(r, Branding{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "Catatan: catatan pelanggan") {
		t.Fatal("return receipt must carry notes when present")
	}
	r.Notes = ""
	out, _ = RenderReturn(r, Branding{})
	if strings.Contains(string(out), "Catatan:") {
		t.Fatal("return receipt must omit the notes line when empty")
	}
}

func TestRenderReturnMissingInvoice(t *testing.T) {
	if _, err := RenderReturn(ReturnReceipt{}, Branding{}); err != ErrMissingInvoice {
		t.Fatal("expected ErrMissingInvoice")
	}
}

func TestRenderReturnStartsWithInitEndsWithCut(t *testing.T) {
	out, err := RenderReturn(sampleReturnRefund(), Branding{})
	if err != nil {
		t.Fatal(err)
	}
	if out[0] != 0x1B || out[1] != '@' {
		t.Fatal("output must start with ESC @")
	}
	if out[len(out)-3] != 0x1D || out[len(out)-2] != 'V' {
		t.Fatal("output must end with GS V cut")
	}
}
