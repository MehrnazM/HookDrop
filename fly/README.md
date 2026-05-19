# WebhookX — Fly.io Deployment Guide

## Prerequisites

- [fly CLI](https://fly.io/docs/hands-on/install-flyctl/) installed
- Authenticated: `fly auth login`
- Run all `fly` commands from the **project root** (the directory containing `docker-compose.yml`)

---

## 1. Create the Fly apps

```bash
fly apps create webhookx-ingestion
fly apps create webhookx-processor
fly apps create webhookx-data-api
fly apps create webhookx-frontend
```

---

## 2. Provision Postgres

```bash
fly postgres create --name webhookx-pg --region iad
```

Attach to the services that need the database:

```bash
fly postgres attach webhookx-pg --app webhookx-data-api
fly postgres attach webhookx-pg --app webhookx-processor
```

Attaching automatically sets `DATABASE_URL` as a secret on each app. Run the migrations manually after the first deploy:

```bash
fly ssh console --app webhookx-data-api
# inside the machine, run your migrate command against DATABASE_URL
```

---

## 3. Provision Redis

```bash
fly redis create --name webhookx-redis --region iad
```

Copy the `redis://` connection string shown after creation, then set it as a secret on every app that needs it (see step 4).

---

## 4. Set secrets

**webhookx-data-api** (DATABASE_URL is already set by `postgres attach`):
```bash
fly secrets set --app webhookx-data-api \
  REDIS_URL=redis://<your-redis-url>
```

**webhookx-processor** (DATABASE_URL is already set by `postgres attach`):
```bash
fly secrets set --app webhookx-processor \
  REDIS_URL=redis://<your-redis-url>
```

**webhookx-ingestion**:
```bash
fly secrets set --app webhookx-ingestion \
  REDIS_URL=redis://<your-redis-url>
```

**webhookx-frontend**:
```bash
fly secrets set --app webhookx-frontend \
  NEXT_PUBLIC_API_URL=https://webhookx-data-api.fly.dev
```

---

## 5. Deploy (in order)

Deploy data-api first (other services depend on the database being migrated), then processor, then ingestion, then frontend.

```bash
fly deploy --config fly/data-api/fly.toml
fly deploy --config fly/processor/fly.toml
fly deploy --config fly/ingestion/fly.toml
fly deploy --config fly/frontend/fly.toml
```

---

## Service URLs

| Service     | URL                                      | Internal port |
|-------------|------------------------------------------|---------------|
| ingestion   | https://webhookx-ingestion.fly.dev       | 8080          |
| data-api    | https://webhookx-data-api.fly.dev        | 8081          |
| processor   | internal worker (no public URL)          | 8082          |
| frontend    | https://webhookx-frontend.fly.dev        | 3000          |

---

## Useful commands

```bash
# View logs
fly logs --app webhookx-ingestion

# SSH into a machine
fly ssh console --app webhookx-data-api

# Scale machines
fly scale count 2 --app webhookx-ingestion

# Check machine status
fly status --app webhookx-processor
```
