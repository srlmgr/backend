package cache

import (
	"bytes"
	"container/list"
	"context"
	"encoding/gob"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/srlmgr/backend/log"
)

type EventType int

const (
	EventFlush EventType = iota
	EventUpdate
)

type Event[K comparable, V any] struct {
	Type  EventType
	Key   K
	Value V
}

// entry wraps key, value, and access metadata inside the linked list.
type entry[K comparable, V any] struct {
	key        K
	value      V
	lastAccess time.Time
}

type Option[K comparable, V any] func(*Cache[K, V])

// WithTTL sets max idle time and the sweep frequency for expired items.
//
//nolint:whitespace // editor/linter issue
func WithTTL[K comparable, V any](
	ttl time.Duration,
	cleanupInterval time.Duration,
) Option[K, V] {
	return func(c *Cache[K, V]) {
		c.ttl = ttl
		c.cleanupInterval = cleanupInterval
	}
}

// WithCapacity bounds the cache to a maximum number of entries.
func WithCapacity[K comparable, V any](capacity int) Option[K, V] {
	return func(c *Cache[K, V]) {
		c.capacity = capacity
	}
}

// Option to attach CacheManager
//
//nolint:whitespace // editor/linter issue
func WithCacheManager[K comparable, V any](
	manager *CacheManager, entityName string,
) Option[K, V] {
	return func(c *Cache[K, V]) {
		c.cacheManager = manager
		c.entityName = entityName
	}
}

// WithCloner sets a custom cloning function for cache values.
func WithCloner[K comparable, V any](cloner func(V) V) Option[K, V] {
	return func(c *Cache[K, V]) {
		c.cloner = cloner
	}
}

// WithClone sets a default cloning function for cache values
// using gob encoding/decoding.
func WithClone[K comparable, V any]() Option[K, V] {
	return func(c *Cache[K, V]) {
		c.cloner = func(v V) V {
			cloned, _ := Clone(v)
			return cloned
		}
	}
}

//nolint:lll // readability
type Cache[K comparable, V any] struct {
	mu              sync.RWMutex
	items           map[K]*list.Element // key -> list element containing *entry[K, V]
	evictList       *list.List          // front = most recently used, back = least recently used
	capacity        int
	ttl             time.Duration
	cleanupInterval time.Duration
	name            string
	attrs           attribute.Set

	entityName   string
	cacheManager *CacheManager
	l            *log.Logger
	cloner       func(V) V
}

// metrics for the cache

type cacheInstance interface {
	Name() string
	Size() int64
}

var (
	metricsOnce      sync.Once
	requestsCounter  metric.Int64Counter
	evictionsCounter metric.Int64Counter
	sizeGauge        metric.Int64ObservableGauge
	registryMu       sync.RWMutex
	registry         = make(map[string]cacheInstance)
)
var _ cacheInstance = (*Cache[any, any])(nil) // compile-time check

//nolint:whitespace // editor/linter issue
func New[K comparable, V any](
	ctx context.Context,
	name string,
	opts ...Option[K, V],
) (*Cache[K, V], error) {
	c := &Cache[K, V]{
		items:     make(map[K]*list.Element),
		evictList: list.New(),
		name:      name,
		attrs:     attribute.NewSet(attribute.String("cache.name", name)),

		l: log.GetFromContext(ctx).Named("cache").Named(name),
	}

	for _, opt := range opts {
		opt(c)
	}
	c.l.Debug("cache created",
		log.String("name", name),
		log.Int("capacity", c.capacity),
		log.Duration("ttl", c.ttl),
		log.Duration("cleanupInterval", c.cleanupInterval),
	)

	if c.ttl > 0 && c.cleanupInterval > 0 {
		c.startEvictionManager(ctx)
	}
	// Register instance for gauge callbacks
	registryMu.Lock()
	registry[name] = c
	registryMu.Unlock()
	return c, nil
}

func Clone[T any](v T) (T, error) {
	var buf bytes.Buffer

	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		var zero T
		return zero, err
	}

	var result T
	if err := gob.NewDecoder(&buf).Decode(&result); err != nil {
		var zero T
		return zero, err
	}

	return result, nil
}

func (c *Cache[K, V]) Name() string {
	return c.name
}

func (c *Cache[K, V]) Size() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return int64(c.evictList.Len())
}

func (c *Cache[K, V]) Keys() []K {
	c.mu.RLock()
	defer c.mu.RUnlock()
	keys := make([]K, 0, len(c.items))
	for k := range c.items {
		keys = append(keys, k)
	}
	return keys
}

func (c *Cache[K, V]) Get(ctx context.Context, key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	elem, ok := c.items[key]
	var val V
	outcomeAttr := attribute.Bool("cache.hit", false)
	if ok {
		c.evictList.MoveToFront(elem)
		e := elem.Value.(*entry[K, V]) //nolint:errcheck // by design
		e.lastAccess = time.Now()
		if c.cloner != nil {
			val = c.cloner(e.value)
		} else {
			val = e.value
		}
		outcomeAttr = attribute.Bool("cache.hit", true)
	}

	requestsCounter.Add(ctx, 1,
		metric.WithAttributeSet(c.attrs),
		metric.WithAttributes(outcomeAttr),
	)

	return val, ok
}

