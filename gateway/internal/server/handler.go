package server

import (
	"context"
	"fmt"
	"net/http"

	"gateway/internal/clients"
	"gateway/internal/config"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"
)

func NewHandler(ctx context.Context, cfg config.Config) (http.Handler, error) {
	corsConfig := cfg.CORS.WithDefaults()
	if err := corsConfig.Validate(); err != nil {
		return nil, fmt.Errorf("validate cors config: %w", err)
	}

	mux := runtime.NewServeMux()

	if err := clients.RegisterAuthHandler(ctx, mux, cfg.AuthService.Addr); err != nil {
		return nil, fmt.Errorf("register auth handler: %w", err)
	}

	if err := clients.RegisterUsersHandler(ctx, mux, cfg.UsersService.Addr); err != nil {
		return nil, fmt.Errorf("register users handler: %w", err)
	}

	handler := cors.New(cors.Options{
		AllowedOrigins:   corsConfig.AllowedOrigins,
		AllowedMethods:   corsConfig.AllowedMethods,
		AllowedHeaders:   corsConfig.AllowedHeaders,
		AllowCredentials: true,
	}).Handler(mux)

	return handler, nil
}
