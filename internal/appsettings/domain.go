package appsettings

// BrandingSettings holds the three fields returned by the unauthenticated
// /api/settings/public endpoint and used for immediate branding reflection.
type BrandingSettings struct {
	StoreName  string `json:"store_name"`
	StoreJargon string `json:"store_jargon"`
	LogoPath   string `json:"logo_path"`
}

// AllSettings is the full payload returned by the authenticated GET /api/settings.
type AllSettings struct {
	BrandingSettings
	DefaultLanguage string `json:"default_language"`
	ReceiptHeader   string `json:"receipt_header"`
	ReceiptFooter   string `json:"receipt_footer"`
}
