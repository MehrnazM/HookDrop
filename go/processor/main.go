package main

import (
	"context"
	"database/sql"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	util "github.com/mehrnazm/webhookx/go/util"
)

const (
	readTimeout     = 15 * time.Second
	writeTimeout    = 0
	idleTimeout     = 20 * time.Second
	shutdownTimeout = 20 * time.Second
)

func main() {
	logger := util.SetupLogger("processor")
	logger.Info("starting up processor service ...")
	// Read environment
	ctx, cancelBackgroundCtx := context.WithCancel(context.Background())
	defer cancelBackgroundCtx()

	port := util.GetStringEnv("PORT", "8082")

	dbURL, err := util.MustGetString("DATABASE_URL")
	if err != nil {
		logger.Error("DATABASE_URL missing")
		os.Exit(1)
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		logger.Error("failed to open database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		logger.Error("failed to ping database", "err", err)
		os.Exit(1)
	}
	logger.Info("✓ Connected to PostgreSQL")

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

	logger.Info("✓ Ready to process events")

	// HTTP server
	router := mux.NewRouter()
	addr := ":" + port
	server := util.NewServer(addr, readTimeout, writeTimeout, idleTimeout, router)

	sseServer := NewSSEServer(redisClient, logger)

	apiClient := NewPostgresDataAPIClient(db, logger)
	handler := NewHandler(apiClient, sseServer, logger)

	router.HandleFunc("/healthz", Health).Methods("GET")
	router.HandleFunc("/api/drops/{url_slug}/stream", handler.HandleSSEStream).Methods("GET")

	go sseServer.ListenToRedis(ctx)
	go ConsumeRedisStreams(ctx, redisClient, apiClient, sseServer, logger)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				for _, slug := range sseServer.ActiveDropSlugs() {
					if !apiClient.DropExists(ctx, slug) {
						sseServer.EvictDrop(slug)
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	logger.Info("✓ Listening on :" + port)

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

		// Cancel background context to stop all goroutines
		cancelBackgroundCtx()

		// Give goroutines time to exit gracefully
		time.Sleep(1 * time.Second)

		// Graceful HTTP shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
			os.Exit(1)
		}
		logger.Info("shutdown complete")
	}
}
