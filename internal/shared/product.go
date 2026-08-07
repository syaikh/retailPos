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
