# WebhookX

> 🚧 **This project is actively under development.** Features and structure may change as new phases are completed.

A developer-first webhook debugger built for the full dev-to-prod lifecycle.

WebhookX gives you an instant, anonymous webhook receiver — called a **Drop** — with zero signup required. Point any third-party service (Stripe, GitHub, Shopify, Twilio) at your Drop URL and inspect every request in real time.

---

## What is a Drop?

A Drop is a unique, temporary webhook endpoint that captures everything sent to it:

- Generated instantly — no account needed
- Lives for 24 hours
- Receives any HTTP method, any headers, any payload up to 1MB
- Streams incoming requests to your browser in real time via SSE
- Lets you inspect request body, headers, query params, and the response WebhookX sent back

---

## Architecture

WebhookX is built as four independent services:

```
Third-party service (Stripe, GitHub, etc.)
        │
        ▼
┌─────────────────────┐
│   Go Ingestion      │  :8080 — public webhook receiver
│   net/http + mux    │  rate-limits, validates, enqueues
└────────┬────────────┘
         │ Redis Streams
         ▼
┌─────────────────────┐       ┌──────────────────────┐
│  Go Processor + SSE │  ───▶ │  PostgreSQL + JSONB   │
│  Gin  :8081         │       │  drops                │
│  Consumes queue     │       │  webhook_events       │
│  Broadcasts via SSE │       │  drop_responses       │
└────────┬────────────┘       └──────────────────────┘
         │ Redis Pub/Sub
         ▼
┌─────────────────────┐       ┌──────────────────────┐
│   Browser (SSE)     │       │   Go Data API        │
│   Drop Inspector    │  ◀──  │   Gin  :8082         │
│   Real-time updates │       │   REST endpoints     │
└─────────────────────┘       └──────────────────────┘
         ▲
         │ HTTP (JSON)
┌─────────────────────┐
│  Next.js Frontend   │  :3000 — dashboard UI (Phase 4)
└─────────────────────┘
```

**Key design decisions:**
- **Queue-first ingestion** — responds in <100ms regardless of downstream load
- **Redis Streams** — durable, ordered event log; survives processor downtime
- **SSE over WebSockets** — browser only receives; simpler and correct for one-way push
- **DataAPIClient interface** — Processor uses a stub (direct DB) in V1, swappable for HTTP client in V2 with zero refactoring
- **Direct SQL, no ORM** — full control, no abstraction overhead
- **JSONB storage** — flexible payload storage, enables V2 search without a separate engine

---

## Tech Stack

| Layer | Technology |
|-------|------------|
| Ingestion | Go + net/http + gorilla/mux |
| Processor + SSE | Go + Gin |
| Data API | Go + Gin |
| Shared Utilities | go/util (separate module) |
| Queue | Redis Streams |
| Database | PostgreSQL + JSONB |
| Real-time | Server-Sent Events (SSE) |
| Frontend | Next.js + TypeScript *(Phase 4)* |
| Local Dev | Docker Compose |
| Production | Fly.io *(planned)* |

---

## Project Structure

```
webhookx/
├── docker/                    # Dockerfiles for each service
├── go/
│   ├── util/                  # Shared module: logger, Redis client, models, errors
│   ├── ingestion/             # Public webhook receiver (port 8080)
│   ├── processor/             # Queue consumer + SSE broadcaster (port 8081)
│   └── data-api/              # REST API for dashboard (port 8082)
├── postgres/
│   └── migrations/
│       └── 001_init.sql       # drops, webhook_events, drop_responses tables
├── frontend/                  # Next.js dashboard (Phase 4)
├── docker-compose.yml
└── .env.example
```

---

## Getting Started

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose
- Go 1.21+ (for local development outside Docker)

### 1. Clone the repo

```bash
git clone https://github.com/MehrnazM/WebhookX.git
cd WebhookX
```

### 2. Set up environment variables

```bash
cp .env.example .env
```

### 3. Start all services

```bash
docker-compose up --build
```

This starts:
- PostgreSQL on `localhost:5432`
- Redis on `localhost:6379`
- Go Ingestion on `localhost:8080`
- Go Processor + SSE on `localhost:8081`
- Go Data API on `localhost:8082`

### 4. Create a Drop

```bash
curl -s -X POST http://localhost:8082/api/drops | jq
```

Response:
```json
{
  "url_slug": "a3f9bc72",
  "session_token": "your-token-here",
  "expires_at": "2026-05-02T14:32:07Z"
}
```

### 5. Send a test webhook

```bash
curl -X POST http://localhost:8080/drop/a3f9bc72 \
  -H "Content-Type: application/json" \
  -d '{"event": "test", "data": {"amount": 4999}}'
```

### 6. Inspect the event

```bash
curl -s http://localhost:8082/api/drops/a3f9bc72/events \
  -H "Authorization: Bearer your-token-here" | jq
```

### 7. Stream events in real time

```bash
curl -N http://localhost:8081/api/drops/a3f9bc72/stream \
  -H "Authorization: Bearer your-token-here"
```

---

## API Reference

### Go Ingestion (port 8080)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `ANY` | `/drop/:url_slug` | None | Receive a webhook. Always returns `200 OK`. Rate-limited at 1 req/s, burst 20. |

### Go Data API (port 8082)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/api/drops` | None | Create a new anonymous Drop |
| `GET` | `/api/drops/:slug` | Bearer token | Get Drop metadata and event count |
| `DELETE` | `/api/drops/:slug` | Bearer token | Delete Drop and all events |
| `GET` | `/api/drops/:slug/events` | Bearer token | List events (paginated, default 20, max 100) |
| `GET` | `/api/drops/:slug/events/:event_id` | Bearer token | Get full event detail |

### Go Processor + SSE (port 8081)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/api/drops/:slug/stream` | Bearer token | SSE stream — real-time webhook events |

---

## Running Tests

### Ingestion (unit tests — no infrastructure needed)

```bash
cd go/ingestion && go test ./...
```

### Processor (unit tests — no infrastructure needed)

```bash
cd go/processor && go test ./...
```

### Data API (unit tests)

```bash
cd go/data-api && go test ./handlers/... ./db/...
```

### Data API (integration tests — requires PostgreSQL and Redis)

```bash
export DATABASE_URL=postgres://webhookx:password@localhost:5432/webhookx_dev
export REDIS_URL=redis://localhost:6379
cd go/data-api && go test ./...
```

Integration tests are automatically skipped if `DATABASE_URL` or `REDIS_URL` are not set.

---

## Development Status

| Phase | Description | Status |
|-------|-------------|--------|
| Phase 1 | Docker Compose + PostgreSQL schema + migrations | ✅ Complete |
| Phase 2 | Go Ingestion service (net/http, rate limiting, Redis queue) | ✅ Complete |
| Phase 3 | Go Processor + SSE + Go Data API | ✅ Complete |
| Phase 4 | Next.js frontend — Drop Inspector UI | 🚧 In Progress |
| Phase 5 | Landing page + Fly.io production deployment | ⏳ Planned |

---

## Roadmap (V2)

- [ ] Replay webhook to a target URL
- [ ] Custom response codes and body
- [ ] Local tunnel CLI (forward to localhost)
- [ ] Multiple Drops per user (requires accounts)
- [ ] Team sharing and collaboration
- [ ] Longer history retention (30–90 days)
- [ ] Webhook signature verification (Stripe, GitHub)
- [ ] Redis-backed rate limiting (multi-instance)
- [ ] Dead-letter queue for failed events

---

## License

MIT
