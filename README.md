# Nexos

**Self-hosted IoT monitoring that auto-generates dashboards from MQTT topics.**

Point your ESP32 at Nexos and a chart appears. Add another sensor, a second
chart appears. No YAML files, no dashboard editor, no code changes — just a
topic convention.

> 한국어 문서: [README.ko.md](README.ko.md)

<!-- Replace sanchez0523/nexos with your GitHub path once the repo is published. -->
<p align="center">
  <img alt="CI" src="https://github.com/sanchez0523/nexos/actions/workflows/ci.yml/badge.svg" />
  <img alt="License" src="https://img.shields.io/badge/license-MIT-green" />
  <img alt="Release" src="https://img.shields.io/github/v/release/sanchez0523/nexos?include_prereleases" />
</p>

---

## Why

Existing self-hosted stacks (Grafana + InfluxDB + Mosquitto) give you the
primitives but hand you the integration work. You design the schema, write
the Telegraf config, lay out the dashboard, wire the alerts — then you
repeat it every time you add a sensor.

Nexos collapses that into one decision: **follow the topic convention**.

```
devices/{device_id}/{sensor}   {"value": 23.5}
```

Publish to that topic. A card appears on the dashboard. Drag it wherever.
Set a threshold. Get a webhook when it fires. Ship.

## Features

- **Topic Auto-Discovery** — new `device_id` or `sensor` → new card, no config
- **Real-time charts** — WebSocket fan-out from the Go ingestion service
- **Drag-and-drop layout** — gridstack.js, saved to browser localStorage
- **Historical queries** — TimescaleDB hypertable, auto-downsampled beyond 1h
- **Threshold alerts → Generic JSON webhook** — wire to Slack, n8n, anything
- **Single binary + Docker Compose** — no Kubernetes, no cloud services
- **TLS + per-device ACL out of the box** — each device can only publish its own topics
- **MIT licensed, no phone-home, no registration**

## Quick start

**Requirements:** Docker 24+ with Compose v2, `openssl` and `bash` on the host. 2 GB RAM free.

```bash
git clone https://github.com/sanchez0523/nexos nexos
cd nexos

# Interactive setup — generates .env, TLS certs, and the broker passwd file.
./scripts/setup.sh

# Bring everything up.
docker compose up -d

# Open the dashboard. Accept the self-signed CA warning (Nexos runs locally).
open https://localhost
```

Register a device:

```bash
./scripts/add-device.sh esp32-living
# → prints username, password, CA cert path, topic prefix
```

Point an ESP32, Raspberry Pi, or the bundled Go simulator at the broker. See
[examples/](examples/) for one-file starter sketches.

## Architecture

```
 IoT device ── MQTT/TLS ─▶ Mosquitto ──▶ Go ingestion ──▶ TimescaleDB
                                              │
                                              ├── WebSocket ──▶ SvelteKit dashboard
                                              └── Webhook dispatch ──▶ your alerting
```

Five services total, coordinated by `docker compose`:

| Service       | Role                                                    |
|---------------|---------------------------------------------------------|
| `broker`      | Mosquitto 2.x, TLS 1.2+, passwd + ACL                   |
| `db`          | TimescaleDB (PostgreSQL 16 + extensions)                |
| `ingestion`   | Go + Fiber — MQTT subscriber, REST API, WebSocket hub, alert engine |
| `dashboard`   | SvelteKit static build (SPA, served by `serve`)         |
| `proxy`       | Caddy — terminates HTTPS, routes `/api`, `/ws`, `/` to the right service |

Longer rationale for each choice lives in [ARCHITECTURE.md](ARCHITECTURE.md) as ADR entries.

## Environment variables

All generated automatically by `./scripts/setup.sh`. Shown for reference —
edit `.env` directly and `docker compose up -d` to apply.

| Variable                  | Purpose                                         |
|---------------------------|-------------------------------------------------|
| `ADMIN_USERNAME`          | Dashboard login                                 |
| `ADMIN_PASSWORD`          | Dashboard login                                 |
| `JWT_SECRET`              | ≥32 bytes, signs access + refresh tokens        |
| `JWT_ACCESS_TTL`          | Access cookie lifetime (default 24h)            |
| `JWT_REFRESH_TTL`         | Refresh cookie lifetime (default 168h = 7 days) |
| `POSTGRES_PASSWORD`       | Auto-generated, used only inside the Docker net |
| `MQTT_INGESTION_USER/PASS`| Internal Mosquitto account the ingestion service uses to subscribe |
| `MQTT_HEALTH_USER/PASS`   | Internal account used by the broker healthcheck |
| `DATA_RETENTION_DAYS`     | TimescaleDB retention policy (default 90)       |
| `ALERT_TIMEOUT_SECONDS`   | Min interval between repeat webhook calls for the same rule (default 60) |

## Topic & payload contract

- **Topic:** must match `devices/{device_id}/{sensor}`. Topics that don't match are silently dropped.
- **Payload:** JSON object `{"value": <number>}` **or** a bare number. Anything else is dropped.

This constraint is what makes Auto-Discovery possible. There's a full ADR for it in `ARCHITECTURE.md`.

## Alerts

Create alert rules from the dashboard (`/alerts`) or via REST:

```bash
curl -b cookies.txt https://localhost/api/alerts \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "esp32-living",
    "sensor": "temperature",
    "condition": "above",
    "threshold": 30,
    "webhook_url": "https://hooks.example.com/t/xxxx",
    "enabled": true
  }'
```

When a reading crosses the threshold (strict `>` / `<`), Nexos POSTs:

```json
{
  "device_id": "esp32-living",
  "sensor": "temperature",
  "value": 31.2,
  "threshold": 30,
  "condition": "above",
  "triggered_at": "2026-04-18T10:30:00Z"
}
```

`ALERT_TIMEOUT_SECONDS` enforces a per-rule cooldown so a flapping sensor
can't spam your receiver.

## What Nexos deliberately does NOT do

This project is small on purpose. Things out of scope for v1:

- **Device control / command sending** — monitoring only, no `POST /devices/{id}/command`.
- **Multi-user / RBAC** — one admin per install.
- **Guaranteed message delivery** — some loss is acceptable; no Redis Streams or Kafka.
- **Token gating / license keys / phone-home telemetry** — Nexos is MIT, runs locally, and never contacts our servers.
- **Binary / Protobuf payloads** — JSON only.

If you need any of these, Nexos is not the right tool. That's fine — we'd
rather say no than ship a kitchen sink.

## Development

```bash
# Backend tests
cd ingestion && go test -race ./...

# Dashboard dev server (proxies /api + /ws to localhost:8080)
cd dashboard && npm install && npm run dev

# Full CI locally
make ci
```

Contributing guide: [CONTRIBUTING.md](CONTRIBUTING.md). Architecture rules
and ADRs: [ARCHITECTURE.md](ARCHITECTURE.md). Phase-by-phase roadmap: [ROADMAP.md](ROADMAP.md).

## License

MIT — see [LICENSE](LICENSE).
