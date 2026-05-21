package main

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	dbpkg "github.com/mehrnazm/hookdrop/go/data-api/db"
	"github.com/mehrnazm/hookdrop/go/data-api/handlers"
	"github.com/mehrnazm/hookdrop/go/data-api/middleware"
	util "github.com/mehrnazm/hookdrop/go/util"
)

const (
	readTimeout     = 15 * time.Second
	writeTimeout    = 15 * time.Second
	idleTimeout     = 20 * time.Second
	shutdownTimeout = 20 * time.Second
)

func runMigrate(logger interface {
	Info(string, ...any)
	Error(string, ...any)
}) {
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

	if err := db.PingContext(context.Background()); err != nil {
		logger.Error("failed to ping database", "err", err)
		os.Exit(1)
	}

	matches, err := filepath.Glob("postgres/migrations/*.up.sql")
	if err != nil {
		logger.Error("failed to glob migrations", "err", err)
		os.Exit(1)
	}
	sort.Strings(matches)

	for _, path := range matches {
		name := filepath.Base(path)
		logger.Info("applying migration", "file", name)
		contents, err := os.ReadFile(path)
		if err != nil {
			logger.Error("failed to read migration", "file", name, "err", err)
			os.Exit(1)
		}
		if _, err := db.ExecContext(context.Background(), string(contents)); err != nil {
			logger.Error("migration failed", "file", name, "err", err)
			os.Exit(1)
		}
		logger.Info("migration applied", "file", name)
	}

	logger.Info("all migrations applied", "count", len(matches))
}

func main() {
	logger := util.SetupLogger("data-api")

	if len(os.Args) > 1 && strings.ToLower(os.Args[1]) == "migrate" {
		logger.Info("running migrations ...")
		runMigrate(logger)
		return
	}

	logger.Info("starting up data-api service ...")

	ctx, cancelBackgroundCtx := context.WithCancel(context.Background())
	defer cancelBackgroundCtx()

	port := util.GetStringEnv("DATA_API_PORT", util.GetStringEnv("PORT", "8081"))

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

	if err := db.PingContext(ctx); err != nil {
		logger.Error("failed to ping database", "err", err)
		os.Exit(1)
	}
	logger.Info("✓ Connected to PostgreSQL")

	redisClient, err := util.InitRedisClient()
	if err != nil {
		logger.Error("failed to initialize redis client", "err", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error("failed to ping redis", "err", err)
		os.Exit(1)
	}
	logger.Info("✓ Connected to Redis")

	router := gin.New()
	router.Use(gin.Recovery())

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "https://frontend-production-7f6d.up.railway.app"},
		AllowMethods:     []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	dbQueries := dbpkg.NewQueries(db)
	h := handlers.NewHandler(dbQueries, logger, redisClient)

	api := router.Group("/api")
	drops := api.Group("/drops")
	drops.POST("", h.CreateDrop)

	authed := drops.Group("/:drop_slug")
	authed.Use(middleware.Authenticate(dbQueries, logger))
	authed.GET("", h.GetDrop)
	authed.DELETE("", h.DeleteDrop)
	authed.GET("/events", h.ListEvents)
	authed.GET("/events/:event_id", h.GetEvent)

	addr := ":" + port
	server := util.NewServer(addr, readTimeout, writeTimeout, idleTimeout, router)

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("✓ Listening on " + addr)
		if err := server.Start(); err != nil {
			serverErrors <- err
		}
	}()

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
