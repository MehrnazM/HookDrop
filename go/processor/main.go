package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Read environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}
	dbURL := os.Getenv("DATABASE_URL")
	redisURL := os.Getenv("REDIS_URL")

	log.Printf("=== Processor Service Starting ===")
	log.Printf("PORT: %s", port)
	log.Printf("DATABASE_URL: %s", maskURL(dbURL))
	log.Printf("REDIS_URL: %s", maskURL(redisURL))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test database connection
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("✓ Connected to PostgreSQL")

	// Test Redis connection
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}
	rc := redis.NewClient(opt)
	if err := rc.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to ping Redis: %v", err)
	}
	log.Println("✓ Connected to Redis")

	log.Println("✓ Ready to process events")

	// HTTP server for SSE
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})
	// TODO: GET /api/drops/:drop_id/stream (SSE endpoint)

	// Background processor loop
	go func() {
		for {
			// TODO: Read from Redis Streams, process events, store in database
			time.Sleep(1 * time.Second)
		}
	}()

	// Handle shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("✓ Listening on :%s", port)
	go func() {
		if err := http.ListenAndServe(":"+port, mux); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	sig := <-quit
	log.Printf("Received signal: %v, shutting down...", sig)

	rc.Close()
	db.Close()
	log.Println("✓ Processor stopped")
}

func maskURL(url string) string {
	if url == "" {
		return "(not set)"
	}
	if len(url) > 40 {
		return url[:40] + "..."
	}
	return url
}
