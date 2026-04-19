# Setting up community channels

One-time setup steps to run **before** the public launch posts
([docs/launch/](launch/)). Each step takes under 5 minutes.

## 1. Enable GitHub Discussions

Repo → Settings → Features → **Discussions** → check.

Categories to seed:

| Category       | Format    | Description                                         |
|----------------|-----------|-----------------------------------------------------|
| Announcements  | Announce  | Releases, breaking changes, RFC links (maintainer-only posting) |
| Q&A            | Q&A       | Install / config questions                          |
| Show & Tell    | Discussion| User setups, dashboards, integrations               |
| Ideas          | Discussion| Feature proposals (before formal Issue)             |
| Polls          | Poll      | Breaking changes, roadmap priorities                |

Pin a sticky post in **Show & Tell** titled *"Show off your Nexos setup!"*
with this body:

```
Post a screenshot, a video, or a tweet-sized story of what you've wired
up. Dashboards from real hardware, weird ESP32 rigs, prod alert
integrations — all welcome. Linking back to your own blog / Mastodon /
X is fine and encouraged.

No commercial announcements (see CODE_OF_CONDUCT.md). Everything else
is fair game.
```

## 2. Seed `good first issue` labels

Open issues from [docs/good-first-issues.md](good-first-issues.md) one at a
time, attach the `good first issue` label, and assign to an unassigned
milestone. Aim for 3–5 open at launch.

Repo → Labels → confirm these exist:

- `good first issue` (default, GitHub auto-creates)
- `area:dashboard`, `area:ingestion`, `area:alerts`, `area:examples`
- `type:bug`, `type:enhancement`, `type:docs`

## 3. Pin the quickstart issue

Create one issue titled **"Welcome! Start here if you're new 👋"** that links
to README, Discussions, and the good-first-issue list. Pin it to the repo
top.

## 4. Set up watching for cross-platform pings

Add notification rules (GitHub web UI or mobile):

- Mentions of `@yourhandle` on HN via [hnrss.org](https://hnrss.org) feed
- Reddit search for `nexos` + `site:reddit.com` in Google Alerts
- Twitter/X search for `github.com/OWNER/REPO` with an email digest

This is how you catch organic mentions fast enough to respond within the
first-day window.

## 5. What NOT to enable

- **No Discord/Slack.** A Discord server fragments Q&A that should live in
  searchable Discussions and drains maintainer time. Add one only if the
  community actively demands it (typically after 1k stars).
- **No mailing list / newsletter.** We don't collect emails.
- **No Patreon / GitHub Sponsors** at launch. Keep the optics focused on
  the code, not funding. Enable later if the project proves durable.

## 6. After launch — weekly cadence (first 4 weeks)

- Monday: triage new issues, close stale, respond to Discussions questions
- Wednesday: post a weekly "what landed this week" update in Announcements
- Friday: pick one "Show & Tell" to cross-post to r/selfhosted (with the
  author's permission) for a mid-week bump

Four weeks of this is enough to seed the community habit. After that, drop
to ad-hoc cadence.
