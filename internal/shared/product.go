package shared

// ProductMeta is the sku/name identity of a product, keyed by product ID in
// provider lookups. It is the cross-module contract between internal/product
// (single-writer of the products table, Katalog) and consumers such as
// internal/inventory (transaksional) that enrich stock listings with product
// display data without importing internal/product.
type ProductMeta struct {
	SKU  string `json:"sku"`
	Name string `json:"name"`
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
