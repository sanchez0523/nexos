package db

import (
	"context"
	"fmt"
	"time"
)

// MetricPoint is a single row returned from a time-series query, optionally
// downsampled by time_bucket.
type MetricPoint struct {
	Time  time.Time `json:"time"`
	Value float64   `json:"value"`
}

// MetricsQuery describes a read-side time-series query.
type MetricsQuery struct {
	DeviceID string
	Sensor   string
	From     time.Time
	To       time.Time
	Limit    int // hard cap on rows returned (safety net against huge responses)
}

// QueryMetrics returns points in [From, To] for the given (device_id, sensor).
// If the window exceeds 1 hour, results are downsampled into 1-minute buckets
// with an AVG aggregation to keep payloads under control.
func (d *DB) QueryMetrics(ctx context.Context, q MetricsQuery) ([]MetricPoint, error) {
	if q.DeviceID == "" || q.Sensor == "" {
		return nil, fmt.Errorf("db: device_id and sensor required")
	}
	if q.From.IsZero() || q.To.IsZero() || !q.From.Before(q.To) {
		return nil, fmt.Errorf("db: from/to invalid")
	}
	if q.Limit <= 0 {
		q.Limit = 5000
	}

	window := q.To.Sub(q.From)
	if window > time.Hour {
		return d.queryBucketed(ctx, q)
	}
	return d.queryRaw(ctx, q)
}

func (d *DB) queryRaw(ctx context.Context, q MetricsQuery) ([]MetricPoint, error) {
	rows, err := d.Pool.Query(ctx, `
		SELECT time, value
		FROM metrics
		WHERE device_id = $1 AND sensor = $2
		  AND time >= $3 AND time <= $4
		ORDER BY time ASC
		LIMIT $5
	`, q.DeviceID, q.Sensor, q.From, q.To, q.Limit)
	if err != nil {
		return nil, fmt.Errorf("db: query raw metrics: %w", err)
	}
	defer rows.Close()

	out := make([]MetricPoint, 0, 512)
	for rows.Next() {
		var p MetricPoint
		if err := rows.Scan(&p.Time, &p.Value); err != nil {
			return nil, fmt.Errorf("db: scan metric: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (d *DB) queryBucketed(ctx context.Context, q MetricsQuery) ([]MetricPoint, error) {
	rows, err := d.Pool.Query(ctx, `
		SELECT time_bucket('1 minute', time) AS bucket, AVG(value) AS value
		FROM metrics
		WHERE device_id = $1 AND sensor = $2
		  AND time >= $3 AND time <= $4
		GROUP BY bucket
		ORDER BY bucket ASC
		LIMIT $5
	`, q.DeviceID, q.Sensor, q.From, q.To, q.Limit)
	if err != nil {
		return nil, fmt.Errorf("db: query bucketed metrics: %w", err)
	}
	defer rows.Close()

	out := make([]MetricPoint, 0, 512)
	for rows.Next() {
		var p MetricPoint
		if err := rows.Scan(&p.Time, &p.Value); err != nil {
			return nil, fmt.Errorf("db: scan bucketed metric: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
