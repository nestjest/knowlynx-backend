package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	authv1 "contracts/gen/go/proto/auth/v1"
	usersv1 "contracts/gen/go/proto/users/v1"
	"gateway/internal/config"
	gatewayserver "gateway/internal/server"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type recordedCall struct {
	method string
	req    any
	md     metadata.MD
}

type fakeAuthServer struct {
	authv1.UnimplementedAuthServiceServer

	mu   sync.Mutex
	last recordedCall
}

func (s *fakeAuthServer) record(ctx context.Context, method string, req any) {
	md, _ := metadata.FromIncomingContext(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.last = recordedCall{method: method, req: req, md: md}
}

func (s *fakeAuthServer) lastCall() recordedCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.last
}

func (s *fakeAuthServer) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	s.record(ctx, "Login", req)
	return &authv1.LoginResponse{AccessToken: "access-token", RefreshToken: "refresh-token"}, nil
}

func (s *fakeAuthServer) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	s.record(ctx, "RefreshToken", req)
	return &authv1.RefreshTokenResponse{AccessToken: "access-token-2", RefreshToken: "refresh-token-2"}, nil
}

func (s *fakeAuthServer) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	s.record(ctx, "Logout", req)
	return &authv1.LogoutResponse{Success: true}, nil
}

func (s *fakeAuthServer) GetMe(ctx context.Context, req *authv1.GetMeRequest) (*authv1.GetMeResponse, error) {
	s.record(ctx, "GetMe", req)
	return &authv1.GetMeResponse{User: testUser("me")}, nil
}

type fakeUsersServer struct {
	usersv1.UnimplementedUsersServiceServer

	mu   sync.Mutex
	last recordedCall
}

func (s *fakeUsersServer) record(ctx context.Context, method string, req any) {
	md, _ := metadata.FromIncomingContext(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.last = recordedCall{method: method, req: req, md: md}
}

func (s *fakeUsersServer) lastCall() recordedCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.last
}

func (s *fakeUsersServer) CreateUser(ctx context.Context, req *usersv1.CreateUserRequest) (*usersv1.CreateUserResponse, error) {
	s.record(ctx, "CreateUser", req)
	return &usersv1.CreateUserResponse{User: testUser("created")}, nil
}

func (s *fakeUsersServer) GetUser(ctx context.Context, req *usersv1.GetUserRequest) (*usersv1.GetUserResponse, error) {
	s.record(ctx, "GetUser", req)
	return &usersv1.GetUserResponse{User: testUser(req.GetId())}, nil
}

func (s *fakeUsersServer) ChangeMyPassword(ctx context.Context, req *usersv1.ChangeMyPasswordRequest) (*usersv1.ChangeMyPasswordResponse, error) {
	s.record(ctx, "ChangeMyPassword", req)
	return &usersv1.ChangeMyPasswordResponse{Success: true}, nil
}

func (s *fakeUsersServer) AdminChangePassword(ctx context.Context, req *usersv1.AdminChangePasswordRequest) (*usersv1.AdminChangePasswordResponse, error) {
	s.record(ctx, "AdminChangePassword", req)
	return &usersv1.AdminChangePasswordResponse{Success: true}, nil
}

func (s *fakeUsersServer) ChangeRole(ctx context.Context, req *usersv1.ChangeRoleRequest) (*usersv1.ChangeRoleResponse, error) {
	s.record(ctx, "ChangeRole", req)
	return &usersv1.ChangeRoleResponse{Success: true}, nil
}

func (s *fakeUsersServer) AssignPermission(ctx context.Context, req *usersv1.AssignPermissionRequest) (*usersv1.AssignPermissionResponse, error) {
	s.record(ctx, "AssignPermission", req)
	return &usersv1.AssignPermissionResponse{Success: true}, nil
}

func (s *fakeUsersServer) RevokePermission(ctx context.Context, req *usersv1.RevokePermissionRequest) (*usersv1.RevokePermissionResponse, error) {
	s.record(ctx, "RevokePermission", req)
	return &usersv1.RevokePermissionResponse{Success: true}, nil
}

func (s *fakeUsersServer) UpdateProfile(ctx context.Context, req *usersv1.UpdateProfileRequest) (*usersv1.UpdateProfileResponse, error) {
	s.record(ctx, "UpdateProfile", req)
	return &usersv1.UpdateProfileResponse{Success: true}, nil
}

func (s *fakeUsersServer) DeleteUser(ctx context.Context, req *usersv1.DeleteUserRequest) (*usersv1.DeleteUserResponse, error) {
	s.record(ctx, "DeleteUser", req)
	return &usersv1.DeleteUserResponse{Success: true}, nil
}

