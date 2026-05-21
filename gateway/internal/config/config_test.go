package config

import (
	"strings"
	"testing"
	"time"
)

func TestCORSConfig_WithDefaults(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins: []string{"https://app.example.com"},
	}

	got := cfg.WithDefaults()

	if len(got.AllowedOrigins) != 1 || got.AllowedOrigins[0] != "https://app.example.com" {
		t.Fatalf("AllowedOrigins = %v, want custom origin", got.AllowedOrigins)
	}
	if len(got.AllowedMethods) == 0 {
		t.Fatalf("AllowedMethods was not filled from defaults")
	}
	if len(got.AllowedHeaders) == 0 {
		t.Fatalf("AllowedHeaders was not filled from defaults")
	}
}

func TestCORSConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		cfg        CORSConfig
		wantErrMsg string
	}{
		{
			name: "valid config",
			cfg: CORSConfig{
				AllowedOrigins: []string{"https://app.example.com"},
				AllowedMethods: []string{"GET", "POST"},
				AllowedHeaders: []string{"Authorization", "Content-Type"},
			},
		},
		{
			name: "empty origins",
			cfg: CORSConfig{
				AllowedMethods: []string{"GET"},
				AllowedHeaders: []string{"Authorization"},
			},
			wantErrMsg: "env var CORS_ALLOWED_ORIGINS must contain at least one value",
		},
		{
			name: "blank origin",
			cfg: CORSConfig{
				AllowedOrigins: []string{" "},
				AllowedMethods: []string{"GET"},
				AllowedHeaders: []string{"Authorization"},
			},
			wantErrMsg: "env var CORS_ALLOWED_ORIGINS contains empty value",
		},
		{
			name: "wildcard origin with credentials",
			cfg: CORSConfig{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET"},
				AllowedHeaders: []string{"Authorization"},
			},
			wantErrMsg: "env var CORS_ALLOWED_ORIGINS cannot contain wildcard origin",
		},
		{
			name: "empty methods",
			cfg: CORSConfig{
				AllowedOrigins: []string{"https://app.example.com"},
				AllowedHeaders: []string{"Authorization"},
			},
			wantErrMsg: "env var CORS_ALLOWED_METHODS must contain at least one value",
		},
		{
			name: "blank method",
			cfg: CORSConfig{
				AllowedOrigins: []string{"https://app.example.com"},
				AllowedMethods: []string{"GET", ""},
				AllowedHeaders: []string{"Authorization"},
			},
			wantErrMsg: "env var CORS_ALLOWED_METHODS contains empty value",
		},
		{
			name: "empty headers",
			cfg: CORSConfig{
				AllowedOrigins: []string{"https://app.example.com"},
				AllowedMethods: []string{"GET"},
			},
			wantErrMsg: "env var CORS_ALLOWED_HEADERS must contain at least one value",
		},
		{
			name: "blank header",
			cfg: CORSConfig{
				AllowedOrigins: []string{"https://app.example.com"},
				AllowedMethods: []string{"GET"},
				AllowedHeaders: []string{"Authorization", " "},
			},
			wantErrMsg: "env var CORS_ALLOWED_HEADERS contains empty value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErrMsg == "" {
				if err != nil {
					t.Fatalf("Validate returned error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("Validate returned nil error, want %q", tt.wantErrMsg)
			}
			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Fatalf("Validate error = %q, want to contain %q", err.Error(), tt.wantErrMsg)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		setupEnv func(t *testing.T)
		assert   func(t *testing.T, cfg *Config)
	}{
		{
			name: "uses defaults for optional server addr and cors",
			setupEnv: func(t *testing.T) {
				setRequiredGatewayEnv(t)
				t.Setenv("SERVER_ADDR", "")
				t.Setenv("CORS_ALLOWED_ORIGINS", "")
				t.Setenv("CORS_ALLOWED_METHODS", "")
				t.Setenv("CORS_ALLOWED_HEADERS", "")
			},
			assert: func(t *testing.T, cfg *Config) {
				if cfg.Server.HttpAddr != ":8080" {
					t.Fatalf("Server.HttpAddr = %q, want %q", cfg.Server.HttpAddr, ":8080")
				}
				if cfg.Server.RHT != 3*time.Second {
					t.Fatalf("Server.RHT = %v, want %v", cfg.Server.RHT, 3*time.Second)
				}
				if got := cfg.CORS.AllowedOrigins; len(got) != 1 || got[0] != "http://localhost:3000" {
					t.Fatalf("CORS.AllowedOrigins = %v, want default localhost origin", got)
				}
			},
		},
		{
			name: "parses custom values and trims cors lists",
			setupEnv: func(t *testing.T) {
				setRequiredGatewayEnv(t)
				t.Setenv("SERVER_ADDR", ":9090")
				t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com, http://localhost:5173")
				t.Setenv("CORS_ALLOWED_METHODS", "GET, POST")
				t.Setenv("CORS_ALLOWED_HEADERS", "Authorization, Content-Type, X-Request-ID")
			},
			assert: func(t *testing.T, cfg *Config) {
				if cfg.Server.HttpAddr != ":9090" {
					t.Fatalf("Server.HttpAddr = %q, want %q", cfg.Server.HttpAddr, ":9090")
				}
				if cfg.AuthService.Addr != "127.0.0.1:10001" {
					t.Fatalf("AuthService.Addr = %q, want %q", cfg.AuthService.Addr, "127.0.0.1:10001")
				}
				if cfg.UsersService.Addr != "127.0.0.1:10002" {
					t.Fatalf("UsersService.Addr = %q, want %q", cfg.UsersService.Addr, "127.0.0.1:10002")
				}
				if got := cfg.CORS.AllowedOrigins; len(got) != 2 || got[0] != "https://app.example.com" || got[1] != "http://localhost:5173" {
					t.Fatalf("CORS.AllowedOrigins = %v", got)
				}
				if got := cfg.CORS.AllowedMethods; len(got) != 2 || got[0] != "GET" || got[1] != "POST" {
					t.Fatalf("CORS.AllowedMethods = %v", got)
				}
				if got := cfg.CORS.AllowedHeaders; len(got) != 3 || got[2] != "X-Request-ID" {
					t.Fatalf("CORS.AllowedHeaders = %v", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv(t)

			cfg, err := Load()

			if err != nil {
				t.Fatalf("Load returned error: %v", err)
			}
			if cfg == nil {
				t.Fatalf("Load returned nil config")
			}
			tt.assert(t, cfg)
		})
	}
}

func TestLoad_Errors(t *testing.T) {
	tests := []struct {
		name       string
		setupEnv   func(t *testing.T)
		wantErrMsg string
	}{
		{
			name: "missing read header timeout",
			setupEnv: func(t *testing.T) {
				setRequiredGatewayEnv(t)
				t.Setenv("RHT", "")
			},
			wantErrMsg: "env var RHT is required but not set",
		},
		{
			name: "invalid read header timeout",
			setupEnv: func(t *testing.T) {
				setRequiredGatewayEnv(t)
				t.Setenv("RHT", "not-a-duration")
			},
			wantErrMsg: "invalid duration for RHT",
		},
		{
			name: "missing auth service address",
			setupEnv: func(t *testing.T) {
				setRequiredGatewayEnv(t)
				t.Setenv("AUTH_SERVICE_ADDR", "")
			},
			wantErrMsg: "env var AUTH_SERVICE_ADDR is required but not set",
		},
		{
			name: "missing users service address",
			setupEnv: func(t *testing.T) {
				setRequiredGatewayEnv(t)
				t.Setenv("USERS_SERVICE_ADDR", "")
			},
			wantErrMsg: "env var USERS_SERVICE_ADDR is required but not set",
		},
		{
			name: "cors list contains empty value",
			setupEnv: func(t *testing.T) {
				setRequiredGatewayEnv(t)
				t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com,")
			},
			wantErrMsg: "env var CORS_ALLOWED_ORIGINS contains empty value",
		},
		{
			name: "cors wildcard origin is rejected",
			setupEnv: func(t *testing.T) {
				setRequiredGatewayEnv(t)
				t.Setenv("CORS_ALLOWED_ORIGINS", "*")
			},
			wantErrMsg: "env var CORS_ALLOWED_ORIGINS cannot contain wildcard origin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv(t)

			cfg, err := Load()

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

func setRequiredGatewayEnv(t *testing.T) {
	t.Helper()

	t.Setenv("SERVER_ADDR", "")
	t.Setenv("RHT", "3s")
	t.Setenv("AUTH_SERVICE_ADDR", "127.0.0.1:10001")
	t.Setenv("USERS_SERVICE_ADDR", "127.0.0.1:10002")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	t.Setenv("CORS_ALLOWED_METHODS", "")
	t.Setenv("CORS_ALLOWED_HEADERS", "")
}
