package main

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
)

type SSEClient struct {
	DropSlug string
	Channel  chan []byte // buffered channel (size 10)
	Done     chan string // closed on disconnect
}

type SSEServer struct {
	clients map[string]map[*SSEClient]bool // drop_slug → set of clients
	mu      sync.RWMutex
	rdb     *redis.Client
	logger  *slog.Logger
}

func NewSSEServer(rdb *redis.Client, logger *slog.Logger) *SSEServer {
	return &SSEServer{
		clients: make(map[string]map[*SSEClient]bool),
		rdb:     rdb,
		logger:  logger,
	}
}

func (s *SSEServer) ActiveDropSlugs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	slugs := make([]string, 0, len(s.clients))
	for slug := range s.clients {
		slugs = append(slugs, slug)
	}
	return slugs
}

func (s *SSEServer) EvictDrop(dropSlug string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for client := range s.clients[dropSlug] {
		select {
		case client.Done <- "drop_expired":
		default:
		}
	}
	delete(s.clients, dropSlug)
}

func (s *SSEServer) RegisterClient(client *SSEClient) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dropSlug := client.DropSlug
	if s.clients[dropSlug] == nil {
		s.clients[dropSlug] = make(map[*SSEClient]bool)
	}
	s.clients[dropSlug][client] = true
	s.logger.Info("sse client registered", "drop_slug", dropSlug)
}

func (s *SSEServer) UnregisterClient(client *SSEClient) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dropSlug := client.DropSlug
	if s.clients[dropSlug] == nil {
		return // already evicted
	}
	delete(s.clients[dropSlug], client)
	if len(s.clients[dropSlug]) == 0 {
		delete(s.clients, dropSlug)
	}
	s.logger.Info("sse client unregistered", "drop_slug", dropSlug)
}

func (s *SSEServer) BroadcastToClients(dropSlug string, data []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for client := range s.clients[dropSlug] {
		select {
		case client.Channel <- data:
		default:
			s.logger.Warn("sse client channel full, skipping", "drop_slug", dropSlug)
		}
	}
}

func (s *SSEServer) ListenToRedis(ctx context.Context) {
	pubsub := s.rdb.PSubscribe(ctx, "drops:*:updates")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			// Extract drop_slug from channel name "drops:{drop_slug}:updates"
			dropSlug := strings.TrimPrefix(msg.Channel, "drops:")
			dropSlug = strings.TrimSuffix(dropSlug, ":updates")
			s.BroadcastToClients(dropSlug, []byte(msg.Payload))
		case <-ctx.Done():
			return
		}
	}
}