func (s *fakeUsersServer) ActivateUser(ctx context.Context, req *usersv1.ActivateUserRequest) (*usersv1.ActivateUserResponse, error) {
	s.record(ctx, "ActivateUser", req)
	return &usersv1.ActivateUserResponse{Success: true}, nil
}

func (s *fakeUsersServer) DeactivateUser(ctx context.Context, req *usersv1.DeactivateUserRequest) (*usersv1.DeactivateUserResponse, error) {
	s.record(ctx, "DeactivateUser", req)
	return &usersv1.DeactivateUserResponse{Success: true}, nil
}

func (s *fakeUsersServer) BanUser(ctx context.Context, req *usersv1.BanUserRequest) (*usersv1.BanUserResponse, error) {
	s.record(ctx, "BanUser", req)
	return &usersv1.BanUserResponse{Success: true}, nil
}

func (s *fakeUsersServer) UnbanUser(ctx context.Context, req *usersv1.UnbanUserRequest) (*usersv1.UnbanUserResponse, error) {
	s.record(ctx, "UnbanUser", req)
	return &usersv1.UnbanUserResponse{Success: true}, nil
}

func (s *fakeUsersServer) ListUsers(ctx context.Context, req *usersv1.ListUsersRequest) (*usersv1.ListUsersResponse, error) {
	s.record(ctx, "ListUsers", req)
	return &usersv1.ListUsersResponse{Users: []*usersv1.User{testUser("listed")}, Total: 1}, nil
}

func testUser(id string) *usersv1.User {
	return &usersv1.User{
		Id:       id,
		Username: "user-" + id,
		Role:     usersv1.Role_ROLE_ADMIN,
		IsActive: true,
		Permissions: []*usersv1.Permission{
			{
				Id:   "permission-1",
				Code: usersv1.PermissionCode_PERMISSION_USERS_READ,
				Name: "read users",
			},
		},
	}
}

