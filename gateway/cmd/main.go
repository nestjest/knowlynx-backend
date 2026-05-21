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

	"gateway/internal/config"
	"gateway/internal/server"
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

	handler, err := server.NewHandler(ctx, *cfg)
	if err != nil {
		return fmt.Errorf("build gateway handler: %w", err)
	}

	httpServer := &http.Server{
		Addr:              cfg.Server.HttpAddr,
		Handler:           handler,
		ReadHeaderTimeout: cfg.Server.RHT,
	}

	log.Println("Gatewat started on :8080")
	log.Println("Proxing request to gRPC auth service:", cfg.AuthService.Addr)
	log.Println("Proxing request to gRPC users service:", cfg.UsersService.Addr)

	serverErrors := make(chan error, 1)

	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
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

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Println("graceful shutdown failed, forcing close...")
		if closeErr := httpServer.Close(); closeErr != nil {
			return fmt.Errorf("failed to force close gateway server: %w", closeErr)
		}
	}

	log.Println("Gateway stopped...")

	return nil

}
