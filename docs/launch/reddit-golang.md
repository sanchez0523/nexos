# Reddit — r/golang

## Title

```
Nexos — self-hosted IoT monitoring in Go + SvelteKit (MIT). Design notes inside.
```

r/golang responds to technical depth and Go-specific trade-offs. Don't lead
with product; lead with implementation.

## Body

```markdown
I released **Nexos** this week and figured r/golang might be interested
in the Go half of the architecture, which had a few decisions I had to
sit with.

Repo: https://github.com/OWNER/REPO

## The design in one paragraph

Nexos is a self-hosted IoT monitoring platform. Devices publish MQTT
over TLS, the Go ingestion service subscribes, parses payloads, writes
to TimescaleDB, and fans out to WebSocket clients for a SvelteKit
dashboard that auto-generates cards as new sensors appear. Five
services, single Docker Compose, MIT licensed.

## Things that might be worth reading the code for

### Channel-based fan-out, not Redis

The MQTT subscriber feeds a buffered `chan Metric` (size 256). A single
goroutine reads from that channel and fans to:
1. The DB writer's input channel (blocking send — we want all metrics
   persisted).
2. The WebSocket hub (non-blocking — realtime is best-effort).
3. The alert engine (non-blocking — slow rule eval doesn't back-pressure
   the pipeline).

Code lives in `cmd/server/main.go`. This is the "no Redis" ADR (#002).
Worth noting: the paho handler itself does a non-blocking send to the
metric channel, so a DB stall doesn't back up into the MQTT client.

### Retention-policy reconciliation on startup

TimescaleDB's `add_retention_policy` is idempotent only if you manage
policy IDs yourself. Instead I `remove_retention_policy('metrics',
if_exists => true)` then re-add with `make_interval(days => $1)` from
the env var. One SQL round-trip, no state to track, env var changes
take effect on service restart.

### WebSocket hub as a Client interface

The hub doesn't know about gorilla/fiber-websocket conns. It owns a map
of `Client` — an interface with `ID()` and `Send() chan<- Event`. The
concrete Fiber client type lives in api/ws_routes.go. Means the hub is
unit-testable without spinning up a socket, and swapping transports
later (SSE? gRPC stream?) is a packaging change, not a rewrite.

Subtlety worth flagging: the hub does a non-blocking send to
`Client.Send()`. If a slow client's buffer fills, it gets evicted
rather than stalling broadcasts to everyone else. Explicit design
contract documented in the Client interface comment.

### JWT in httpOnly cookies, not headers

WebSocket handshakes carry cookies same-origin. No `?token=` in the URL
(don't want JWTs in access logs). No Bearer header (XSS-exfiltratable).
Server reads `nexos_access` cookie in both the Fiber middleware and the
WebSocket upgrade path — same claim parser both places.

Cookie attributes: `httpOnly + Secure + SameSite=Strict + Path=/`. For
the refresh cookie I scope Path to `/api/auth` so it's only sent to the
endpoint that needs it.

### Batch write with pgx CopyFrom + unnest UPSERT

Metric writer flushes every 100ms or 50 rows (whichever comes first).
CopyFrom for the metrics table, a single `INSERT … SELECT FROM unnest()
ON CONFLICT DO UPDATE` for the devices table. This lets Auto-Discovery
(a new device_id showing up) cost one DB round-trip per flush instead
of one per metric.

## Stuff I punted on

- Full integration-test coverage. Unit tests are thorough on business
  logic (parsing, alerts, JWT) but `httptest`-style endpoint tests are
  marked TODO for post-v1.
- A TUI / CLI client for the API. Feels out of scope.
- Prometheus metrics on the ingestion service. Would love feedback on
  whether this is table-stakes.

## Feedback welcome

Especially on:
- The single-channel fan-out pattern — would you split into per-consumer
  channels earlier, or only once load demands it?
- The blocking send to the DB writer vs non-blocking to everything else
  — principled or accidental?
- Dropping pgx's tracer vs keeping it on in production.

MIT license, no registration, fully local. Happy to dig into any of the
above.
```

## Notes

r/golang's rules:
- No bare link posts. The writeup above is the post.
- Self-promotion is OK if the technical content is the focus.
- Downvotes are fast on anything that feels like marketing. Keep the
  product language in the repo's README — this post is about code.
