package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gateway/internal/clients"
	"gateway/internal/config"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"
)

func main() {

	if err := run(); err != nil {
		log.Fatalf("gateway failed to run: %v", err)
	}
}

func run() error {

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	mux := runtime.NewServeMux()

	if err := clients.RegisterAuthHandler(ctx, mux, cfg.AuthService.Addr); err != nil {
		return err
	}

	if err := clients.RegisterUsersHandler(ctx, mux, cfg.UsersService.Addr); err != nil {
		return err
	}

	handler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}).Handler(mux)

	server := &http.Server{
		Addr:              cfg.Server.HttpAddr,
		Handler:           handler,
		ReadHeaderTimeout: cfg.Server.RHT,
	}

	log.Println("Gatewat started on :8080")
	log.Println("Proxing request to gRPC auth service:", cfg.AuthService.Addr)
	log.Println("Proxing request to gRPC users service:", cfg.UsersService.Addr)

	serverErrors := make(chan error, 1)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			serverErrors <- err
		}
	}()

	stop := make(chan os.Signal, 1)

	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("gateway server error: %w", err)
	case <-stop:
		log.Println("Shutting down...")

	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Println("graceful shutdown failed, forcing close...")
		if closeErr := server.Close(); closeErr != nil {
			return fmt.Errorf("failed to force close gateway server: %w", closeErr)
		}
	}

	log.Println("Gateway stopped...")

	return nil

}
