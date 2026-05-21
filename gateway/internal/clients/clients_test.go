package clients

import (
	"context"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

func TestRegisterHandlers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		register   func(context.Context, *runtime.ServeMux, string) error
		addr       string
		wantErrMsg string
	}{
		{
			name:     "auth handler registration succeeds",
			register: RegisterAuthHandler,
			addr:     "127.0.0.1:1",
		},
		{
			name:     "users handler registration succeeds",
			register: RegisterUsersHandler,
			addr:     "127.0.0.1:2",
		},
		{
			name:       "auth handler wraps invalid endpoint error",
			register:   RegisterAuthHandler,
			addr:       "%",
			wantErrMsg: "failed to register auth service handler",
		},
		{
			name:       "users handler wraps invalid endpoint error",
			register:   RegisterUsersHandler,
			addr:       "%",
			wantErrMsg: "failed to register users service handler",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			err := tt.register(ctx, runtime.NewServeMux(), tt.addr)
			if tt.wantErrMsg == "" {
				if err != nil {
					t.Fatalf("register returned error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("register returned nil error, want %q", tt.wantErrMsg)
			}
			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Fatalf("register error = %q, want to contain %q", err.Error(), tt.wantErrMsg)
			}
		})
	}
}
