package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// AlertCondition is a typed enum matching the CHECK constraint on alert_rules.
type AlertCondition string

const (
	ConditionAbove AlertCondition = "above"
	ConditionBelow AlertCondition = "below"
)

// Valid reports whether the condition is one of the supported values.
func (c AlertCondition) Valid() bool {
	return c == ConditionAbove || c == ConditionBelow
}

// ErrAlertNotFound is returned when the given UUID does not exist.
var ErrAlertNotFound = errors.New("db: alert rule not found")

// AlertRule mirrors the alert_rules table.
type AlertRule struct {
	ID         uuid.UUID      `json:"id"`
	DeviceID   string         `json:"device_id"`
	Sensor     string         `json:"sensor"`
	Threshold  float64        `json:"threshold"`
	Condition  AlertCondition `json:"condition"`
	WebhookURL string         `json:"webhook_url"`
	Enabled    bool           `json:"enabled"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// AlertRuleInput is the write-side payload accepted by Create/Update. Kept
// separate so server-managed fields (id, timestamps) never leak into the API.
type AlertRuleInput struct {
	DeviceID   string         `json:"device_id"`
	Sensor     string         `json:"sensor"`
	Threshold  float64        `json:"threshold"`
	Condition  AlertCondition `json:"condition"`
	WebhookURL string         `json:"webhook_url"`
	Enabled    bool           `json:"enabled"`
}

// ListAlertRules returns every rule, newest first.
func (d *DB) ListAlertRules(ctx context.Context) ([]AlertRule, error) {
	rows, err := d.Pool.Query(ctx, `
		SELECT id, device_id, sensor, threshold, condition, webhook_url, enabled, created_at, updated_at
		FROM alert_rules
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("db: list alert rules: %w", err)
	}
	defer rows.Close()

	var out []AlertRule
	for rows.Next() {
		r, err := scanAlertRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListEnabledAlertRulesFor returns the subset of active rules matching a
// specific (device_id, sensor). Used by the threshold engine on each metric.
func (d *DB) ListEnabledAlertRulesFor(ctx context.Context, deviceID, sensor string) ([]AlertRule, error) {
	rows, err := d.Pool.Query(ctx, `
		SELECT id, device_id, sensor, threshold, condition, webhook_url, enabled, created_at, updated_at
		FROM alert_rules
		WHERE device_id = $1 AND sensor = $2 AND enabled = true
	`, deviceID, sensor)
	if err != nil {
		return nil, fmt.Errorf("db: list enabled rules for %q/%q: %w", deviceID, sensor, err)
	}
	defer rows.Close()

	var out []AlertRule
	for rows.Next() {
		r, err := scanAlertRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CreateAlertRule inserts a new rule and returns the stored row (including the
// generated UUID and timestamps).
func (d *DB) CreateAlertRule(ctx context.Context, in AlertRuleInput) (AlertRule, error) {
	row := d.Pool.QueryRow(ctx, `
		INSERT INTO alert_rules (device_id, sensor, threshold, condition, webhook_url, enabled)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, device_id, sensor, threshold, condition, webhook_url, enabled, created_at, updated_at
	`, in.DeviceID, in.Sensor, in.Threshold, string(in.Condition), in.WebhookURL, in.Enabled)

	return scanAlertRule(row)
}

// UpdateAlertRule modifies an existing rule and bumps updated_at.
func (d *DB) UpdateAlertRule(ctx context.Context, id uuid.UUID, in AlertRuleInput) (AlertRule, error) {
	row := d.Pool.QueryRow(ctx, `
		UPDATE alert_rules
		SET device_id = $2, sensor = $3, threshold = $4,
		    condition = $5, webhook_url = $6, enabled = $7,
		    updated_at = now()
		WHERE id = $1
		RETURNING id, device_id, sensor, threshold, condition, webhook_url, enabled, created_at, updated_at
	`, id, in.DeviceID, in.Sensor, in.Threshold, string(in.Condition), in.WebhookURL, in.Enabled)

	r, err := scanAlertRule(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return AlertRule{}, ErrAlertNotFound
	}
	return r, err
}

// DeleteAlertRule removes the rule. Returns ErrAlertNotFound when the id is
// unknown.
func (d *DB) DeleteAlertRule(ctx context.Context, id uuid.UUID) error {
	tag, err := d.Pool.Exec(ctx, `DELETE FROM alert_rules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("db: delete alert rule %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAlertNotFound
	}
	return nil
}

// scanner abstracts Row and Rows so we can share column ordering.
type scanner interface {
	Scan(dest ...any) error
}

func scanAlertRule(s scanner) (AlertRule, error) {
	var r AlertRule
	var cond string
	if err := s.Scan(
		&r.ID, &r.DeviceID, &r.Sensor, &r.Threshold,
		&cond, &r.WebhookURL, &r.Enabled, &r.CreatedAt, &r.UpdatedAt,
	); err != nil {
		return AlertRule{}, err
	}
	r.Condition = AlertCondition(cond)
	return r, nil
}
