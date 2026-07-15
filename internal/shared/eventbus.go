package shared

import "context"

// EventBus is the common interface for publishing domain events.
// Consolidates the duplicate definitions in product, sale, inventory, and report services.
type EventBus interface {
	Publish(ctx context.Context, topic string, event interface{}) error
}
