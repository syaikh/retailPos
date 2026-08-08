package events

const TopicCartCheckedOut = "cart.checked_out.v1"

// CartCheckedOut is the cross-module payload published after a cart session is
// converted into a sale. Consumers must rely on this DTO instead of importing
// the sale module.
type CartCheckedOut struct {
	CartID    int `json:"cart_id"`
	SaleID    int `json:"sale_id"`
	CashierID int `json:"cashier_id"`
}
