package db

import (
	"context"
	"fmt"
)

// ReconcileRetentionPolicy ensures exactly one retention policy exists on the
// metrics hypertable with the given number of days. It removes any existing
// policy first (ignoring "not found" errors) and re-adds with the configured
// interval. Called once on service startup.
func (d *DB) ReconcileRetentionPolicy(ctx context.Context, days int) error {
	if days <= 0 {
		return fmt.Errorf("db: retention days must be positive, got %d", days)
	}

	// remove_retention_policy errors if no policy exists — swallow via if_exists.
	if _, err := d.Pool.Exec(ctx,
		`SELECT remove_retention_policy('metrics', if_exists => true)`); err != nil {
		return fmt.Errorf("db: remove retention policy: %w", err)
	}

	if _, err := d.Pool.Exec(ctx,
		`SELECT add_retention_policy('metrics', drop_after => make_interval(days => $1))`,
		days); err != nil {
		return fmt.Errorf("db: add retention policy: %w", err)
	}
	return nil
}