func (c *Cache[K, V]) Set(ctx context.Context, key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	// Update existing entry
	if elem, ok := c.items[key]; ok {
		c.evictList.MoveToFront(elem)
		e := elem.Value.(*entry[K, V]) //nolint:errcheck // by design
		e.value = value
		e.lastAccess = now
		return
	}

	// Add new entry
	e := &entry[K, V]{
		key:        key,
		value:      value,
		lastAccess: now,
	}
	elem := c.evictList.PushFront(e)
	c.items[key] = elem

	// Enforce capacity bounds (LRU Eviction)
	if c.capacity > 0 && c.evictList.Len() > c.capacity {
		c.removeOldest(ctx, "capacity")
	}
}

// SetAndPublish updates the cache and broadcasts the change to the bus.
//
//nolint:whitespace // editor/linter issue
func (c *Cache[K, V]) SetAndPublish(
	ctx context.Context, key K, value V, action Action,
) {
	c.Set(ctx, key, value)

	if c.cacheManager != nil {
		c.cacheManager.Publish(ctx, InvalidationEvent{
			EntityName: c.entityName,
			EntityID:   key,
			Action:     action,
			Payload:    value,
		})
	}
}

// Range inspects cache items safely under RLock (needed for reactive eviction scans).
func (c *Cache[K, V]) Range(fn func(key K, value V) bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for k, elem := range c.items {
		e := elem.Value.(*entry[K, V]) //nolint:errcheck // by design
		if !fn(k, e.value) {
			break
		}
	}
}

func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.removeElement(elem)
	}
}

func (c *Cache[K, V]) DeleteAndPublish(ctx context.Context, key K) {
	c.Delete(key)

	if c.cacheManager != nil {
		c.cacheManager.Publish(ctx, InvalidationEvent{
			EntityName: c.entityName,
			EntityID:   key,
			Action:     ActionDeleted,
		})
	}
}

func (c *Cache[K, V]) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()

	clear(c.items)
	c.evictList.Init()
}

func (c *Cache[K, V]) removeElement(elem *list.Element) {
	c.evictList.Remove(elem)
	e := elem.Value.(*entry[K, V]) //nolint:errcheck // by design
	delete(c.items, e.key)
}

func (c *Cache[K, V]) removeOldest(ctx context.Context, reason string) {
	elem := c.evictList.Back()
	if elem != nil {
		c.removeElement(elem)
		evictionsCounter.Add(ctx, 1,
			metric.WithAttributeSet(c.attrs),
			metric.WithAttributes(attribute.String("reason", reason)))
	}
}

func (c *Cache[K, V]) startEvictionManager(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(c.cleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				c.evictExpired(ctx, now)
			}
		}
	}()
}

func (c *Cache[K, V]) evictExpired(ctx context.Context, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	evicted := int64(0)
	// Iterate from LRU (back) to MRU (front)
	for elem := c.evictList.Back(); elem != nil; {
		prev := elem.Prev()            // Save reference before removal
		e := elem.Value.(*entry[K, V]) //nolint:errcheck // by design

		if now.Sub(e.lastAccess) > c.ttl {
			c.removeElement(elem)
			evicted++
		}
		elem = prev
	}

	if evicted > 0 {
		evictionsCounter.Add(ctx, 1,
			metric.WithAttributeSet(c.attrs),
			metric.WithAttributes(attribute.String("reason", "ttl")))
	}
}

func (c *Cache[K, V]) ListenEvents(ctx context.Context, events <-chan Event[K, V]) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-events:
				if !ok {
					return
				}
				switch evt.Type {
				case EventFlush:
					c.Flush()
				case EventUpdate:
					c.Set(ctx, evt.Key, evt.Value)
				}
			}
		}
	}()
}

// InitMetrics initializes OTel instruments ONCE and registers the async gauge callback.
// Call this in main() or let New() invoke it lazily with a default meter.
func InitMetrics(meter metric.Meter) error {
	var initErr error
	metricsOnce.Do(func() {
		requestsCounter, initErr = meter.Int64Counter(
			"cache.requests",
			metric.WithDescription("Total cache request operations"),
		)
		if initErr != nil {
			return
		}

		evictionsCounter, initErr = meter.Int64Counter(
			"cache.evictions",
			metric.WithDescription("Total cache evictions"),
		)
		if initErr != nil {
			return
		}

		sizeGauge, initErr = meter.Int64ObservableGauge(
			"cache.size",
			metric.WithDescription("Current element count in cache"),
		)
		if initErr != nil {
			return
		}

		// Register a single global callback for all current and future caches
		_, initErr = meter.RegisterCallback(
			func(ctx context.Context, obs metric.Observer) error {
				registryMu.RLock()
				defer registryMu.RUnlock()

				for name, c := range registry {
					obs.ObserveInt64(
						sizeGauge,
						c.Size(),
						metric.WithAttributes(attribute.String("cache.name", name)),
					)
				}
				return nil
			},
			sizeGauge,
		)
	})

	return initErr
}
