package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"

	ingestmqtt "github.com/nexos-io/nexos/ingestion/internal/mqtt"
)

// MetricsWriter buffers incoming metrics and flushes them to the database
// every flushInterval or once the buffer reaches flushSize — whichever comes
// first. It is safe for a single goroutine to call Enqueue; the flush
// goroutine owns all DB I/O.
type MetricsWriter struct {
	db            *DB
	in            <-chan ingestmqtt.Metric
	flushInterval time.Duration
	flushSize     int
}

// NewMetricsWriter constructs a writer that reads from `in` and flushes using
// the defaults specified in CLAUDE.md (100ms / 50 rows).
func NewMetricsWriter(d *DB, in <-chan ingestmqtt.Metric) *MetricsWriter {
	return &MetricsWriter{
		db:            d,
		in:            in,
		flushInterval: 100 * time.Millisecond,
		flushSize:     50,
	}
}

// Run blocks until ctx is cancelled, then flushes any remaining buffered
// metrics before returning. The final flush uses a detached context with a
// short deadline so in-flight rows survive graceful shutdown even though the
// parent ctx is already Done.
func (w *MetricsWriter) Run(ctx context.Context) {
	buf := make([]ingestmqtt.Metric, 0, w.flushSize)
	tick := time.NewTicker(w.flushInterval)
	defer tick.Stop()

	flush := func(flushCtx context.Context) {
		if len(buf) == 0 {
			return
		}
		if err := w.flush(flushCtx, buf); err != nil {
			slog.Error("metrics: flush failed", "err", err, "batch_size", len(buf))
		}
		buf = buf[:0]
	}

	for {
		select {
		case <-ctx.Done():
			drainCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			flush(drainCtx)
			cancel()
			return
		case m, ok := <-w.in:
			if !ok {
				drainCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				flush(drainCtx)
				cancel()
				return
			}
			buf = append(buf, m)
			if len(buf) >= w.flushSize {
				flush(ctx)
			}
		case <-tick.C:
			flush(ctx)
		}
	}
}

func (w *MetricsWriter) flush(ctx context.Context, batch []ingestmqtt.Metric) error {
	if len(batch) == 0 {
		return nil
	}

	// Upsert unique device IDs first so Auto-Discovery sees the device
	// immediately, even before any metric row is queried.
	if err := w.upsertDevices(ctx, batch); err != nil {
		return fmt.Errorf("db: upsert devices: %w", err)
	}

	rows := make([][]any, len(batch))
	for i, m := range batch {
		rows[i] = []any{m.Time, m.DeviceID, m.Sensor, m.Value}
	}

	_, err := w.db.Pool.CopyFrom(ctx,
		pgx.Identifier{"metrics"},
		[]string{"time", "device_id", "sensor", "value"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("db: copy metrics: %w", err)
	}
	return nil
}

// upsertDevices deduplicates device IDs in the batch, records the latest
// seenAt per device, then performs a single multi-row UPSERT.
func (w *MetricsWriter) upsertDevices(ctx context.Context, batch []ingestmqtt.Metric) error {
	latest := make(map[string]time.Time, len(batch))
	for _, m := range batch {
		if prev, ok := latest[m.DeviceID]; !ok || m.Time.After(prev) {
			latest[m.DeviceID] = m.Time
		}
	}

	ids := make([]string, 0, len(latest))
	times := make([]time.Time, 0, len(latest))
	for id, t := range latest {
		ids = append(ids, id)
		times = append(times, t)
	}

	_, err := w.db.Pool.Exec(ctx, `
		INSERT INTO devices (device_id, first_seen, last_seen)
		SELECT id, seen, seen FROM unnest($1::text[], $2::timestamptz[]) AS t(id, seen)
		ON CONFLICT (device_id) DO UPDATE
		SET last_seen = EXCLUDED.last_seen
		WHERE devices.last_seen < EXCLUDED.last_seen
	`, ids, times)
	return err
}
