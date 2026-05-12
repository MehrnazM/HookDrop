package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	util "github.com/mehrnazm/webhookx/go/util"
)

const (
	readTimeout     = 15 * time.Second
	writeTimeout    = 15 * time.Second
	idleTimeout     = 20 * time.Second
	shutdownTimeout = 20 * time.Second
)

func main() {
	logger := util.SetupLogger("ingestion")
	logger.Info("starting up ingestion service ...")
	ctx, cancelBackgroundCtx := context.WithCancel(context.Background())
	defer cancelBackgroundCtx()

	// Read environment
	port := util.GetStringEnv("PORT", "8080")

	// Redis client
	redisClient, err := util.InitRedisClient()
	if err != nil {
		logger.Error("failed to initialize redis client", "err", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error("Failed to ping Redis", "err", err)
		os.Exit(1)
	}
	logger.Info("✓ Connected to Redis")

	// HTTP server
	router := mux.NewRouter()
	addr := ":" + port
	server := util.NewServer(addr, readTimeout, writeTimeout, idleTimeout, router)

	handler := NewHandler(redisClient, logger)
	router.HandleFunc("/drop/{url_slug}", handler.PublicDropPost).Methods("POST", "GET", "PUT", "PATCH", "DELETE")
	router.HandleFunc("/healthz", Health)

	// Start HTTP server
	serverErrors := make(chan error, 1)
	go func() {
		if err := server.Start(); err != nil {
			serverErrors <- err
		}
	}()

	// Wait for shutdown signal or server error
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	select {
	case err := <-serverErrors:
		logger.Error("server failed", "error", err)
		os.Exit(1)

	case sig := <-shutdown:
		logger.Info("shutdown initiated", "signal", sig)

		// Graceful HTTP shutdown
		shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
			os.Exit(1)
		}

		logger.Info("shutdown complete")
	}
}
