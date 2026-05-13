package cache

import (
	"context"
	"gateway/internal/errs"
	"sync"
	"time"
)

type item[T any] struct {
	Value     T
	ExpiresAt time.Time
}

type MemoryCache[T any] struct {
	mu    sync.RWMutex
	items map[string]item[T]
}

func NewMemoryCache[T any]() *MemoryCache[T] {
	return &MemoryCache[T]{
		items: make(map[string]item[T]),
	}
}

func (c *MemoryCache[T]) Get(ctx context.Context, key string) (T, error) {
	var zero T

	select {
	case <-ctx.Done():
		return zero, ctx.Err()
	default:

	}

	c.mu.RLock()

	item, ok := c.items[key]

	c.mu.RUnlock()

	if !ok {
		return zero, errs.ErrKeyNotFound
	}

	if time.Now().After(item.ExpiresAt) {
		_ = c.Delete(ctx, key)
		return zero, errs.ErrKeyNotFound
	}

	return item.Value, nil

}

func (c *MemoryCache[T]) Set(ctx context.Context, key string, value T, ttl time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c.mu.Lock()

	defer c.mu.Unlock()

	c.items[key] = item[T]{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}

	return nil
}

func (c *MemoryCache[T]) Delete(ctx context.Context, key string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c.mu.Lock()

	defer c.mu.Unlock()

	delete(c.items, key)

	return nil
}
