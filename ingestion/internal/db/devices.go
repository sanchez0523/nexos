package db

import (
	"context"
	"fmt"
	"time"
)

// Device is the minimal device registry row exposed to callers.
type Device struct {
	DeviceID  string    `json:"device_id"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}

// TouchDevice upserts a device row, advancing last_seen. This is called once
// per accepted MQTT message and is the sole mechanism behind Auto-Discovery:
// if the device_id is new, first_seen is set here.
func (d *DB) TouchDevice(ctx context.Context, deviceID string, seenAt time.Time) error {
	_, err := d.Pool.Exec(ctx, `
		INSERT INTO devices (device_id, first_seen, last_seen)
		VALUES ($1, $2, $2)
		ON CONFLICT (device_id) DO UPDATE
		SET last_seen = EXCLUDED.last_seen
	`, deviceID, seenAt)
	if err != nil {
		return fmt.Errorf("db: touch device %q: %w", deviceID, err)
	}
	return nil
}

// ListDevices returns all registered devices ordered by last_seen DESC.
func (d *DB) ListDevices(ctx context.Context) ([]Device, error) {
	rows, err := d.Pool.Query(ctx, `
		SELECT device_id, first_seen, last_seen
		FROM devices
		ORDER BY last_seen DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("db: list devices: %w", err)
	}
	defer rows.Close()

	var out []Device
	for rows.Next() {
		var dev Device
		if err := rows.Scan(&dev.DeviceID, &dev.FirstSeen, &dev.LastSeen); err != nil {
			return nil, fmt.Errorf("db: scan device: %w", err)
		}
		out = append(out, dev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: iterate devices: %w", err)
	}
	return out, nil
}

// ListSensorsByDevice returns the distinct sensor names observed for a given
// device_id. Backed by the `(device_id, sensor, time DESC)` index.
func (d *DB) ListSensorsByDevice(ctx context.Context, deviceID string) ([]string, error) {
	rows, err := d.Pool.Query(ctx, `
		SELECT DISTINCT sensor
		FROM metrics
		WHERE device_id = $1
		ORDER BY sensor
	`, deviceID)
	if err != nil {
		return nil, fmt.Errorf("db: list sensors for %q: %w", deviceID, err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("db: scan sensor: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
