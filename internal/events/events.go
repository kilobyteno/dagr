// Package events provides cross-instance messaging via Redis pub/sub.
package events

import "context"

// Publisher publishes domain events to other Dagr instances.
type Publisher interface {
	Publish(ctx context.Context, topic string, payload []byte) error
}

// Subscriber receives domain events.
type Subscriber interface {
	Subscribe(ctx context.Context, topic string, handler func(payload []byte)) error
}

// Bus combines publish and subscribe for Redis-backed event fan-out.
type Bus interface {
	Publisher
	Subscriber
}

// NopBus is a no-op Bus used until Redis wiring is complete.
type NopBus struct{}

func (NopBus) Publish(context.Context, string, []byte) error { return nil }

func (NopBus) Subscribe(context.Context, string, func([]byte)) error { return nil }
