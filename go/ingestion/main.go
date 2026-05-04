package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	util "github.com/mehrnazm/webhookx/go/util"
	"github.com/redis/go-redis/v9"
)

var logger *slog.Logger
var redisClient *redis.Client

func main() {
	logger = util.SetupLogger("ingestion")
	logger.Info("starting up ingestion service ...")

	// Read environment
	port := util.GetStringEnv("PORT", "8080")
	dbURL, err := util.MustGetString("DATABASE_URL")
	if err != nil {
		logger.Error("DATABASE_URL missing")
		os.Exit(1)
	}

	// Database connection
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		logger.Error("Failed to open database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		logger.Error("Failed to ping database", "err", err)
		os.Exit(1)
	}
	logger.Info("✓ Connected to PostgreSQL")

	// Redis client
	redisClient, err = util.InitRedisClient()
	if err != nil {
		logger.Error("failed to initialize redis client", "err", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		logger.Error("Failed to ping Redis", "err", err)
		os.Exit(1)
	}
	logger.Info("✓ Connected to Redis")

	// HTTP server
	router := mux.NewRouter()
	router.HandleFunc("/drop/{url_slug}", PublicDropPost).Methods("POST", "GET", "PUT", "PATCH", "DELETE")
	router.HandleFunc("/healthz", Health)

	logger.Info("✓ Listening on :" + port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		logger.Error("server error", "err", err)
		os.Exit(1)
	}
}
