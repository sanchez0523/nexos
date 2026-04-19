# Hacker News — Show HN

## Title (60 char limit, one shot)

```
Show HN: Nexos – self-hosted IoT dashboard that auto-discovers sensors
```

Alternatives to test in order of preference:
1. `Show HN: Nexos – an IoT dashboard that builds itself from MQTT topics`
2. `Show HN: I made a zero-config self-hosted IoT monitoring stack`

## Link

`https://github.com/OWNER/REPO`

Not the landing page — HN readers prefer the repo. The README has the demo
GIF embedded at the top, so the first screen is the same as the landing's
hero regardless.

## Opening comment (post immediately after submitting)

```
Nexos is a self-hosted IoT monitoring stack with one opinionated
constraint: devices publish MQTT to devices/{device_id}/{sensor} with a
JSON payload of {"value": N}. In exchange, new sensors show up as
dashboard cards automatically — no YAML, no Telegraf config, no Grafana
editor.

A few design decisions that might be interesting:

- No Redis. A buffered Go channel fans metrics from the paho subscriber
  to the DB writer and the WebSocket hub. At 50-device scale the ops
  surface of Redis is pure overhead. ADR-002 in ARCHITECTURE.md if you want
  the longer version.
- TimescaleDB hypertable with a single metrics(time, device_id, sensor,
  value) table. Dynamic per-sensor tables sound clean but become an
  operational nightmare once topics are discovered, not declared.
- Cookie-based auth (httpOnly, SameSite=Strict) — not localStorage.
  Means WebSocket uses the same credential as REST instead of carrying
  a token in the URL.
- Monitoring only. No device control endpoint. Ever. Writing to MQTT
  from the server introduces a whole class of trust + topology problems
  we deliberately avoided.

MIT licensed. The install is a single `docker compose up -d` after
running `scripts/setup.sh` (generates TLS + .env + broker passwd in one
shot). Three example clients in the repo: ESP32 Arduino sketch,
Raspberry Pi Python, Go simulator.

Happy to answer questions, take feedback on the trade-offs, or hear
what you think this should integrate with next.
```

## Response prep — FAQs to have ready

- **"Why not just Grafana + InfluxDB?"** — You can. Nexos is the opinionated
  version that skips the dashboard-authoring and ingestion-config steps for
  the price of accepting the topic convention.
- **"What about multi-tenancy / RBAC?"** — Not in v1. The auth model is a
  single admin. Repositioning as a tool for teams is a v2 question if there's
  demand.
- **"How does it compare to ThingsBoard?"** — ThingsBoard is a platform;
  Nexos is a hyper-focused tool. 5 services, one binary per service, 6 weeks
  to v1 rather than years to a commercial product.
- **"Can I run it on a Raspberry Pi?"** — The Docker images are multi-arch
  (amd64 + arm64). Pi 4 / Pi 5 should be fine. Haven't stressed a Pi Zero.

## Timing

Post on a weekday, 7:30–9:00 AM PT for maximum visibility on the front page.
Avoid Tuesdays (peak competition) and Fridays (low evening engagement).

## After submitting

- [ ] Refresh every 10 min for first 2 hours; respond to every comment.
- [ ] Don't vote-ring. HN's algorithm punishes it harder than it rewards.
- [ ] If it dies before front page, don't resubmit within 30 days.
