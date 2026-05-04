package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Read environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	dbURL := os.Getenv("DATABASE_URL")
	redisURL := os.Getenv("REDIS_URL")

	log.Printf("=== Ingestion Service Starting ===")
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

	// HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("POST /drop/:url-slug", handleWebhook(rc))
	mux.HandleFunc("/health", handleHealth)

	log.Printf("✓ Listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func handleWebhook(rc *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("url-slug")
		log.Printf("slug [%s]\n", slug)
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		log.Printf("[%s] %s %s (body: %d bytes)", r.Method, r.URL.Path, r.RemoteAddr, len(body))

		// TODO: Validate drop exists, enqueue to Redis
		w.WriteHeader(http.StatusOK)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
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
