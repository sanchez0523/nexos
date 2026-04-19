# Nexos — Architecture

## Navigation

- **Architecture rules, constraints, conventions** → this file (ARCHITECTURE.md)
- **What to build and in what order** → [ROADMAP.md](ROADMAP.md)

---

## Architecture Invariants

These must NEVER change without explicit user approval. Violating any of these breaks the core design.

- **No Redis.** TimescaleDB handles both storage and realtime via WebSocket from Go. Adding Redis reintroduces complexity we deliberately removed.
- **Monitoring only (unidirectional).** No MQTT publish from server to device. No control commands. Ever.
- **Single admin.** No multi-tenancy, no role system, no team management.
- **JSON payloads only.** `{"value": 23.5}` or bare numeric. No Protobuf, no binary, no CSV.
- **Topic convention enforced.** Only `devices/{device_id}/{sensor}` is valid. Reject or ignore other formats.
- **5 services max.** Mosquitto, Go+Fiber, TimescaleDB, SvelteKit, Caddy. Do not add services without explicit approval.

---

## Decision Log (ADR)

### ADR-001: Go + Fiber over FastAPI
- **Decided:** Go + Fiber
- **Why:** Goroutines make MQTT subscription + HTTP API + WebSocket natural concurrency. Single binary → tiny Docker image. `paho.mqtt.golang` is mature.
- **Rejected:** FastAPI (Python — no perf bottleneck at this scale but Go is cleaner for concurrent I/O), Bun+Hono (MQTT ecosystem immature)

### ADR-002: No Redis
- **Decided:** Direct Go channel from MQTT subscriber → WebSocket broadcaster
- **Why:** Sub-50 devices, some message loss acceptable. Redis Pub/Sub adds an ops surface for zero gain at this scale.
- **Rejected:** Redis Pub/Sub, Redis Streams, Kafka

### ADR-003: Single TimescaleDB table
- **Decided:** One `metrics` table with `(time, device_id, sensor, value)`
- **Why:** Topics are discovered dynamically. Per-topic tables require dynamic DDL — operational nightmare.
- **Rejected:** Per-topic tables, InfluxDB, VictoriaMetrics

### ADR-004: Topic convention `devices/{device_id}/{sensor}`
- **Decided:** Enforced at ingestion layer, invalid topics are silently dropped
- **Why:** Auto-Discovery requires structured topic parsing. Free-form topics make device_id extraction ambiguous.
- **Rejected:** Free-form topics, prefix-only convention

### ADR-005: Caddy as reverse proxy
- **Decided:** Caddy
- **Why:** Automatic TLS cert generation/renewal out of the box. Zero config for localhost HTTPS.
- **Rejected:** nginx (manual cert management), Traefik (overkill)

### ADR-006: gridstack.js for dashboard layout
- **Decided:** gridstack.js
- **Why:** Grid-specific drag-and-drop purpose-built for dashboard UIs. Integrates with SvelteKit.
- **Rejected:** svelte-dnd-action (generic, not grid-aware)

### ADR-009: HTTP available alongside HTTPS on the dashboard
- **Decided:** Caddy listens on both `:80` and `:443`. No automatic HTTP→HTTPS redirect. Auth cookies set `Secure` only when the request came in via HTTPS (detected via `X-Forwarded-Proto`).
- **Why:** A first-time developer hitting a self-signed cert warning is the #1 friction point for a tool aimed at GitHub Stars. We keep HTTPS as an option for production-style deployments, but the default `./scripts/setup.sh && docker compose up -d` flow produces a working `http://localhost` with no warning, no trust-store manipulation, no HSTS cache issues. MQTT TLS remains mandatory (device credentials still need TLS).
- **Rejected:** HTTPS-only with a trust-the-CA script (worked but still a friction step), dev-mode env flag (brittle, easy to leak into prod).

