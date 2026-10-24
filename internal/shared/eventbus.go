package shared

import "context"

// EventBus is the common interface for publishing domain events.
// Consolidates the duplicate definitions in product, sale, inventory, and report services.
type EventBus interface {
	Publish(ctx context.Context, topic string, event interface{}) error
	// PublishCritical publishes an event whose loss would corrupt business
	// state (e.g. purchase_receipt.completed, which drives stock deductions).
	// Unlike Publish, it never silently drops: on a full channel it blocks
	// briefly, then persists the payload to the dead-letter store for replay,
	// returning ErrChannelFull in that persisted-but-not-delivered case.
	PublishCritical(ctx context.Context, topic string, event interface{}) error
}
