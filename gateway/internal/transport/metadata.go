package transport

import (
	"context"
	"net/http"

	"google.golang.org/grpc/metadata"
)

func HTTPHeadersToMetadata(ctx context.Context, r *http.Request) metadata.MD {
	md := metadata.MD{}
	authorization := r.Header.Get("Authorization")

	if authorization != "" {
		md.Set("Authorization", authorization)
	}

	return md
}
