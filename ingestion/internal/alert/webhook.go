package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// Dispatcher delivers Event payloads to webhook URLs over HTTP POST.
type Dispatcher struct {
	client *http.Client
}

// NewDispatcher returns a dispatcher with a bounded HTTP client. The 10-second
// timeout matches the value specified in ROADMAP Phase 4-3.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Dispatch sends the event to webhookURL. On network failure it performs a
// single retry after a short backoff. HTTP 2xx and 3xx are treated as success;
// 4xx/5xx responses are not retried (client/server error is unlikely to be
// transient from the user's perspective).
func (d *Dispatcher) Dispatch(ctx context.Context, webhookURL string, ev Event) error {
	if err := validateWebhookURL(webhookURL); err != nil {
		slog.Warn("alert: webhook url invalid",
			"err", err, "url", webhookURL,
			"device_id", ev.DeviceID, "sensor", ev.Sensor)
		return err
	}

	body, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("alert: marshal event: %w", err)
	}

	const maxAttempts = 2
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := d.post(ctx, webhookURL, body)
		if err == nil {
			return nil
		}
		lastErr = err

		// Don't retry on non-transient failures. Only retry network errors
		// (timeout, connection refused, TLS handshake, etc.).
		if isPermanent(err) {
			break
		}
		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(500 * time.Millisecond):
			}
		}
	}
	slog.Warn("alert: webhook dispatch failed",
		"err", lastErr, "url", webhookURL,
		"device_id", ev.DeviceID, "sensor", ev.Sensor)
	return lastErr
}

func (d *Dispatcher) post(ctx context.Context, webhookURL string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return permanentErr{err}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "nexos-alerting/1.0")

	resp, err := d.client.Do(req)
	if err != nil {
		return err // transient (network-level)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return permanentErr{fmt.Errorf("webhook returned status %d", resp.StatusCode)}
	}
	return nil
}

// validateWebhookURL enforces http/https scheme only. We do NOT block private
// IP ranges: IoT users commonly run webhook receivers on the same host
// (n8n, Node-RED, Home Assistant), and blocking them would regress the
// primary use case. The README documents this trade-off.
func validateWebhookURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return permanentErr{fmt.Errorf("alert: parse webhook url: %w", err)}
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return permanentErr{fmt.Errorf("alert: webhook scheme must be http or https, got %q", u.Scheme)}
	}
	if u.Host == "" {
		return permanentErr{fmt.Errorf("alert: webhook url missing host")}
	}
	return nil
}

// permanentErr marks errors that should not be retried.
type permanentErr struct{ error }

func (e permanentErr) Unwrap() error { return e.error }

func isPermanent(err error) bool {
	_, ok := err.(permanentErr)
	return ok
}
