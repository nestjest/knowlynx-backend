package clients

import (
	"context"
	"fmt"

	authv1 "contracts/gen/go/proto/auth/v1"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func RegisterAuthHandler(ctx context.Context, mux *runtime.ServeMux, addr string) error {
	if err := authv1.RegisterAuthServiceHandlerFromEndpoint(ctx, mux, addr, []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}); err != nil {
		return fmt.Errorf("failed to register auth service handler: %w", err)
	}

	return nil
}
