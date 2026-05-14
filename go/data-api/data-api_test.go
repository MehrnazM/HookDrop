package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	dbpkg "github.com/mehrnazm/webhookx/go/data-api/db"
	"github.com/mehrnazm/webhookx/go/data-api/handlers"
	"github.com/mehrnazm/webhookx/go/data-api/middleware"
)

func setupRouter(db *sql.DB, rc *redis.Client) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(cors.Default())

	queries := dbpkg.NewQueries(db)
	logger := slog.Default()
	h := handlers.NewHandler(queries, logger, rc)

	api := r.Group("/api")
	drops := api.Group("/drops")
	drops.POST("", h.CreateDrop)

	authed := drops.Group("/:drop_slug")
	authed.Use(middleware.Authenticate(queries, logger))
	authed.GET("", h.GetDrop)
	authed.DELETE("", h.DeleteDrop)
	authed.GET("/events", h.ListEvents)
	authed.GET("/events/:event_id", h.GetEvent)

	return r
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set — skipping integration test")
	}
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func openTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		t.Skip("REDIS_URL not set — skipping integration test")
	}
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatalf("parse redis url: %v", err)
	}
	rc := redis.NewClient(opt)
	if err := rc.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("ping redis: %v", err)
	}
	t.Cleanup(func() { rc.Close() })
	return rc
}

func TestCreateDrop(t *testing.T) {
	db := openTestDB(t)
	rc := openTestRedis(t)
	r := setupRouter(db, rc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/drops", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body["url_slug"].(string)) != 8 {
		t.Errorf("expected 8-char url_slug, got %q", body["url_slug"])
	}
	if len(body["session_token"].(string)) != 64 {
		t.Errorf("expected 64-char session_token, got len=%d", len(body["session_token"].(string)))
	}
	if body["expires_at"] == nil {
		t.Error("expected expires_at in response")
	}
}

func TestGetDrop_Unauthorized(t *testing.T) {
	db := openTestDB(t)
	rc := openTestRedis(t)
	r := setupRouter(db, rc)

	// Create a drop first
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/drops", nil))
	if w.Code != http.StatusCreated {
		t.Fatalf("create drop: %d", w.Code)
	}
	var created map[string]interface{}
	json.NewDecoder(w.Body).Decode(&created)
	dropSlug := created["url_slug"].(string)

	// No token
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/api/drops/"+dropSlug, nil))
	if w2.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w2.Code)
	}

	// Wrong token
	w3 := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/drops/"+dropSlug, nil)
	req.Header.Set("Authorization", "Bearer "+"0000000000000000000000000000000000000000000000000000000000000000")
	r.ServeHTTP(w3, req)
	if w3.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong token, got %d", w3.Code)
	}
}

func TestGetDrop_Authorized(t *testing.T) {
	db := openTestDB(t)
	rc := openTestRedis(t)
	r := setupRouter(db, rc)

	// Create
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/drops", nil))
	var created map[string]interface{}
	json.NewDecoder(w.Body).Decode(&created)
	dropSlug := created["url_slug"].(string)
	token := created["session_token"].(string)

	// Get with valid token
	w2 := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/drops/"+dropSlug, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w2, req)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var body map[string]interface{}
	json.NewDecoder(w2.Body).Decode(&body)
	if body["url_slug"] != dropSlug {
		t.Errorf("expected url_slug=%s, got %v", dropSlug, body["url_slug"])
	}
	if _, ok := body["event_count"]; !ok {
		t.Error("expected event_count in response")
	}
}

func TestDeleteDrop(t *testing.T) {
	db := openTestDB(t)
	rc := openTestRedis(t)
	r := setupRouter(db, rc)

	// Create
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/drops", nil))
	var created map[string]interface{}
	json.NewDecoder(w.Body).Decode(&created)
	dropSlug := created["url_slug"].(string)
	token := created["session_token"].(string)

	// Delete
	w2 := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/drops/"+dropSlug, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w2, req)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var body map[string]interface{}
	json.NewDecoder(w2.Body).Decode(&body)
	if body["deleted"] != true {
		t.Errorf("expected deleted=true, got %v", body["deleted"])
	}

	// Subsequent GET should 404
	w3 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/drops/"+dropSlug, nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w3, req2)
	if w3.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", w3.Code)
	}
}

func TestListEvents_Empty(t *testing.T) {
	db := openTestDB(t)
	rc := openTestRedis(t)
	r := setupRouter(db, rc)

	// Create drop
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/drops", nil))
	var created map[string]interface{}
	json.NewDecoder(w.Body).Decode(&created)
	dropSlug := created["url_slug"].(string)
	token := created["session_token"].(string)

	// List events (should be empty)
	w2 := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/drops/"+dropSlug+"/events", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w2, req)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var body map[string]interface{}
	json.NewDecoder(w2.Body).Decode(&body)
	if body["total_count"].(float64) != 0 {
		t.Errorf("expected total_count=0, got %v", body["total_count"])
	}
	events := body["events"].([]interface{})
	if len(events) != 0 {
		t.Errorf("expected empty events, got %d", len(events))
	}
}
