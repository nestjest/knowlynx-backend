package transport

import (
	"context"
	"net/http"
	"testing"
)

func TestHTTPHeadersToMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		setupHeader       func(http.Header)
		wantAuthorization []string
	}{
		{
			name: "forwards authorization header",
			setupHeader: func(header http.Header) {
				header.Set("Authorization", "Bearer token")
			},
			wantAuthorization: []string{"Bearer token"},
		},
		{
			name: "canonicalizes lowercase authorization header",
			setupHeader: func(header http.Header) {
				header.Set("authorization", "Bearer lower")
			},
			wantAuthorization: []string{"Bearer lower"},
		},
		{
			name: "skips missing authorization header",
			setupHeader: func(http.Header) {
			},
			wantAuthorization: nil,
		},
		{
			name: "skips empty authorization header",
			setupHeader: func(header http.Header) {
				header.Set("Authorization", "")
			},
			wantAuthorization: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "/", nil)
			if err != nil {
				t.Fatalf("NewRequest returned error: %v", err)
			}
			tt.setupHeader(req.Header)

			md := HTTPHeadersToMetadata(context.Background(), req)

			got := md.Get("authorization")
			if len(got) != len(tt.wantAuthorization) {
				t.Fatalf("authorization metadata = %v, want %v", got, tt.wantAuthorization)
			}
			for i := range got {
				if got[i] != tt.wantAuthorization[i] {
					t.Fatalf("authorization metadata = %v, want %v", got, tt.wantAuthorization)
				}
			}
		})
	}
}
