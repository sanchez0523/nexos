# Reddit — r/selfhosted

## Title

```
[Release] Nexos — self-hosted IoT monitoring where the dashboard builds itself from MQTT topics
```

r/selfhosted appreciates clear flair ("Release") and direct value claims over
marketing language. The subreddit has a 20-char minimum, 300-char max.

## Body

```markdown
Hey r/selfhosted,

I built **Nexos** because I got tired of re-gluing Grafana + InfluxDB +
Mosquitto every time I added a new sensor. It's a small, opinionated
stack where devices publish to `devices/{device_id}/{sensor}` and a
dashboard card appears automatically. No YAML, no dashboard editor, no
reload.

**What you get**

- 5-service Docker Compose (broker, TimescaleDB, Go ingestion, SvelteKit
  dashboard, Caddy)
- MQTT/TLS with per-device ACL — each device can only publish its own
  topic namespace
- Real-time charts via WebSocket, historical queries with automatic
  1-minute bucketing beyond 1 hour
- Threshold alerts → Generic JSON webhook (plug into n8n / Node-RED /
  your Slack bridge)
- Drag-and-drop layout, saved per-browser to localStorage

**What it deliberately doesn't do**

- No device control (monitoring only, no `POST /command`)
- No multi-user (single admin per install)
- No phone-home, no license keys, no registration — MIT, fully local
- No Redis / Kafka (a buffered Go channel is enough at this scale)

**5-min quickstart**

```bash
git clone https://github.com/OWNER/REPO nexos
cd nexos
./scripts/setup.sh           # generates .env + TLS + broker passwd
docker compose up -d
./scripts/add-device.sh esp32-living
```

Examples in the repo for ESP32 (Arduino), Raspberry Pi (Python), and a
Go simulator if you don't have hardware handy.

Runs fine on a Pi 4 (multi-arch images). ~800 MB total RAM for the
whole stack with 10 devices connected.

Repo: https://github.com/OWNER/REPO

Feedback welcome. Especially curious what integrations you'd want beyond
webhooks, and whether the "topic convention is mandatory" design is too
restrictive for your setups.
```

## Subreddit etiquette

- Add flair: **Release** or **Self-Promotion**.
- Don't post more than once per 30 days to the same subreddit.
- Respond to every top-level comment within 12 hours of posting.
- Link to the repo, not the landing page. r/selfhosted trusts repos.

## Timing

Weekend mornings (Sat/Sun 8-11 AM ET) typically outperform weekdays on
r/selfhosted because the audience is hobbyist homelabbers.
