package clients

import (
	"context"
	usersv1 "contracts/gen/go/proto/users/v1"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func RegisterUsersHandler(ctx context.Context, mux *runtime.ServeMux, addr string) error {
	if err := usersv1.RegisterUsersServiceHandlerFromEndpoint(ctx, mux, addr, []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}); err != nil {
		return fmt.Errorf("failed to register users service handler: %w", err)
	}

	return nil
}
