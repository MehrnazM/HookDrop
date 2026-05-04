package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Read environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	dbURL := os.Getenv("DATABASE_URL")
	redisURL := os.Getenv("REDIS_URL")

	log.Printf("=== Data API Service Starting ===")
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

	// Gin router
	router := gin.Default()
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	log.Printf("✓ Listening on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Server error: %v", err)
	}
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
