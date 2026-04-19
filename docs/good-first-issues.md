# Good first issues — release-ready drafts

Copy these into GitHub Issues when cutting each release so newcomers always
have a warm entry point. All five are scoped to land in under 200 lines of
change, respect the architectural invariants, and touch code that has
implicit test coverage from adjacent files.

Attach the `good first issue` label on creation.

---

## 1. Per-device card icons

**Labels:** `good first issue`, `area:dashboard`, `type:enhancement`

**Description**

Right now every sensor card shows the device_id in small grey text under the
sensor name. Power users requested visual differentiation — an icon (temperature
thermometer, battery, pressure gauge, etc.) picked from the sensor name or
explicit metadata.

**Scope**

- Add an `icon` field on the SensorCard component, derived from the sensor
  string via a small lookup table (e.g. `temperature → 🌡`, `battery → 🔋`).
- Fallback icon for unknown sensor names.
- Place the icon to the left of the sensor title.

**Out of scope**

- Icon upload from the dashboard (don't add a storage service).
- Per-device color themes (separate issue if requested).

**Files to touch**

- `dashboard/src/lib/components/SensorCard.svelte`
- Possibly a new `dashboard/src/lib/icons.ts`

**Definition of done**

- Card renders an icon from the lookup table
- Unknown sensors render a default
- `npm run check` + `npm run build` pass
- A screenshot attached to the PR

---

## 2. "Send test webhook" button on alert rules

**Labels:** `good first issue`, `area:alerts`, `type:enhancement`

**Description**

When creating a webhook alert, users currently have no way to verify the URL
is reachable until a real threshold triggers. Add a test endpoint + UI
button that dispatches a sample payload.

**Scope**

- `POST /api/alerts/:id/test` on the Go side — reuses the dispatcher with a
  synthetic event (`value = threshold + 1`, `triggered_at = now`).
- Button on the alerts page that calls it and shows success/fail toast.

**Out of scope**

- Test history (log of past test calls). Keep the endpoint stateless.

**Files to touch**

- `ingestion/internal/api/alert_routes.go`
- `dashboard/src/routes/alerts/+page.svelte`
- `dashboard/src/lib/api/alerts.ts`

**Definition of done**

- `curl -X POST /api/alerts/<id>/test` triggers one webhook call
- Dashboard button shows success/failure
- Go test coverage for the new handler
- `make ci` green

---

## 3. ESP8266 example sketch

**Labels:** `good first issue`, `area:examples`, `type:docs`

**Description**

We have an ESP32 example but ESP8266 (NodeMCU, Wemos D1 Mini) is still
popular for budget IoT projects. Add a parallel `.ino` sketch that works
within ESP8266's tighter TLS memory constraints.

**Scope**

- `examples/esp8266/nexos_example.ino`
- ESP8266-specific notes: smaller TLS fragment, `BearSSL` client, possibly
  CA fingerprint fallback instead of full cert validation.
- Update `examples/README.md` with ESP8266 row.

**Out of scope**

- Other microcontrollers (ESP32-C3, RP2040) — separate PRs.

**Definition of done**

- Sketch compiles against ESP8266 board package in Arduino IDE
- README update explains memory trade-offs
- Author has verified a physical ESP8266 board connecting (or documents
  that they couldn't test hardware)

---

## 4. Dashboard dark/light theme toggle

**Labels:** `good first issue`, `area:dashboard`, `type:enhancement`

**Description**

Nexos defaults to dark mode. Some users (and hackathon judges 👀) prefer
light mode. Add a toggle that switches the Tailwind palette.

**Scope**

- `tailwind.config.js` — extend dark-mode palette to a `light:` variant.
- `src/lib/stores/theme.ts` — persistent store in localStorage.
- Toggle button in the header.
- Respect `prefers-color-scheme` media query on first load.

**Out of scope**

- Per-chart color overrides.
- A full theming system (CSS variables, custom palettes). Keep it binary.

**Files to touch**

- `dashboard/tailwind.config.js`
- `dashboard/src/lib/stores/theme.ts` (new)
- `dashboard/src/routes/+layout.svelte`
- `dashboard/src/app.css`

**Definition of done**

- Toggle flips every visible surface (cards, charts, background)
- Setting persists across reloads
- First-visit honours system preference

---

## 5. Basic i18n structure (en / ko / ja)

**Labels:** `good first issue`, `area:dashboard`, `type:enhancement`

**Description**

The dashboard currently hardcodes English strings. Set up a minimal i18n
scaffold so contributors can add translations without a framework rewrite.

**Scope**

- Introduce `dashboard/src/lib/i18n/` with a flat JSON catalog per locale.
  (`en.json`, `ko.json`, `ja.json` as starters — the last two can stay
  partial; English is the fallback.)
- Simple `t(key)` helper in a Svelte store.
- Locale switcher in the header.
- Persist chosen locale in localStorage.

**Out of scope**

- Full framework integration (svelte-i18n, intl-messageformat, etc.). Keep
  it under 100 lines of new code. A richer library can replace this
  helper later.
- Translating the marketing landing — that lives in the `docs/` Astro project.

**Files to touch**

- `dashboard/src/lib/i18n/*` (new)
- `dashboard/src/lib/stores/locale.ts` (new)
- `dashboard/src/routes/+layout.svelte`
- All user-facing strings across components
