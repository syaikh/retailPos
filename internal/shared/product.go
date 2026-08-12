package shared

// ProductMeta is the sku/name identity of a product, keyed by product ID in
// provider lookups. It is the cross-module contract between internal/product
// (single-writer of the products table, Katalog) and consumers such as
// internal/inventory (transaksional) that enrich stock listings with product
// display data without importing internal/product.
type ProductMeta struct {
	SKU     string `json:"sku"`
	Name    string `json:"name"`
	StoreID *int   `json:"store_id,omitempty"`
}

// SnapshotProduct is the catalog half of the stock opname snapshot read-model:
// product identity plus unit-of-measure id, WITHOUT stock quantities (those
// live in product_stock, owned by internal/inventory). internal/product is the
// single-writer of products and provides the rows; the consumer
// (internal/stockopname) merges them with the inventory-owned quantities.
type SnapshotProduct struct {
	ProductID int
	Name      string
	SKU       string
	Barcode   string
	UOMID     *int
}

// ProductScope is the category/brand membership of a product, keyed by product
// ID in provider lookups. It is the cross-module contract between
// internal/product (single-writer of products, Katalog) and consumers such as
// internal/pricing (Katalog) that match product-, category-, or brand-scoped
// pricing rules without reading the products table directly.
type ProductScope struct {
	CategoryID *int
	BrandID    *int
}

// ProductCostTax is the cost/tax snapshot of a product at price-resolution
// time. internal/product is the single-writer of products and tax_classes and
// provides the rows; internal/pricing consumes them to build price snapshots
// without a products JOIN across module boundaries.
type ProductCostTax struct {
	Cost        int
	TaxClassID  *int
	TaxRate     *float64
	ProductName string
}

// ProductSearchResult is an autocomplete hit for the pricing product search
// endpoint. internal/product provides the rows; the consumer (internal/pricing)
// returns them as-is to the client without querying products directly.
type ProductSearchResult struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	SKU   string `json:"sku"`
	Price int    `json:"price"`
}
