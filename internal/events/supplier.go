package events

const (
	// TopicSupplierChanged is published after a supplier is deactivated
	// (active -> inactive) or soft-deleted. Reactivation and ordinary field
	// edits do not publish: they do not change the supplier's availability.
	TopicSupplierChanged = "supplier.changed.v1"

	SupplierActionDeactivated = "deactivated"
	SupplierActionDeleted     = "deleted"
)

// SupplierChanged is the cross-module payload published when a supplier leaves
// the selectable set. Suppliers are global (no store scope), so the websocket
// broadcast is not store-filtered; consumers must rely on this DTO instead of
// importing the supplier module.
type SupplierChanged struct {
	SupplierID int    `json:"supplier_id"`
	Name       string `json:"name"`
	Code       string `json:"code"`
	Action     string `json:"action"`
	Version    int    `json:"version"`
}
