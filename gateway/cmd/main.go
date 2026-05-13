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

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/rs/cors"
)

func main() {

	if err := run(); err != nil {
		log.Fatalf("gateway failed to run: %v", err)
	}
	//TODO: Инициализировать методы микросервисов.
	//TODO: Написать тесты для HTTP Gateway.
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	mux := runtime.NewServeMux()

	if err := clients.RegisterAuthHandler(ctx, mux, authServiceAddr); err != nil {
		return err
	}

	handler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}).Handler(mux)

	server := &http.Server{
		Addr:              httpAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Println("Gatewat started on :8080")
	log.Println("Proxing request to gRPC auth service:", authServiceAddr)
	log.Println("Proxing request to gRPC users service:", usersServiceAddr)

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
