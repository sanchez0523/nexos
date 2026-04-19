# Dev.to — long-form launch

## Frontmatter

```yaml
---
title: I built an IoT dashboard that auto-discovers sensors from MQTT topics
published: true
description: Nexos is a self-hosted IoT monitoring stack with one
  opinionated constraint — and in return, your dashboard assembles
  itself as devices connect.
tags: iot, go, sveltekit, opensource
cover_image: https://raw.githubusercontent.com/OWNER/REPO/main/docs/public/demo.gif
canonical_url: https://github.com/OWNER/REPO
---
```

## Body

```markdown
I've set up Grafana + InfluxDB + Mosquitto three times for different
side projects. Every time I end up doing the same manual work:

1. Write the Telegraf config so InfluxDB knows what fields to index.
2. Open Grafana, build a dashboard, lay out panels.
3. Add a new sensor. Back to Grafana. Add another panel. Repeat.

The primitives are great. The integration work is tedious.

So I built **Nexos** — a self-hosted IoT monitoring stack with one
deliberate constraint: devices follow a topic convention.

```
devices/{device_id}/{sensor}   {"value": 23.5}
```

That's the entire contract. Publish to a matching topic, a card shows
up on the dashboard. Publish to a new sensor, a new card appears. Drag
cards around, set thresholds, get webhooks when they fire.

![Nexos demo](https://raw.githubusercontent.com/OWNER/REPO/main/docs/public/demo.gif)

## The stack

Five services, coordinated by `docker compose`:

| Service     | Role                                                      |
|-------------|-----------------------------------------------------------|
| broker      | Mosquitto 2.x, TLS 1.2+, per-device ACL                   |
| db          | TimescaleDB (PostgreSQL 16)                               |
| ingestion   | Go + Fiber — MQTT subscriber, REST API, WebSocket hub     |
| dashboard   | SvelteKit static build (SPA)                              |
| proxy       | Caddy — HTTPS termination, routing                        |

All MIT licensed. No phone-home, no license keys, no cloud account.

## Three decisions that matter

**No Redis.** A buffered Go channel fans metrics from the paho MQTT
subscriber out to the DB writer and the WebSocket broadcaster. At the
intended scale (tens of devices, not tens of thousands), Redis is pure
ops overhead.

**Single TimescaleDB table.** `metrics(time, device_id, sensor, value)`.
No per-sensor schema, no dynamic DDL. Add a sensor — just a new column
value in the same table. Queries stay cheap because of the
`(device_id, sensor, time DESC)` composite index.

**Cookie auth, not localStorage.** JWT goes in an `httpOnly`,
`Secure`, `SameSite=Strict` cookie. Means XSS can't exfiltrate the
token, and the WebSocket handshake uses the same cookie as REST — no
`?token=` in the URL.

## Getting started

```bash
git clone https://github.com/OWNER/REPO nexos
cd nexos
./scripts/setup.sh              # generates .env, TLS, passwd
docker compose up -d
```

Add a device:

```bash
./scripts/add-device.sh esp32-living
```

The script prints the MQTT credentials. Point an ESP32, Raspberry Pi,
or the bundled Go simulator at the broker and the dashboard fills in.

## What's next

- GitHub Discussions is on. Would love to see `Show off your Nexos
  setup` posts.
- Good first issues up for anyone who wants to contribute.
- Feedback especially welcome on: the topic convention (too strict?),
  the alert semantics (strict `>` vs `>=`?), what protocols to add
  beyond MQTT.

Repo: https://github.com/OWNER/REPO

If you build something with it, drop a link — I read everything.
```

## Post-publish

- [ ] Cross-post to `https://OWNER.dev.to/` profile if applicable.
- [ ] Engage with comments in the first 24h.
- [ ] If it gets traction, submit it as a follow-up to Dev.to's weekly
      newsletter curators via Discord.