func TestGatewayRoutesProxyToGRPCServices(t *testing.T) {
	t.Parallel()

	authSrv := &fakeAuthServer{}
	usersSrv := &fakeUsersServer{}
	httpServer := startGatewayTestServer(t, authSrv, usersSrv)

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantMethod string
		assert     func(*testing.T, recordedCall)
	}{
		{
			name:       "auth login",
			method:     http.MethodPost,
			path:       "/api/v1/auth/login",
			body:       `{"username":"alice","password":"secret"}`,
			wantMethod: "Login",
			assert: func(t *testing.T, call recordedCall) {
				req := call.req.(*authv1.LoginRequest)
				if req.GetUsername() != "alice" || req.GetPassword() != "secret" {
					t.Fatalf("Login request = %+v", req)
				}
			},
		},
		{
			name:       "auth refresh",
			method:     http.MethodPost,
			path:       "/api/v1/auth/refresh",
			body:       `{"refreshToken":"refresh-in"}`,
			wantMethod: "RefreshToken",
			assert: func(t *testing.T, call recordedCall) {
				if got := call.req.(*authv1.RefreshTokenRequest).GetRefreshToken(); got != "refresh-in" {
					t.Fatalf("RefreshToken = %q", got)
				}
			},
		},
		{
			name:       "auth logout",
			method:     http.MethodPost,
			path:       "/api/v1/auth/logout",
			body:       `{"refreshToken":"refresh-out"}`,
			wantMethod: "Logout",
			assert: func(t *testing.T, call recordedCall) {
				if got := call.req.(*authv1.LogoutRequest).GetRefreshToken(); got != "refresh-out" {
					t.Fatalf("Logout refresh token = %q", got)
				}
			},
		},
		{
			name:       "auth me",
			method:     http.MethodPost,
			path:       "/api/v1/auth/me",
			body:       `{}`,
			wantMethod: "GetMe",
			assert:     assertAuthorizationMetadata,
		},
		{
			name:       "users create",
			method:     http.MethodPost,
			path:       "/api/v1/users",
			body:       `{"username":"bob","password":"pw","role":"ROLE_ADMIN","isActive":true,"permissionCodes":["PERMISSION_USERS_READ","PERMISSION_USERS_CREATE"]}`,
			wantMethod: "CreateUser",
			assert: func(t *testing.T, call recordedCall) {
				req := call.req.(*usersv1.CreateUserRequest)
				if req.GetUsername() != "bob" || req.GetPassword() != "pw" || req.GetRole() != usersv1.Role_ROLE_ADMIN || !req.GetIsActive() {
					t.Fatalf("CreateUser request = %+v", req)
				}
				if got := req.GetPermissionCodes(); len(got) != 2 || got[0] != usersv1.PermissionCode_PERMISSION_USERS_READ || got[1] != usersv1.PermissionCode_PERMISSION_USERS_CREATE {
					t.Fatalf("CreateUser permission codes = %v", got)
				}
			},
		},
		{
			name:       "users get",
			method:     http.MethodGet,
			path:       "/api/v1/users/user-1",
			wantMethod: "GetUser",
			assert: func(t *testing.T, call recordedCall) {
				if got := call.req.(*usersv1.GetUserRequest).GetId(); got != "user-1" {
					t.Fatalf("GetUser id = %q", got)
				}
			},
		},
		{
			name:       "users me change password currently matches admin route",
			method:     http.MethodPost,
			path:       "/api/v1/users/me/change-password",
			body:       `{"id":"user-1","oldPassword":"old","newPassword":"new"}`,
			wantMethod: "AdminChangePassword",
			assert: func(t *testing.T, call recordedCall) {
				req := call.req.(*usersv1.AdminChangePasswordRequest)
				if req.GetId() != "me" || req.GetNewPassword() != "new" {
					t.Fatalf("AdminChangePassword request for /me/change-password = %+v", req)
				}
			},
		},
		{
			name:       "users admin change password",
			method:     http.MethodPost,
			path:       "/api/v1/users/user-1/change-password",
			body:       `{"newPassword":"new-admin"}`,
			wantMethod: "AdminChangePassword",
			assert: func(t *testing.T, call recordedCall) {
				req := call.req.(*usersv1.AdminChangePasswordRequest)
				if req.GetId() != "user-1" || req.GetNewPassword() != "new-admin" {
					t.Fatalf("AdminChangePassword request = %+v", req)
				}
			},
		},
		{
			name:       "users change role",
			method:     http.MethodPost,
			path:       "/api/v1/users/user-1/role",
			body:       `{"role":"ROLE_TEACHER"}`,
			wantMethod: "ChangeRole",
			assert: func(t *testing.T, call recordedCall) {
				req := call.req.(*usersv1.ChangeRoleRequest)
				if req.GetId() != "user-1" || req.GetRole() != usersv1.Role_ROLE_TEACHER {
					t.Fatalf("ChangeRole request = %+v", req)
				}
			},
		},
		{
			name:       "users assign permission",
			method:     http.MethodPost,
			path:       "/api/v1/users/user-1/permissions/assign",
			body:       `{"permission":"PERMISSION_USERS_BAN"}`,
			wantMethod: "AssignPermission",
			assert: func(t *testing.T, call recordedCall) {
				req := call.req.(*usersv1.AssignPermissionRequest)
				if req.GetId() != "user-1" || req.GetPermission() != usersv1.PermissionCode_PERMISSION_USERS_BAN {
					t.Fatalf("AssignPermission request = %+v", req)
				}
			},
		},
		{
			name:       "users revoke permission",
			method:     http.MethodPost,
			path:       "/api/v1/users/user-1/permissions/revoke",
			body:       `{"permission":"PERMISSION_USERS_BAN"}`,
			wantMethod: "RevokePermission",
			assert: func(t *testing.T, call recordedCall) {
				req := call.req.(*usersv1.RevokePermissionRequest)
				if req.GetId() != "user-1" || req.GetPermission() != usersv1.PermissionCode_PERMISSION_USERS_BAN {
					t.Fatalf("RevokePermission request = %+v", req)
				}
			},
		},
		{
			name:       "users update profile",
			method:     http.MethodPut,
			path:       "/api/v1/users/user-1/profile",
			body:       `{"username":"renamed"}`,
			wantMethod: "UpdateProfile",
			assert: func(t *testing.T, call recordedCall) {
				req := call.req.(*usersv1.UpdateProfileRequest)
				if req.GetId() != "user-1" || req.GetUsername() != "renamed" {
					t.Fatalf("UpdateProfile request = %+v", req)
				}
			},
		},
		{
			name:       "users delete",
			method:     http.MethodDelete,
			path:       "/api/v1/users/user-1",
			wantMethod: "DeleteUser",
			assert: func(t *testing.T, call recordedCall) {
				if got := call.req.(*usersv1.DeleteUserRequest).GetId(); got != "user-1" {
					t.Fatalf("DeleteUser id = %q", got)
				}
			},
		},
		{
			name:       "users activate",
			method:     http.MethodPost,
			path:       "/api/v1/users/user-1/activate",
			body:       `{}`,
			wantMethod: "ActivateUser",
			assert: func(t *testing.T, call recordedCall) {
				if got := call.req.(*usersv1.ActivateUserRequest).GetId(); got != "user-1" {
					t.Fatalf("ActivateUser id = %q", got)
				}
			},
		},
		{
			name:       "users deactivate",
			method:     http.MethodPost,
			path:       "/api/v1/users/user-1/deactivate",
			body:       `{}`,
			wantMethod: "DeactivateUser",
			assert: func(t *testing.T, call recordedCall) {
				if got := call.req.(*usersv1.DeactivateUserRequest).GetId(); got != "user-1" {
					t.Fatalf("DeactivateUser id = %q", got)
				}
			},
		},
		{
			name:       "users ban",
			method:     http.MethodPost,
			path:       "/api/v1/users/user-1/ban",
			body:       `{"reason":"policy"}`,
			wantMethod: "BanUser",
			assert: func(t *testing.T, call recordedCall) {
				req := call.req.(*usersv1.BanUserRequest)
				if req.GetId() != "user-1" || req.GetReason() != "policy" {
					t.Fatalf("BanUser request = %+v", req)
				}
			},
		},
		{
			name:       "users unban",
			method:     http.MethodPost,
			path:       "/api/v1/users/user-1/unban",
			body:       `{}`,
			wantMethod: "UnbanUser",
			assert: func(t *testing.T, call recordedCall) {
				if got := call.req.(*usersv1.UnbanUserRequest).GetId(); got != "user-1" {
					t.Fatalf("UnbanUser id = %q", got)
				}
			},
		},
		{
			name:       "users list",
			method:     http.MethodGet,
			path:       "/api/v1/users?limit=10&offset=5&role=ROLE_ADMIN&onlyActive=true",
			wantMethod: "ListUsers",
			assert: func(t *testing.T, call recordedCall) {
				req := call.req.(*usersv1.ListUsersRequest)
				if req.GetLimit() != 10 || req.GetOffset() != 5 || req.GetRole() != usersv1.Role_ROLE_ADMIN || !req.GetOnlyActive() {
					t.Fatalf("ListUsers request = %+v", req)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			respBody := doGatewayRequest(t, httpServer.URL, tt.method, tt.path, tt.body)
			if len(respBody) == 0 {
				t.Fatalf("empty JSON response")
			}

			var call recordedCall
			if tt.wantMethod == "Login" || tt.wantMethod == "RefreshToken" || tt.wantMethod == "Logout" || tt.wantMethod == "GetMe" {
				call = authSrv.lastCall()
			} else {
				call = usersSrv.lastCall()
			}
			if call.method != tt.wantMethod {
				t.Fatalf("gRPC method = %q, want %q", call.method, tt.wantMethod)
			}
			tt.assert(t, call)
		})
	}
}

