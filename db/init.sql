-- Nexos TimescaleDB schema
-- Runs once on first container start via docker-entrypoint-initdb.d

CREATE EXTENSION IF NOT EXISTS timescaledb;

-- ── Devices registry (auto-populated by Auto-Discovery) ─────────────────────
CREATE TABLE IF NOT EXISTS devices (
    device_id   TEXT        PRIMARY KEY,
    first_seen  TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ── Time-series metrics (hypertable) ────────────────────────────────────────
-- Note: no FK to devices — TimescaleDB hypertables do not support FK constraints.
-- Referential integrity is maintained at the application layer (ingestion service).
CREATE TABLE IF NOT EXISTS metrics (
    time        TIMESTAMPTZ      NOT NULL,
    device_id   TEXT             NOT NULL,
    sensor      TEXT             NOT NULL,
    value       DOUBLE PRECISION NOT NULL
);

SELECT create_hypertable('metrics', by_range('time'));

CREATE INDEX IF NOT EXISTS idx_metrics_device_sensor_time
    ON metrics (device_id, sensor, time DESC);

-- ── Alert rules ──────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS alert_rules (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id   TEXT        NOT NULL,
    sensor      TEXT        NOT NULL,
    threshold   DOUBLE PRECISION NOT NULL,
    condition   TEXT        NOT NULL CHECK (condition IN ('above', 'below')),
    webhook_url TEXT        NOT NULL,
    enabled     BOOLEAN     NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (device_id) REFERENCES devices (device_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_alert_rules_device_sensor
    ON alert_rules (device_id, sensor) WHERE enabled = true;

-- ── Retention policy (overridden at runtime by Go service if env differs) ───
-- Default: 90 days. Go ingestion service calls add_retention_policy on startup
-- with the value from DATA_RETENTION_DAYS env var, replacing this default.
SELECT add_retention_policy('metrics', INTERVAL '90 days');
