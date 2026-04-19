package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/nexos-io/nexos/ingestion/internal/alert"
	"github.com/nexos-io/nexos/ingestion/internal/api"
	"github.com/nexos-io/nexos/ingestion/internal/api/ws"
	"github.com/nexos-io/nexos/ingestion/internal/auth"
	nexosdb "github.com/nexos-io/nexos/ingestion/internal/db"
	nexosmqtt "github.com/nexos-io/nexos/ingestion/internal/mqtt"
)

const (
	// Fan-in buffer between the MQTT subscriber and the downstream consumers.
	// Matches the value documented in CLAUDE.md.
	metricsChannelBuffer = 256

	// Path inside the container where migrations are baked by the Dockerfile.
	migrationsPath = "/app/migrations"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := loadConfig()
	if err != nil {
		slog.Error("config: load failed", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ── DB ────────────────────────────────────────────────────────────────
	if err := nexosdb.RunMigrations(cfg.DatabaseURL, migrationsPath); err != nil {
		slog.Error("db: migrations failed", "err", err)
		os.Exit(1)
	}

	db, err := nexosdb.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("db: open failed", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.ReconcileRetentionPolicy(ctx, cfg.DataRetentionDays); err != nil {
		slog.Error("db: retention policy reconcile failed", "err", err)
		os.Exit(1)
	}
	slog.Info("db: ready", "retention_days", cfg.DataRetentionDays)

	// ── Auth ──────────────────────────────────────────────────────────────
	issuer, err := auth.NewIssuer(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	if err != nil {
		slog.Error("auth: issuer init failed", "err", err)
		os.Exit(1)
	}

	// ── WebSocket hub ─────────────────────────────────────────────────────
	hub := ws.NewHub()
	hubStop := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		hub.Run(hubStop)
	}()

	// ── MQTT subscriber ───────────────────────────────────────────────────
	sub := nexosmqtt.NewSubscriber(nexosmqtt.SubscriberConfig{
		BrokerURL:  cfg.MQTTBrokerURL,
		Username:   cfg.MQTTUsername,
		Password:   cfg.MQTTPassword,
		CACertPath: cfg.MQTTCACertPath,
		ClientID:   "nexos-ingestion",
		Topic:      "devices/#",
	}, metricsChannelBuffer)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := sub.Run(ctx); err != nil {
			slog.Error("mqtt: subscriber stopped", "err", err)
			stop()
		}
	}()

	// ── Alert engine ──────────────────────────────────────────────────────
	alertCooldown := time.Duration(cfg.AlertTimeoutSecs) * time.Second
	engine := alert.NewEngine(db, alertCooldown)
	alertIn := make(chan nexosmqtt.Metric, metricsChannelBuffer)
	wg.Add(1)
	go func() {
		defer wg.Done()
		engine.Run(ctx, alertIn)
	}()

	// ── Fan-out: subscriber → (writer, ws hub, alert engine) ──────────────
	writerIn := make(chan nexosmqtt.Metric, metricsChannelBuffer)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(writerIn)
		defer close(alertIn)
		for m := range sub.Out() {
			// DB writer: blocking send so metrics aren't dropped before
			// they reach the batcher. The batcher keeps up easily at this
			// scale.
			writerIn <- m
			// WebSocket hub: non-blocking — real-time view is best-effort.
			hub.Broadcast(ws.Event{
				DeviceID: m.DeviceID,
				Sensor:   m.Sensor,
				Value:    m.Value,
				Time:     m.Time.Format(time.RFC3339Nano),
			})
			// Alert engine: non-blocking to protect the pipeline from a
			// slow rule evaluation. If the engine's buffer fills, the
			// alert for this tick is dropped (accepted trade-off).
			select {
			case alertIn <- m:
			default:
				slog.Warn("alert: engine queue full, dropping metric",
					"device_id", m.DeviceID, "sensor", m.Sensor)
			}
		}
	}()

	// ── Metrics writer ────────────────────────────────────────────────────
	writer := nexosdb.NewMetricsWriter(db, writerIn)
	wg.Add(1)
	go func() {
		defer wg.Done()
		writer.Run(ctx)
	}()

	// ── HTTP server ───────────────────────────────────────────────────────
	server := api.New(api.Deps{
		DB:            db,
		Issuer:        issuer,
		Hub:           hub,
		AdminUsername: cfg.AdminUsername,
		AdminPassword: cfg.AdminPassword,
	})

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.Listen(":8080"); err != nil {
			slog.Error("http: server stopped", "err", err)
			stop()
		}
	}()
	slog.Info("http: listening", "addr", ":8080")

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("http: shutdown", "err", err)
	}
	close(hubStop)

	// Wait for subscriber, fanout, writer, hub, and server to drain.
	wg.Wait()
}

// ── Config ───────────────────────────────────────────────────────────────────

type config struct {
	DatabaseURL       string
	MQTTBrokerURL     string
	MQTTUsername      string
	MQTTPassword      string
	MQTTCACertPath    string
	DataRetentionDays int
	JWTSecret         string
	JWTAccessTTL      time.Duration
	JWTRefreshTTL     time.Duration
	AdminUsername     string
	AdminPassword     string
	AlertTimeoutSecs  int
}

func loadConfig() (*config, error) {
	c := &config{
		DatabaseURL:    mustEnv("DATABASE_URL"),
		MQTTBrokerURL:  mustEnv("MQTT_BROKER_URL"),
		MQTTUsername:   mustEnv("MQTT_USERNAME"),
		MQTTPassword:   mustEnv("MQTT_PASSWORD"),
		MQTTCACertPath: envOrDefault("MQTT_CA_CERT_PATH", "/certs/ca.crt"),
		JWTSecret:      mustEnv("JWT_SECRET"),
		AdminUsername:  mustEnv("ADMIN_USERNAME"),
		AdminPassword:  mustEnv("ADMIN_PASSWORD"),
	}

	if len(c.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 bytes (got %d)", len(c.JWTSecret))
	}

	days, err := strconv.Atoi(envOrDefault("DATA_RETENTION_DAYS", "90"))
	if err != nil || days <= 0 {
		return nil, fmt.Errorf("DATA_RETENTION_DAYS must be a positive integer")
	}
	c.DataRetentionDays = days

	c.JWTAccessTTL, err = time.ParseDuration(envOrDefault("JWT_ACCESS_TTL", "24h"))
	if err != nil {
		return nil, fmt.Errorf("JWT_ACCESS_TTL: %w", err)
	}
	c.JWTRefreshTTL, err = time.ParseDuration(envOrDefault("JWT_REFRESH_TTL", "168h"))
	if err != nil {
		return nil, fmt.Errorf("JWT_REFRESH_TTL: %w", err)
	}

	secs, err := strconv.Atoi(envOrDefault("ALERT_TIMEOUT_SECONDS", "60"))
	if err != nil || secs < 0 {
		return nil, fmt.Errorf("ALERT_TIMEOUT_SECONDS must be a non-negative integer")
	}
	c.AlertTimeoutSecs = secs

	return c, nil
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("config: missing required env var", "key", key)
		os.Exit(1)
	}
	return v
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
