package cache

import (
	"context"
	"sync"
)

type Action string

const (
	ActionUpdated Action = "UPDATED"
	ActionDeleted Action = "DELETED"
	ActionFlushed Action = "FLUSHED"
)

// Special channel for broadcasting to all subscribers
const BroadcastChannel string = "ALL_ENTITIES"

// InvalidationEvent carries entity changes across caches.
type InvalidationEvent struct {
	EntityName string // e.g., "season", "event"
	EntityID   any    // e.g., int ID of the modified entity
	Action     Action // UPDATED, DELETED, FLUSHED
	Payload    any    // Optional: modified struct or metadata
}

type EventHandler func(ctx context.Context, evt InvalidationEvent)

type CacheManager struct {
	mu          sync.RWMutex
	subscribers map[string][]EventHandler
}

func NewManager() *CacheManager {
	return &CacheManager{
		subscribers: make(map[string][]EventHandler),
	}
}

// Subscribe registers an event handler for a specific entity type (e.g., "season").
// use BroadcastChannel to subscribe to all entity changes.
func (m *CacheManager) Subscribe(entityName string, handler EventHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscribers[entityName] = append(m.subscribers[entityName], handler)
}

// Publish broadcasts an invalidation event to all subscribers of that entity.
func (m *CacheManager) Publish(ctx context.Context, evt InvalidationEvent) {
	m.mu.RLock()
	handlers := append([]EventHandler(nil), m.subscribers[evt.EntityName]...)
	handlers = append(handlers, m.subscribers[BroadcastChannel]...)
	m.mu.RUnlock()

	for _, handler := range handlers {
		handler(ctx, evt)
	}
}
