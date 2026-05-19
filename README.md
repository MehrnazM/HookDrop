# WebhookX

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
| Frontend | Next.js + TypeScript |
| Local Dev | Docker Compose |
| Production | Fly.io |

---

## Getting Started

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose
- Go 1.21+ (for local development outside Docker)
- Node.js 18+ and npm (for frontend development)

### 1. Clone the repo

```bash
git clone https://github.com/MehrnazM/WebhookX.git
cd WebhookX
```

### 2. Set up environment variables

```bash
cp .env.example .env
```

Edit `.env` with your local values. The key variables are:

| Variable | Description |
|----------|-------------|
| `DATABASE_URL` | PostgreSQL connection string |
| `REDIS_URL` | Redis connection string |
| `SESSION_SECRET` | Secret for signing session tokens |
| `NEXT_PUBLIC_API_URL` | Base URL for the Data API (used by the frontend) |

### 3. Start the backend services

```bash
docker-compose up --build
```

This starts:
- PostgreSQL on `localhost:5432`
- Redis on `localhost:6379`
- Go Ingestion on `localhost:8080`
- Go Processor + SSE on `localhost:8081`
- Go Data API on `localhost:8082`

### 4. Run the frontend locally

```bash
cd frontend
npm install
npm run dev
```

The Next.js dashboard will be available at `http://localhost:3000`.

### 5. Create a Drop

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

### 6. Send a test webhook

```bash
curl -X POST http://localhost:8080/drop/a3f9bc72 \
  -H "Content-Type: application/json" \
  -d '{"event": "test", "data": {"amount": 4999}}'
```

### 7. Inspect the event

```bash
curl -s http://localhost:8082/api/drops/a3f9bc72/events \
  -H "Authorization: Bearer your-token-here" | jq
```

### 8. Stream events in real time

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
