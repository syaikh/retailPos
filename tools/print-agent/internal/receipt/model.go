package receipt

// Branding holds store identity rendered at the top and bottom of a receipt.
// Field names match the JSON keys sent by print-service.ts (ReceiptBranding).
type Branding struct {
	StoreName     string `json:"storeName"`
	StoreAddress  string `json:"storeAddress"`
	StorePhone    string `json:"storePhone"`
	ReceiptHeader string `json:"receiptHeader"`
	ReceiptFooter string `json:"receiptFooter"`
}

// Item is a single receipt line.
type Item struct {
	Name            string `json:"name"`
	Quantity        int64  `json:"quantity"`
	UnitPrice       int64  `json:"unit_price"`
	OriginalPrice   int64  `json:"original_price"`
	PricingRuleName string `json:"pricing_rule_name"`
	PricingType     string `json:"pricing_type"`
}

// Payment is a single tender applied to the sale.
type Payment struct {
	Method          string `json:"method"`
	Amount          int64  `json:"amount"`
	ReferenceNumber string `json:"reference_number"`
}

// Receipt is the structured receipt payload. It mirrors the Svelte ReceiptData
// contract used by the POS (see web/src/shared/stores/printReceipt.svelte.ts).
type Receipt struct {
	InvoiceNumber string    `json:"invoice_number"`
	CreatedAt     string    `json:"created_at"`
	Items         []Item    `json:"items"`
	TotalAmount   int64     `json:"total_amount"`
	SubtotalDPP   int64     `json:"subtotal_dpp"`
	Tax           int64     `json:"tax"`
	PaymentMethod string    `json:"paymentMethod"`
	Payments      []Payment `json:"payments"`
	CashReceived  int64     `json:"cashReceived"`
	ChangeDue     int64     `json:"changeDue"`
	CustomerName  string    `json:"customer_name"`
	TotalSavings  int64     `json:"total_savings"`
}

// ReplacementItem is one same-product unit handed back to the customer on a
// replacement (the name lives on the returned line, not on the record).
type ReplacementItem struct {
	Name     string `json:"name"`
	Quantity int64  `json:"quantity"`
}

// ReturnReceipt mirrors the Svelte ReturnReceiptData contract (Phase 4.4,
// web/src/shared/stores/printReceipt.svelte.ts): a processed sales return.
// InvoiceNumber carries the return's own document number; the sale it undoes
// is referenced by OriginalInvoiceNumber. All numbers are precomputed display
// data, exactly like Receipt.
type ReturnReceipt struct {
	InvoiceNumber         string            `json:"invoice_number"`
	CreatedAt             string            `json:"created_at"`
	ReturnType            string            `json:"return_type"` // "replacement" | "refund"
	OriginalInvoiceNumber string            `json:"original_invoice_number"`
	RefundAmount          int64             `json:"refund_amount"`
	Payments              []Payment         `json:"payments"`
	Reason                string            `json:"reason"`
	Condition             string            `json:"condition"`
	Notes                 string            `json:"notes"`
	Items                 []Item            `json:"items"`
	Replacements          []ReplacementItem `json:"replacements"`
}

// PrintRequest is the JSON body accepted at POST /print. It is compatible with
// the current frontend (print-service.ts sends { invoice, type, data |
// return_data, branding }) and the design-doc shape ({ job_id, type, receipt,
// branding }). The agent uses whichever of "data" / "receipt" is present; a
// request with type "return" decodes ReturnData instead.
//
// A return payload travels under its own "return_data" key rather than reusing
// "data" because Go decodes JSON into concrete structs: unknown fields are
// silently dropped, so an agent that did not know the shape would print a wrong
// receipt. With the separate key an old agent finds no payload and rejects the
// job instead.
type PrintRequest struct {
	JobID      string        `json:"job_id"`
	Type       string        `json:"type"`
	Copies     int           `json:"copies"`
	Data       Receipt       `json:"data"`
	Receipt    Receipt       `json:"receipt"`
	ReturnData ReturnReceipt `json:"return_data"`
	Branding   Branding      `json:"branding"`
}
