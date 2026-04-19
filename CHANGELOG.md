# Changelog

All notable changes to Nexos are recorded here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Nothing yet — cut a release to populate this section.

## [1.0.0] — 2026-04-19

First public release. Everything below ships in the initial tag.

### Added
- **Ingestion service** — Go + Fiber. MQTT TLS subscriber, TimescaleDB batch
  writer, retention-policy reconciliation on startup, graceful shutdown.
- **Topic Auto-Discovery** — new `device_id` / `sensor` combinations appear
  on the dashboard without config edits, driven by the enforced
  `devices/{device_id}/{sensor}` topic convention.
- **Authenticated REST API** — devices list, sensor list, historical
  metrics with automatic 1-minute bucketing beyond a 1-hour window, alert
  rule CRUD.
- **WebSocket hub** — real-time fan-out of every accepted metric, per-client
  bounded send buffer, slow-client eviction.
- **Alert engine** — strict-comparison threshold rules (`>` / `<`),
  per-rule cooldown (`ALERT_TIMEOUT_SECONDS`), asynchronous Generic-JSON
  webhook dispatcher with 10s timeout and one retry.
- **Dashboard (SvelteKit + Tailwind + Chart.js + gridstack.js)** — login,
  auto-generated sensor cards, drag-and-drop layout persisted to
  localStorage, time-range picker (15m/1h/6h/24h/7d), alert rule UI.
- **Cookie-based session auth** — httpOnly, Secure, SameSite=Strict access
  + refresh cookies. Transparent single-flight refresh on 401.
- **Setup flow** — `./scripts/setup.sh` generates TLS CA + server cert,
  Mosquitto passwd, and `.env` with 32+ byte secrets in one pass.
- **Device registration** — `./scripts/add-device.sh` adds MQTT accounts
  and hot-reloads the broker via SIGHUP.
- **Examples** — ESP32 Arduino sketch, Raspberry Pi Python client, Go
  simulator for N synthetic devices.
- **Full Docker Compose stack** — broker, db, ingestion, dashboard, caddy
  reverse proxy (automatic TLS for localhost).
- **CI** — GitHub Actions pipeline: Go tests with TimescaleDB service
  container, golangci-lint, svelte-check, dashboard build, compose config
  validation.
- **Release automation** — `v*.*.*` tags publish multi-arch Docker images
  to GHCR and create a GitHub Release.

### Security
- TLS 1.2+ required for all MQTT connections; plaintext port 1883 is never
  exposed outside the Docker network.
- Per-device ACL pattern (`devices/%u/#`) prevents device-ID spoofing at
  the broker layer.
- `JWT_SECRET` must be at least 32 bytes; enforced at service startup.

[Unreleased]: https://github.com/OWNER/REPO/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/OWNER/REPO/releases/tag/v1.0.0
