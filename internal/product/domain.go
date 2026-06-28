package product

type Product struct {
	ID                 int      `json:"id"`
	SKU                string   `json:"sku"`
	Name               string   `json:"name"`
	Barcode            *string  `json:"barcode,omitempty"`
	CategoryID         *int     `json:"category_id,omitempty"`
	CategoryName       *string  `json:"category_name,omitempty"`
	BrandID            *int     `json:"brand_id,omitempty"`
	BrandName          *string  `json:"brand_name,omitempty"`
	Description        *string  `json:"description,omitempty"`
	Price              int      `json:"price"`
	Cost               int      `json:"cost"`
	Stock              int      `json:"stock"`
	StoreID            *int     `json:"store_id,omitempty"`
	Status             string   `json:"status"`
	TaxClassID         *int     `json:"tax_class_id,omitempty"`
	TaxRate            *float64 `json:"tax_rate,omitempty"`
	WeightGrams        *int     `json:"weight_grams,omitempty"`
	UnitOfMeasureID    *int     `json:"unit_of_measure_id,omitempty"`
	UnitOfMeasure      *string  `json:"unit_of_measure,omitempty"`
	DefaultDiscountPct *float64 `json:"default_discount_percent,omitempty"`
	CreatedAt          string   `json:"created_at,omitempty"`
	UpdatedAt          string   `json:"updated_at,omitempty"`
}

type Brand struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at,omitempty"`
}

type TaxClass struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	RatePercent float64 `json:"rate_percent"`
	Description string  `json:"description,omitempty"`
	IsActive    bool    `json:"is_active"`
	CreatedAt   string  `json:"created_at,omitempty"`
}

type UnitOfMeasure struct {
	ID          int    `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at,omitempty"`
}

type BrandImportRow struct {
	Row         int
	Name        string
	Description string
	IsActive    bool
}

type UnitOfMeasureImportRow struct {
	Row         int
	Code        string
	Name        string
	Description string
	IsActive    bool
}

type ProductImportRow struct {
	Row           int
	SKU           string
	Name          string
	Barcode       string
	Category      string
	Brand         string
	Price         int
	Cost          int
	Stock         int
	Status        string
	UnitOfMeasure string
	WeightGrams   int
	Description   string
}

type ProductImportPayload struct {
	SKU             string
	Name            string
	Barcode         *string
	CategoryID      *int
	BrandID         *int
	Price           int
	Cost            int
	Stock           int
	Status          string
	UnitOfMeasureID *int
	WeightGrams     *int
	Description     *string
}

type Warehouse struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	Address   string `json:"address,omitempty"`
	StoreID   *int   `json:"store_id,omitempty"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at,omitempty"`
}
