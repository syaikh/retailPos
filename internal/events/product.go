package events

const TopicProductUpdated = "product.updated.v1"

// ProductUpdated is the cross-module payload published after a product is
// updated. Consumers (websocket broadcast) must rely on this DTO instead of
// importing the product module.
type ProductUpdated struct {
	ID      int    `json:"id"`
	SKU     string `json:"sku"`
	Stock   int    `json:"stock"`
	Price   int    `json:"price"`
	StoreID *int   `json:"store_id,omitempty"`
}
