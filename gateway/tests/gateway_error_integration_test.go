package tests

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authv1 "contracts/gen/go/proto/auth/v1"
	usersv1 "contracts/gen/go/proto/users/v1"
	"gateway/internal/config"
	gatewayserver "gateway/internal/server"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type failingAuthServer struct {
	authv1.UnimplementedAuthServiceServer
}

func (s *failingAuthServer) Login(context.Context, *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	return nil, status.Error(codes.Unauthenticated, "bad credentials")
}

type failingUsersServer struct {
	usersv1.UnimplementedUsersServiceServer
}

func (s *failingUsersServer) GetUser(context.Context, *usersv1.GetUserRequest) (*usersv1.GetUserResponse, error) {
	return nil, status.Error(codes.NotFound, "user not found")
}

func TestGatewayRejectsInvalidHTTPRequests(t *testing.T) {
	t.Parallel()

	httpServer := startGatewayTestServer(t, &fakeAuthServer{}, &fakeUsersServer{})

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		wantStatusCode int
		wantBodyPart   string
	}{
		{
			name:           "invalid json body",
			method:         http.MethodPost,
			path:           "/api/v1/auth/login",
			body:           `{"username":`,
			wantStatusCode: http.StatusBadRequest,
			wantBodyPart:   "unexpected EOF",
		},
		{
			name:           "unknown route",
			method:         http.MethodGet,
			path:           "/api/v1/unknown",
			wantStatusCode: http.StatusNotFound,
			wantBodyPart:   "Not Found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statusCode, body := doGatewayRequestRaw(t, httpServer.URL, tt.method, tt.path, tt.body)

			if statusCode != tt.wantStatusCode {
				t.Fatalf("status = %d, want %d; body = %s", statusCode, tt.wantStatusCode, body)
			}
			if !strings.Contains(body, tt.wantBodyPart) {
				t.Fatalf("body = %q, want to contain %q", body, tt.wantBodyPart)
			}
		})
	}
}

func TestGatewayMapsGRPCErrorsToHTTP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		authServer     authv1.AuthServiceServer
		usersServer    usersv1.UsersServiceServer
		method         string
		path           string
		body           string
		wantStatusCode int
		wantBodyPart   string
	}{
		{
			name:           "unauthenticated login maps to 401",
			authServer:     &failingAuthServer{},
			usersServer:    &fakeUsersServer{},
			method:         http.MethodPost,
			path:           "/api/v1/auth/login",
			body:           `{"username":"alice","password":"wrong"}`,
			wantStatusCode: http.StatusUnauthorized,
			wantBodyPart:   "bad credentials",
		},
		{
			name:           "missing user maps to 404",
			authServer:     &fakeAuthServer{},
			usersServer:    &failingUsersServer{},
			method:         http.MethodGet,
			path:           "/api/v1/users/missing",
			wantStatusCode: http.StatusNotFound,
			wantBodyPart:   "user not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpServer := startGatewayTestServerWithServices(t, tt.authServer, tt.usersServer)

			statusCode, body := doGatewayRequestRaw(t, httpServer.URL, tt.method, tt.path, tt.body)

			if statusCode != tt.wantStatusCode {
				t.Fatalf("status = %d, want %d; body = %s", statusCode, tt.wantStatusCode, body)
			}
			if !strings.Contains(body, tt.wantBodyPart) {
				t.Fatalf("body = %q, want to contain %q", body, tt.wantBodyPart)
			}
		})
	}
}

func startGatewayTestServerWithServices(t *testing.T, authSrv authv1.AuthServiceServer, usersSrv usersv1.UsersServiceServer) *httptest.Server {
	t.Helper()

	authAddr, stopAuth := startGRPCServer(t, func(s *grpc.Server) {
		authv1.RegisterAuthServiceServer(s, authSrv)
	})
	t.Cleanup(stopAuth)

	usersAddr, stopUsers := startGRPCServer(t, func(s *grpc.Server) {
		usersv1.RegisterUsersServiceServer(s, usersSrv)
	})
	t.Cleanup(stopUsers)

	handler, err := gatewayserver.NewHandler(context.Background(), config.Config{
		AuthService: config.AuthServiceConfig{
			Addr: authAddr,
		},
		UsersService: config.UsersServiceConfig{
			Addr: usersAddr,
		},
	})
	if err != nil {
		t.Fatalf("NewHandler returned error: %v", err)
	}

	httpServer := httptest.NewServer(handler)
	t.Cleanup(httpServer.Close)

	return httpServer
}

func doGatewayRequestRaw(t *testing.T, baseURL, method, path, body string) (int, string) {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = bytes.NewBufferString(body)
	}
	req, err := http.NewRequest(method, baseURL+path, reader)
	if err != nil {
		t.Fatalf("NewRequest returned error: %v", err)
	}
	req.Header.Set("Authorization", "Bearer integration-token")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s returned error: %v", method, path, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}

	return resp.StatusCode, string(data)
}