### ADR-008: httpOnly Cookie for JWT storage
- **Decided:** Access + refresh tokens delivered as httpOnly, Secure, SameSite=Strict cookies. No `Authorization: Bearer` header support.
- **Why:** XSS cannot steal tokens stored in httpOnly cookies. SameSite=Strict prevents CSRF for same-origin dashboard (Caddy serves dashboard + API from https://localhost). WebSocket handshake is a regular HTTP GET so the browser attaches the same cookie — no query-param token leakage in logs.
- **Rejected:** localStorage (XSS-exposed), Bearer-only (CSRF on cookies if we mixed, plus no clean browser WebSocket story)

### ADR-007: Chart.js + svelte-chartjs
- **Decided:** Chart.js + svelte-chartjs
- **Why:** Most compatible, mature, large reference pool. Real-time update support via dataset mutation.
- **Rejected:** Apache ECharts (heavier bundle), D3 (too low-level for this scope)

---

## Stack

| Layer | Technology |
|-------|-----------|
| MQTT Broker | Mosquitto (TLS + username/password ACL) |
| Ingestion + API | Go 1.22+ + Fiber v2 |
| Time-series DB | TimescaleDB (PostgreSQL extension) |
| Dashboard | SvelteKit + Tailwind CSS |
| Reverse Proxy | Caddy |
| Container | Docker Compose |

---

## Directory Structure

```
Nexos/
├── docker-compose.yml
├── .env.example
├── Makefile
├── ARCHITECTURE.md
├── README.md
├── scripts/
│   ├── setup.sh              # interactive: generates .env + TLS certs
│   └── add-device.sh         # adds MQTT credentials for a device
├── broker/
│   ├── config/mosquitto.conf
│   ├── certs/                # gitignored
│   └── passwd                # gitignored
├── ingestion/
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── mqtt/             # MQTT subscriber, topic parser
│   │   ├── api/              # Fiber routes (REST + WebSocket)
│   │   ├── auth/             # JWT middleware (24h + refresh)
│   │   ├── alert/            # threshold engine, webhook dispatcher
│   │   └── db/               # TimescaleDB queries, migrations runner
│   ├── migrations/           # golang-migrate SQL files
│   ├── Dockerfile            # multi-stage build
│   └── go.mod
├── proxy/
│   └── Caddyfile
├── db/
│   └── init.sql
├── dashboard/
│   └── src/
│       ├── lib/
│       │   ├── components/
│       │   │   ├── charts/   # LineChart, Gauge, StatusIndicator
│       │   │   └── dashboard/ # gridstack grid, card wrapper
│       │   ├── stores/       # websocket store, device store, alert store
│       │   └── api/          # REST client (typed)
│       └── routes/
└── examples/
    ├── esp32/                # Arduino .ino sketch
    ├── raspberry-pi/         # paho-mqtt Python script
    └── simulator/            # Go data generator for testing
```

---

## Exact Dev Commands

```bash
# Initial setup
./scripts/setup.sh

# Start all services
docker compose up -d

# Stop all services
docker compose down

# Add a device
./scripts/add-device.sh

# Go — build
cd ingestion && go build ./cmd/server/...

# Go — test (all)
cd ingestion && go test ./...

# Go — test (single package)
cd ingestion && go test ./internal/mqtt/...

# Go — lint
cd ingestion && golangci-lint run

# Go — format
cd ingestion && gofmt -w .

# SvelteKit — dev server
cd dashboard && npm run dev

# SvelteKit — build
cd dashboard && npm run build

# SvelteKit — type check
cd dashboard && npx svelte-check

# DB migrations — up (reads .env automatically via Makefile)
make migrate-up

# DB migrations — new (scaffolds up + down SQL files)
make migrate-new name=<migration_name>

# DB migrations — down one step
make migrate-down

# Logs
docker compose logs -f ingestion
docker compose logs -f broker
```

---

## Code Contracts

### Go

**Package boundaries:**
- `internal/mqtt` — only owns: subscribe, parse topic, parse payload. Must not touch DB or WebSocket directly.
- `internal/db` — only owns: read/write TimescaleDB. No business logic.
- `internal/alert` — only owns: threshold evaluation, webhook dispatch.
- `internal/api` — owns: Fiber routes, WebSocket hub, JWT middleware wiring.

**Error handling:**
- Always wrap errors: `fmt.Errorf("mqtt: parse topic %q: %w", topic, err)`
- Never swallow errors silently. Log + return.
- Fatal only in `main.go` during startup.

**Logging:**
- Use `log/slog` with JSON handler in production, text handler in dev.
- Always include `device_id` and `sensor` fields on metric-related logs.

**Goroutines:**
- MQTT subscriber runs in a dedicated goroutine started in `main.go`.
- WebSocket hub runs in a dedicated goroutine started in `main.go`.
- Channel buffer size: 256 for metric fan-out.

### SvelteKit

**Stores:**
- `websocketStore` — owns WebSocket connection lifecycle + reconnect logic.
- `deviceStore` — derived from incoming messages, keyed by device_id.
- `alertStore` — alert rules, CRUD via REST.

**WebSocket reconnect policy:** exponential backoff, max 30s, indefinite retries.

**No inline styles.** Tailwind classes only.

---

## Data Schema

Canonical schema lives in [db/init.sql](db/init.sql). Summary:

- `devices(device_id PK, first_seen, last_seen)` — auto-populated by ingestion on every accepted message.
- `metrics(time, device_id, sensor, value)` — TimescaleDB hypertable via `create_hypertable('metrics', by_range('time'))` (TimescaleDB 2.13+ API). Composite index `(device_id, sensor, time DESC)`.
  - **No FK to devices** — TimescaleDB hypertables do not support FK constraints. Referential integrity is enforced at the application layer.
- `alert_rules(id, device_id, sensor, threshold, condition, webhook_url, enabled, created_at, updated_at)` — FK to `devices(device_id) ON DELETE CASCADE`.

**Retention:** `add_retention_policy('metrics', INTERVAL '90 days')` set in `init.sql`. The ingestion service reconciles the retention policy on startup based on `DATA_RETENTION_DAYS` (removes the existing policy and re-adds with the env-configured interval) so runtime changes in `.env` take effect on restart.

---

## Testing Philosophy

**Test these (business-critical logic):**
- `internal/mqtt`: topic parsing, payload parsing, invalid topic rejection
- `internal/alert`: threshold evaluation (above/below, edge cases)
- `internal/auth`: JWT sign/verify, expiry, refresh token rotation

**Test these with integration tests (not unit):**
- `internal/db`: queries against real TimescaleDB (use Docker in CI)
- `internal/api`: endpoint contracts via `httptest`

**Do NOT test:**
- `main.go` wiring
- Fiber middleware boilerplate
- Docker/infra config

---

## Security Constraints

- No API endpoint is accessible without valid JWT. No exceptions.
- **TLS is mandatory for MQTT.** No plaintext port 1883 exposed. This is non-negotiable because device credentials traverse this path.
- **TLS for the dashboard is recommended but not forced.** Caddy serves both `http://localhost` (port 80) and `https://localhost` (port 443). Developers testing locally can use HTTP without fighting a cert warning; the `Secure` attribute on auth cookies is set at runtime based on `X-Forwarded-Proto`, so login works on both. Production deployers behind a real domain should remove or firewall the HTTP listener.
- `.env`, `broker/passwd`, and `broker/certs/` contents are gitignored. Always.
- Webhook URLs are stored in DB, never in `.env` or config files.
- JWT secret minimum 32 bytes, generated by `setup.sh`. Never hardcoded.

---

## Explicit Anti-patterns

- **Do not add a Redis service.** If real-time fan-out seems slow, optimize the Go channel buffer first.
- **Do not add device control endpoints.** No `POST /devices/{id}/command`. Ever.
- **Do not accept free-form MQTT topics.** Topics not matching `devices/{device_id}/{sensor}` are silently dropped at ingestion.
- **Do not create per-sensor DB tables.** All metrics go into the single `metrics` hypertable.
- **Do not use Svelte component libraries** (Flowbite, Skeleton, etc.). Tailwind only.
- **Do not add multi-user features.** Single admin JWT is the auth model.
- **Do not parse non-JSON payloads.** If payload is not valid JSON with a `value` field (or bare number), drop it.
- **Do not add registration, license keys, or token gating.** Nexos is MIT open-source, locally installable, and has zero server-side dependencies. Anyone can clone and run.
- **Do not add server-side telemetry or phone-home.** Adoption is measured via GitHub stars only. No analytics beacons, no install-count pings, not even opt-in. A self-hosted tool that calls external services loses trust immediately with the IoT developer audience.

---

## Environment Variables

Canonical list lives in [.env.example](.env.example). Notes for developers:

- **DATABASE_URL** is **not** in `.env` — it is constructed in `docker-compose.yml` from `POSTGRES_PASSWORD`. Do not add it to `.env`.
- **MQTT_BROKER_URL** is also hardcoded in `docker-compose.yml` (`mqtts://broker:8883`). Only the credentials are in `.env`.
- `.env` is created by `scripts/setup.sh`. All secret values (`ADMIN_PASSWORD`, `JWT_SECRET`, `POSTGRES_PASSWORD`, `MQTT_INGESTION_PASS`, `MQTT_HEALTH_PASS`) are generated there and never committed.
- Two MQTT accounts exist internally:
  - `MQTT_INGESTION_USER` / `MQTT_INGESTION_PASS` — subscribes to `devices/#` for the ingestion service.
  - `MQTT_HEALTH_USER` / `MQTT_HEALTH_PASS` — used only by the broker healthcheck to verify TLS + auth.
