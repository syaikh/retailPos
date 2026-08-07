package eventbus

import (
	"context"
	"time"
)

type EventType string

const (
	SaleCreated    EventType = "sale.created"
	CartCheckedOut EventType = "cart.checked_out"
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
