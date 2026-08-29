package appsettings

// BrandingSettings holds the three fields returned by the unauthenticated
// /api/settings/public endpoint and used for immediate branding reflection.
type BrandingSettings struct {
	StoreName   string `json:"store_name"`
	StoreJargon string `json:"store_jargon"`
	LogoPath    string `json:"logo_path"`
}

// AllSettings is the full payload returned by the authenticated GET /api/settings.
// StoreAddress and StorePhone are per-branch values injected from the stores table
// by the handler (not stored in app_settings).
type AllSettings struct {
	BrandingSettings
	ReceiptHeader string `json:"receipt_header"`
	ReceiptFooter string `json:"receipt_footer"`
	StoreAddress  string `json:"store_address,omitempty"`
	StorePhone    string `json:"store_phone,omitempty"`
}
