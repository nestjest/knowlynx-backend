package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gateway/internal/config"
)

func TestNewHandler_CORSPreflight(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	handler, err := NewHandler(ctx, validHandlerConfig())
	if err != nil {
		t.Fatalf("NewHandler returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "authorization, content-type")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, "http://localhost:3000")
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Access-Control-Allow-Credentials = %q, want %q", got, "true")
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, http.MethodPost) {
		t.Fatalf("Access-Control-Allow-Methods = %q, want to contain %q", got, http.MethodPost)
	}
}

func TestNewHandler_CORSValidationErrors(t *testing.T) {
	tests := []struct {
		name       string
		cors       config.CORSConfig
		wantErrMsg string
	}{
		{
			name: "wildcard origin is rejected with credentials",
			cors: config.CORSConfig{
				AllowedOrigins: []string{"*"},
			},
			wantErrMsg: "validate cors config: env var CORS_ALLOWED_ORIGINS cannot contain wildcard origin",
		},
		{
			name: "blank method is rejected",
			cors: config.CORSConfig{
				AllowedMethods: []string{"POST", " "},
			},
			wantErrMsg: "validate cors config: env var CORS_ALLOWED_METHODS contains empty value",
		},
		{
			name: "blank header is rejected",
			cors: config.CORSConfig{
				AllowedHeaders: []string{"Authorization", ""},
			},
			wantErrMsg: "validate cors config: env var CORS_ALLOWED_HEADERS contains empty value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validHandlerConfig()
			cfg.CORS = tt.cors

			handler, err := NewHandler(context.Background(), cfg)

			if err == nil {
				t.Fatalf("NewHandler returned nil error, want %q", tt.wantErrMsg)
			}
			if handler != nil {
				t.Fatalf("NewHandler returned handler on validation error")
			}
			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Fatalf("NewHandler error = %q, want to contain %q", err.Error(), tt.wantErrMsg)
			}
		})
	}
}

func TestNewHandler_RegisterErrors(t *testing.T) {
	tests := []struct {
		name       string
		cfg        config.Config
		wantErrMsg string
	}{
		{
			name: "auth endpoint registration error is wrapped",
			cfg: config.Config{
				AuthService: config.AuthServiceConfig{
					Addr: "%",
				},
				UsersService: config.UsersServiceConfig{
					Addr: "127.0.0.1:1",
				},
			},
			wantErrMsg: "register auth handler: failed to register auth service handler",
		},
		{
			name: "users endpoint registration error is wrapped",
			cfg: config.Config{
				AuthService: config.AuthServiceConfig{
					Addr: "127.0.0.1:1",
				},
				UsersService: config.UsersServiceConfig{
					Addr: "%",
				},
			},
			wantErrMsg: "register users handler: failed to register users service handler",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			handler, err := NewHandler(ctx, tt.cfg)

			if err == nil {
				t.Fatalf("NewHandler returned nil error, want %q", tt.wantErrMsg)
			}
			if handler != nil {
				t.Fatalf("NewHandler returned handler on registration error")
			}
			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Fatalf("NewHandler error = %q, want to contain %q", err.Error(), tt.wantErrMsg)
			}
		})
	}
}

func validHandlerConfig() config.Config {
	return config.Config{
		AuthService: config.AuthServiceConfig{
			Addr: "127.0.0.1:1",
		},
		UsersService: config.UsersServiceConfig{
			Addr: "127.0.0.1:2",
		},
	}
}