func TestGatewayCORSPreflight(t *testing.T) {
	t.Parallel()

	httpServer := startGatewayTestServer(t, &fakeAuthServer{}, &fakeUsersServer{})

	req, err := http.NewRequest(http.MethodOptions, httpServer.URL+"/api/v1/auth/login", nil)
	if err != nil {
		t.Fatalf("NewRequest returned error: %v", err)
	}
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "authorization, content-type")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("preflight request returned error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
	if got := resp.Header.Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Access-Control-Allow-Credentials = %q", got)
	}
}

func startGatewayTestServer(t *testing.T, authSrv *fakeAuthServer, usersSrv *fakeUsersServer) *httptest.Server {
	t.Helper()

	authAddr, stopAuth := startGRPCServer(t, func(s *grpc.Server) {
		authv1.RegisterAuthServiceServer(s, authSrv)
	})
	t.Cleanup(stopAuth)

	usersAddr, stopUsers := startGRPCServer(t, func(s *grpc.Server) {
		usersv1.RegisterUsersServiceServer(s, usersSrv)
	})
	t.Cleanup(stopUsers)

	cfg := config.Config{
		AuthService: config.AuthServiceConfig{
			Addr: authAddr,
		},
		UsersService: config.UsersServiceConfig{
			Addr: usersAddr,
		},
	}

	handler, err := gatewayserver.NewHandler(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewHandler returned error: %v", err)
	}

	httpServer := httptest.NewServer(handler)
	t.Cleanup(httpServer.Close)

	return httpServer
}

func startGRPCServer(t *testing.T, register func(*grpc.Server)) (string, func()) {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen returned error: %v", err)
	}

	server := grpc.NewServer()
	register(server)

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = server.Serve(lis)
	}()

	return lis.Addr().String(), func() {
		server.Stop()
		<-done
	}
}

func doGatewayRequest(t *testing.T, baseURL, method, path, body string) map[string]any {
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
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s %s status = %d, body = %s", method, path, resp.StatusCode, string(data))
	}

	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("response is not JSON: %v; body = %s", err, string(data))
	}
	return out
}

func assertAuthorizationMetadata(t *testing.T, call recordedCall) {
	t.Helper()

	if got := call.md.Get("authorization"); len(got) != 1 || got[0] != "Bearer integration-token" {
		t.Fatalf("authorization metadata = %v, want %v", got, []string{"Bearer integration-token"})
	}
}
