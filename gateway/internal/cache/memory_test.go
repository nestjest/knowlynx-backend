package cache

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"gateway/internal/errs"
)

func TestMemoryCache_SetGetDelete(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cache := NewMemoryCache[string]()

	if err := cache.Set(ctx, "key", "value", time.Minute); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	got, err := cache.Get(ctx, "key")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got != "value" {
		t.Fatalf("Get = %q, want %q", got, "value")
	}

	if err := cache.Delete(ctx, "key"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if _, err := cache.Get(ctx, "key"); !errors.Is(err, errs.ErrKeyNotFound) {
		t.Fatalf("Get deleted key error = %v, want %v", err, errs.ErrKeyNotFound)
	}
}

func TestMemoryCache_GetMissingAndExpired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cache := NewMemoryCache[string]()

	if _, err := cache.Get(ctx, "missing"); !errors.Is(err, errs.ErrKeyNotFound) {
		t.Fatalf("Get missing key error = %v, want %v", err, errs.ErrKeyNotFound)
	}

	if err := cache.Set(ctx, "expired", "value", -time.Nanosecond); err != nil {
		t.Fatalf("Set expired item returned error: %v", err)
	}
	if _, err := cache.Get(ctx, "expired"); !errors.Is(err, errs.ErrKeyNotFound) {
		t.Fatalf("Get expired key error = %v, want %v", err, errs.ErrKeyNotFound)
	}
	if _, ok := cache.items["expired"]; ok {
		t.Fatalf("expired key was not removed from cache storage")
	}
}

func TestMemoryCache_ContextErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ctx     func() (context.Context, context.CancelFunc)
		wantErr error
	}{
		{
			name: "canceled context",
			ctx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
			wantErr: context.Canceled,
		},
		{
			name: "deadline exceeded context",
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			},
			wantErr: context.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewMemoryCache[string]()
			if err := cache.Set(context.Background(), "key", "value", time.Minute); err != nil {
				t.Fatalf("Set setup returned error: %v", err)
			}

			ctx, cancel := tt.ctx()
			t.Cleanup(cancel)

			if err := cache.Set(ctx, "other", "value", time.Minute); !errors.Is(err, tt.wantErr) {
				t.Fatalf("Set context error = %v, want %v", err, tt.wantErr)
			}
			if _, err := cache.Get(ctx, "key"); !errors.Is(err, tt.wantErr) {
				t.Fatalf("Get context error = %v, want %v", err, tt.wantErr)
			}
			if err := cache.Delete(ctx, "key"); !errors.Is(err, tt.wantErr) {
				t.Fatalf("Delete context error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestMemoryCache_DeleteMissingIsNoop(t *testing.T) {
	t.Parallel()

	cache := NewMemoryCache[string]()
	if err := cache.Delete(context.Background(), "missing"); err != nil {
		t.Fatalf("Delete missing key returned error: %v", err)
	}
}

func TestMemoryCache_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cache := NewMemoryCache[int]()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()

			key := fmt.Sprintf("key-%d", i)
			if err := cache.Set(ctx, key, i, time.Minute); err != nil {
				t.Errorf("Set returned error: %v", err)
				return
			}

			got, err := cache.Get(ctx, key)
			if err != nil {
				t.Errorf("Get returned error: %v", err)
				return
			}
			if got != i {
				t.Errorf("Get = %d, want %d", got, i)
			}

			if i%2 == 0 {
				if err := cache.Delete(ctx, key); err != nil {
					t.Errorf("Delete returned error: %v", err)
				}
			}
		}()
	}
	wg.Wait()
}
