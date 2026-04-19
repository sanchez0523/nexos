package mqtt

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

// SubscriberConfig captures all broker connection parameters. It is populated
// from environment variables in main.go.
type SubscriberConfig struct {
	BrokerURL  string // mqtts://broker:8883
	Username   string
	Password   string
	CACertPath string // PEM file mounted from broker/certs/ca.crt
	ClientID   string // must be stable across reconnects
	Topic      string // subscribed topic filter, typically "devices/#"
}

// Subscriber owns the paho client lifecycle and converts incoming messages
// into Metric values published to Out.
type Subscriber struct {
	cfg    SubscriberConfig
	client paho.Client
	out    chan Metric
}

// NewSubscriber constructs a Subscriber with an allocated output channel of
// the given buffer size. The CLAUDE.md default is 256.
func NewSubscriber(cfg SubscriberConfig, bufferSize int) *Subscriber {
	return &Subscriber{cfg: cfg, out: make(chan Metric, bufferSize)}
}

// Out exposes the metric stream. The channel is closed when Run returns.
func (s *Subscriber) Out() <-chan Metric { return s.out }

// Run connects to the broker, subscribes, and blocks until ctx is cancelled.
// Paho manages reconnection internally; this method only exits on ctx.Done.
func (s *Subscriber) Run(ctx context.Context) error {
	tlsCfg, err := loadTLSConfig(s.cfg.CACertPath)
	if err != nil {
		return fmt.Errorf("mqtt: load tls: %w", err)
	}

	opts := paho.NewClientOptions().
		AddBroker(s.cfg.BrokerURL).
		SetClientID(s.cfg.ClientID).
		SetUsername(s.cfg.Username).
		SetPassword(s.cfg.Password).
		SetTLSConfig(tlsCfg).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(2 * time.Second).
		SetMaxReconnectInterval(30 * time.Second).
		SetCleanSession(false).
		SetOrderMatters(false).
		SetKeepAlive(30 * time.Second).
		SetPingTimeout(10 * time.Second).
		SetOnConnectHandler(func(c paho.Client) {
			slog.Info("mqtt: connected", "broker", s.cfg.BrokerURL)
			if tok := c.Subscribe(s.cfg.Topic, 1, s.handle); tok.Wait() && tok.Error() != nil {
				slog.Error("mqtt: subscribe failed", "err", tok.Error(), "topic", s.cfg.Topic)
			}
		}).
		SetConnectionLostHandler(func(_ paho.Client, err error) {
			slog.Warn("mqtt: connection lost", "err", err)
		})

	s.client = paho.NewClient(opts)
	tok := s.client.Connect()
	if !tok.WaitTimeout(10 * time.Second) {
		return errors.New("mqtt: connect timed out")
	}
	if err := tok.Error(); err != nil {
		return fmt.Errorf("mqtt: connect: %w", err)
	}

	<-ctx.Done()
	s.client.Disconnect(500)
	close(s.out)
	return nil
}

func (s *Subscriber) handle(_ paho.Client, msg paho.Message) {
	topic := msg.Topic()
	deviceID, sensor, err := ParseTopic(topic)
	if err != nil {
		slog.Debug("mqtt: drop invalid topic", "topic", topic)
		return
	}
	value, err := ParsePayload(msg.Payload())
	if err != nil {
		slog.Debug("mqtt: drop invalid payload",
			"topic", topic, "device_id", deviceID, "sensor", sensor)
		return
	}

	metric := Metric{
		Time:     time.Now().UTC(),
		DeviceID: deviceID,
		Sensor:   sensor,
		Value:    value,
	}

	// Non-blocking send: if the downstream channel is saturated we drop the
	// message rather than stall the paho worker goroutine. This is the
	// documented trade-off in CLAUDE.md ("some message loss acceptable").
	select {
	case s.out <- metric:
	default:
		slog.Warn("mqtt: downstream channel full, dropping metric",
			"device_id", deviceID, "sensor", sensor)
	}
}

func loadTLSConfig(caPath string) (*tls.Config, error) {
	pem, err := os.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("read ca: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return nil, errors.New("append ca: no PEM blocks parsed")
	}
	return &tls.Config{
		RootCAs:    pool,
		MinVersion: tls.VersionTLS12,
	}, nil
}
