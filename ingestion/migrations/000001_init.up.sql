-- Nexos base schema.
-- Mirrors db/init.sql but owned by golang-migrate so the ingestion service
-- can reconcile schema changes over time.

CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE IF NOT EXISTS devices (
    device_id   TEXT        PRIMARY KEY,
    first_seen  TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- No FK to devices — TimescaleDB hypertables don't support FK constraints.
CREATE TABLE IF NOT EXISTS metrics (
    time        TIMESTAMPTZ      NOT NULL,
    device_id   TEXT             NOT NULL,
    sensor      TEXT             NOT NULL,
    value       DOUBLE PRECISION NOT NULL
);

SELECT create_hypertable('metrics', by_range('time'), if_not_exists => true);

CREATE INDEX IF NOT EXISTS idx_metrics_device_sensor_time
    ON metrics (device_id, sensor, time DESC);

CREATE TABLE IF NOT EXISTS alert_rules (
    id          UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id   TEXT             NOT NULL,
    sensor      TEXT             NOT NULL,
    threshold   DOUBLE PRECISION NOT NULL,
    condition   TEXT             NOT NULL CHECK (condition IN ('above', 'below')),
    webhook_url TEXT             NOT NULL,
    enabled     BOOLEAN          NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ      NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ      NOT NULL DEFAULT now(),
    FOREIGN KEY (device_id) REFERENCES devices (device_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_alert_rules_device_sensor
    ON alert_rules (device_id, sensor) WHERE enabled = true;
