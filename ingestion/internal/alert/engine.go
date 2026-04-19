package alert

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/nexos-io/nexos/ingestion/internal/db"
	"github.com/nexos-io/nexos/ingestion/internal/mqtt"
)

// Engine evaluates incoming metrics against stored rules and dispatches
// webhooks for every rule that triggers (respecting per-rule cooldown).
type Engine struct {
	db         *db.DB
	dispatcher *Dispatcher
	cooldown   time.Duration

	mu          sync.Mutex
	lastFiredAt map[uuid.UUID]time.Time
}

// NewEngine constructs an engine. cooldown must be non-negative; zero disables
// cooldown entirely (every triggering metric dispatches a webhook).
func NewEngine(d *db.DB, cooldown time.Duration) *Engine {
	if cooldown < 0 {
		cooldown = 0
	}
	return &Engine{
		db:          d,
		dispatcher:  NewDispatcher(),
		cooldown:    cooldown,
		lastFiredAt: make(map[uuid.UUID]time.Time),
	}
}

// Run reads metrics from `in` until ctx is cancelled. Each metric triggers a
// DB lookup for matching rules; any triggered webhook is dispatched in its
// own goroutine so slow webhook targets cannot back up the metric pipeline.
func (e *Engine) Run(ctx context.Context, in <-chan mqtt.Metric) {
	for {
		select {
		case <-ctx.Done():
			return
		case m, ok := <-in:
			if !ok {
				return
			}
			e.evaluate(ctx, m)
		}
	}
}

func (e *Engine) evaluate(ctx context.Context, m mqtt.Metric) {
	rules, err := e.db.ListEnabledAlertRulesFor(ctx, m.DeviceID, m.Sensor)
	if err != nil {
		slog.Error("alert: rule lookup failed", "err", err,
			"device_id", m.DeviceID, "sensor", m.Sensor)
		return
	}
	for _, r := range rules {
		if !triggered(r, m.Value) {
			continue
		}
		if !e.shouldFire(r.ID, m.Time) {
			continue
		}
		ev := Event{
			DeviceID:    m.DeviceID,
			Sensor:      m.Sensor,
			Value:       m.Value,
			Threshold:   r.Threshold,
			Condition:   string(r.Condition),
			TriggeredAt: m.Time,
		}
		go func(url string) {
			// Detached context so dispatch survives the parent ctx cancel
			// on rule evaluation completion. Budget covers both retry
			// attempts (2 × 10s HTTP timeout + 500ms backoff + slack).
			dispatchCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_ = e.dispatcher.Dispatch(dispatchCtx, url, ev)
		}(r.WebhookURL)
	}
}

// triggered reports whether the value crosses the rule's threshold. The
// semantics match user expectations:
//   - "above"  ⇒ value > threshold  (strict)
//   - "below"  ⇒ value < threshold  (strict)
//
// Strict comparison prevents a reading that equals the threshold from firing
// repeatedly in both directions.
func triggered(r db.AlertRule, value float64) bool {
	switch r.Condition {
	case db.ConditionAbove:
		return value > r.Threshold
	case db.ConditionBelow:
		return value < r.Threshold
	default:
		return false
	}
}

// shouldFire consults the in-memory cooldown map and records a fresh
// timestamp when firing is allowed. Serialized by a mutex because rules for
// different (device_id, sensor) can evaluate concurrently once Run spawns
// multiple evaluation goroutines in the future.
func (e *Engine) shouldFire(ruleID uuid.UUID, at time.Time) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	last, ok := e.lastFiredAt[ruleID]
	if ok && at.Sub(last) < e.cooldown {
		return false
	}
	e.lastFiredAt[ruleID] = at
	return true
}
