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

// PrintRequest is the JSON body accepted at POST /print. It is compatible with
// the current frontend (print-service.ts sends { invoice, data, branding }) and
// the design-doc shape ({ job_id, type, receipt, branding }). The agent uses
// whichever of "data" / "receipt" is present.
type PrintRequest struct {
	JobID    string   `json:"job_id"`
	Type     string   `json:"type"`
	Copies   int      `json:"copies"`
	Data     Receipt  `json:"data"`
	Receipt  Receipt  `json:"receipt"`
	Branding Branding `json:"branding"`
}
