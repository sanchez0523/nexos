// Package alert implements threshold evaluation and webhook dispatch for
// sensor metrics that cross user-configured limits.
//
// Design:
//   - The engine is purely a consumer: it reads incoming Metric values, looks
//     up matching rules from the DB, evaluates thresholds, and asks the
//     dispatcher to fire webhooks.
//   - Per-rule cooldown (ALERT_TIMEOUT_SECONDS) prevents a flapping sensor
//     from spamming the webhook target. Cooldown state lives in-memory; on
//     ingestion restart all cooldowns reset, which is acceptable for a
//     single-instance deployment.
//   - Webhooks are dispatched asynchronously in a worker goroutine so slow
//     downstream services never block the metric pipeline.
package alert

import "time"

// Event is the Generic JSON payload documented in CLAUDE.md. The field set is
// stable — downstream users wire this directly into Slack bridges, custom
// integrations, etc.
type Event struct {
	DeviceID    string    `json:"device_id"`
	Sensor      string    `json:"sensor"`
	Value       float64   `json:"value"`
	Threshold   float64   `json:"threshold"`
	Condition   string    `json:"condition"` // "above" or "below"
	TriggeredAt time.Time `json:"triggered_at"`
}
