package eventbus

import (
	"context"
	"time"
)

type UpdatePayload struct {
	Old interface{}
	New interface{}
}

type EventType string

const (
	SaleCreated    EventType = "sale.created"
	ProductUpdated EventType = "product.updated"
	StockAdjusted  EventType = "stock.adjusted"
	LowStock       EventType = "stock.low"
	UserLoggedIn   EventType = "user.logged_in"
	EntityCreated  EventType = "entity.created"
	EntityUpdated  EventType = "entity.updated"
	EntityDeleted  EventType = "entity.deleted"
)

// Event bersifat IMMUTABLE setelah dipublikasikan.
// PERINGATAN: Jangan pernah mengubah (mutate) data di dalam Payload atau Metadata
// di dalam komponen Listener, karena objek ini diakses secara konkuren oleh banyak goroutine.
// Proteksi: Read-Only enforcement + go test -race.
type Event struct {
	Type      EventType
	Payload   interface{}     // Read-Only! Jangan di-cast untuk mengubah field.
	Ctx       context.Context // Konteks asli HTTP request (trace ID, user info)
	Timestamp time.Time
	Metadata  map[string]interface{} // Read-Only!
}
