package eventbus

import "context"

type Listener interface {
	HandleEvent(ctx context.Context, event Event) error
	EventTypes() []EventType
}

type ListenerFunc struct {
	eventTypes []EventType
	handler    func(ctx context.Context, event Event) error
}

func NewListenerFunc(eventTypes []EventType, handler func(ctx context.Context, event Event) error) *ListenerFunc {
	return &ListenerFunc{eventTypes: eventTypes, handler: handler}
}

func (f *ListenerFunc) HandleEvent(ctx context.Context, event Event) error {
	return f.handler(ctx, event)
}

func (f *ListenerFunc) EventTypes() []EventType {
	return f.eventTypes
}
