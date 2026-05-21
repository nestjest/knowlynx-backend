package tests

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"gateway/internal/cache"
	"gateway/internal/config"
	"gateway/internal/errs"
	"gateway/internal/transport"
)

func TestMemoryCacheSetGetDeleteAndExpiry(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	c := cache.NewMemoryCache[string]()

	if _, err := c.Get(ctx, "missing"); !errors.Is(err, errs.ErrKeyNotFound) {
		t.Fatalf("Get missing key error = %v, want %v", err, errs.ErrKeyNotFound)
	}

	if err := c.Set(ctx, "key", "value", time.Minute); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	got, err := c.Get(ctx, "key")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got != "value" {
		t.Fatalf("Get = %q, want %q", got, "value")
	}

	if err := c.Delete(ctx, "key"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if _, err := c.Get(ctx, "key"); !errors.Is(err, errs.ErrKeyNotFound) {
		t.Fatalf("Get deleted key error = %v, want %v", err, errs.ErrKeyNotFound)
	}

	if err := c.Set(ctx, "expired", "value", -time.Nanosecond); err != nil {
		t.Fatalf("Set expired item returned error: %v", err)
	}
	if _, err := c.Get(ctx, "expired"); !errors.Is(err, errs.ErrKeyNotFound) {
		t.Fatalf("Get expired key error = %v, want %v", err, errs.ErrKeyNotFound)
	}
}

func TestMemoryCacheHonorsCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := cache.NewMemoryCache[string]()
	if err := c.Set(ctx, "key", "value", time.Minute); !errors.Is(err, context.Canceled) {
		t.Fatalf("Set canceled context error = %v, want %v", err, context.Canceled)
	}
	if _, err := c.Get(ctx, "key"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Get canceled context error = %v, want %v", err, context.Canceled)
	}
	if err := c.Delete(ctx, "key"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Delete canceled context error = %v, want %v", err, context.Canceled)
	}
}

func TestMemoryCacheConcurrentAccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	c := cache.NewMemoryCache[int]()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := "key"
			if err := c.Set(ctx, key, i, time.Minute); err != nil {
				t.Errorf("Set returned error: %v", err)
				return
			}
			if _, err := c.Get(ctx, key); err != nil {
				t.Errorf("Get returned error: %v", err)
			}
		}(i)
	}
	wg.Wait()
}

func TestHTTPHeadersToMetadata(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatalf("NewRequest returned error: %v", err)
	}
	req.Header.Set("Authorization", "Bearer token")

	md := transport.HTTPHeadersToMetadata(context.Background(), req)
	if got := md.Get("authorization"); len(got) != 1 || got[0] != "Bearer token" {
		t.Fatalf("authorization metadata = %v, want %v", got, []string{"Bearer token"})
	}

	req.Header.Del("Authorization")
	md = transport.HTTPHeadersToMetadata(context.Background(), req)
	if got := md.Get("authorization"); len(got) != 0 {
		t.Fatalf("authorization metadata for empty header = %v, want empty", got)
	}
}

func TestConfigLoadSuccess(t *testing.T) {
	t.Setenv("SERVER_ADDR", "")
	t.Setenv("RHT", "3s")
	t.Setenv("AUTH_SERVICE_ADDR", "127.0.0.1:10001")
	t.Setenv("USERS_SERVICE_ADDR", "127.0.0.1:10002")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Server.HttpAddr != ":8080" {
		t.Fatalf("Server.HttpAddr = %q, want %q", cfg.Server.HttpAddr, ":8080")
	}
	if cfg.Server.RHT != 3*time.Second {
		t.Fatalf("Server.RHT = %v, want %v", cfg.Server.RHT, 3*time.Second)
	}
	if cfg.AuthService.Addr != "127.0.0.1:10001" {
		t.Fatalf("AuthService.Addr = %q", cfg.AuthService.Addr)
	}
	if cfg.UsersService.Addr != "127.0.0.1:10002" {
		t.Fatalf("UsersService.Addr = %q", cfg.UsersService.Addr)
	}
}

func TestConfigLoadErrors(t *testing.T) {
	tests := []struct {
		name       string
		setupEnv   func(t *testing.T)
		wantErrMsg string
	}{
		{
			name: "missing auth service addr",
			setupEnv: func(t *testing.T) {
				t.Setenv("AUTH_SERVICE_ADDR", "")
			},
			wantErrMsg: "env var AUTH_SERVICE_ADDR is required but not set",
		},
		{
			name: "invalid read header timeout",
			setupEnv: func(t *testing.T) {
				t.Setenv("RHT", "not-a-duration")
			},
			wantErrMsg: "invalid duration for RHT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SERVER_ADDR", "")
			t.Setenv("RHT", "2s")
			t.Setenv("AUTH_SERVICE_ADDR", "127.0.0.1:10001")
			t.Setenv("USERS_SERVICE_ADDR", "127.0.0.1:10002")
			tt.setupEnv(t)

			cfg, err := config.Load()
			if err == nil {
				t.Fatalf("Load returned nil error, want %q", tt.wantErrMsg)
			}
			if cfg != nil {
				t.Fatalf("Load returned config on error: %+v", cfg)
			}
			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Fatalf("Load error = %q, want to contain %q", err.Error(), tt.wantErrMsg)
			}
		})
	}
}
