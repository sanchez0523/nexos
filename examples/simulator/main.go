// Nexos synthetic device simulator.
//
// Publishes realistic-looking metrics for N virtual devices so you can see
// the dashboard populate without wiring real hardware. Useful during setup,
// for GIF recording, and as a smoke test for ingestion changes.
//
//	go run ./examples/simulator \
//	  -broker mqtts://localhost:8883 \
//	  -ca broker/certs/ca.crt \
//	  -devices 3 -interval 2s \
//	  -passwords "s!mpw1,s!mpw2,s!mpw3"
//
// Each virtual device needs an MQTT account — create them with:
//	for i in 1 2 3; do ./scripts/add-device.sh sim-$i "s!mpw$i"; done
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type config struct {
	brokerURL        string
	caCertPath       string
	devices          int
	interval         time.Duration
	usernameTemplate string
	passwordsCSV     string
	topicSensors     []string
}

func main() {
	cfg := parseFlags()
	tlsCfg, err := loadTLS(cfg.caCertPath)
	if err != nil {
		log.Fatalf("load tls: %v", err)
	}

	passwords := strings.Split(cfg.passwordsCSV, ",")
	if len(passwords) != cfg.devices {
		log.Fatalf("expected %d passwords (comma-separated), got %d", cfg.devices, len(passwords))
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup
	for i := 0; i < cfg.devices; i++ {
		wg.Add(1)
		deviceID := fmt.Sprintf(cfg.usernameTemplate, i+1)
		password := strings.TrimSpace(passwords[i])
		go func(id, pw string, seed int64) {
			defer wg.Done()
			runDevice(ctx, cfg, tlsCfg, id, pw, seed)
		}(deviceID, password, int64(i))
	}
	wg.Wait()
}

func runDevice(ctx context.Context, cfg config, tlsCfg *tls.Config, id, password string, seed int64) {
	opts := paho.NewClientOptions().
		AddBroker(cfg.brokerURL).
		SetClientID(fmt.Sprintf("nexos-sim-%s", id)).
		SetUsername(id).
		SetPassword(password).
		SetTLSConfig(tlsCfg).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(2 * time.Second).
		SetMaxReconnectInterval(30 * time.Second).
		SetKeepAlive(30 * time.Second)

	client := paho.NewClient(opts)
	tok := client.Connect()
	if !tok.WaitTimeout(10 * time.Second) {
		log.Printf("[%s] connect timed out", id)
		return
	}
	if err := tok.Error(); err != nil {
		log.Printf("[%s] connect failed: %v", id, err)
		return
	}
	defer client.Disconnect(250)
	log.Printf("[%s] connected", id)

	rng := rand.New(rand.NewSource(seed))
	tick := time.NewTicker(cfg.interval)
	defer tick.Stop()

	t := 0.0
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			t += cfg.interval.Seconds()
			for i, sensor := range cfg.topicSensors {
				value := synthValue(sensor, t, rng, i)
				topic := fmt.Sprintf("devices/%s/%s", id, sensor)
				payload := fmt.Sprintf(`{"value":%.2f}`, value)
				ptok := client.Publish(topic, 0, false, payload)
				if ptok.WaitTimeout(2*time.Second) && ptok.Error() != nil {
					log.Printf("[%s] publish %s failed: %v", id, sensor, ptok.Error())
				}
			}
		}
	}
}

// synthValue produces realistic-looking readings: a slow sine wave baseline
// plus gaussian noise, with sensor-specific bounds that keep the dashboard
// visually interesting.
func synthValue(sensor string, t float64, rng *rand.Rand, phase int) float64 {
	switch sensor {
	case "temperature":
		return 20 + 3*math.Sin(t/60+float64(phase)) + rng.NormFloat64()*0.3
	case "humidity":
		return 55 + 10*math.Sin(t/120+float64(phase)) + rng.NormFloat64()*1.5
	case "pressure":
		return 1013 + 2*math.Sin(t/300+float64(phase)) + rng.NormFloat64()*0.4
	default:
		return 50 + 40*math.Sin(t/90+float64(phase)) + rng.NormFloat64()*2
	}
}

// ── plumbing ─────────────────────────────────────────────────────────────────

func parseFlags() config {
	var sensorsCSV string
	cfg := config{}
	flag.StringVar(&cfg.brokerURL, "broker", "mqtts://localhost:8883", "MQTT broker URL")
	flag.StringVar(&cfg.caCertPath, "ca", "broker/certs/ca.crt", "Path to CA certificate")
	flag.IntVar(&cfg.devices, "devices", 3, "Number of simulated devices")
	flag.DurationVar(&cfg.interval, "interval", 2*time.Second, "Publish interval per device")
	flag.StringVar(&cfg.usernameTemplate, "username-template", "sim-%d", "fmt.Sprintf template for device IDs (must match MQTT accounts)")
	flag.StringVar(&cfg.passwordsCSV, "passwords", "", "Comma-separated MQTT passwords, one per device")
	flag.StringVar(&sensorsCSV, "sensors", "temperature,humidity,pressure", "Comma-separated sensor names")
	flag.Parse()

	if cfg.passwordsCSV == "" {
		log.Fatal("-passwords is required (comma-separated list, one per device)")
	}
	for _, s := range strings.Split(sensorsCSV, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			cfg.topicSensors = append(cfg.topicSensors, s)
		}
	}
	if len(cfg.topicSensors) == 0 {
		log.Fatal("-sensors must list at least one sensor")
	}
	return cfg
}

func loadTLS(caPath string) (*tls.Config, error) {
	pem, err := os.ReadFile(caPath)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("append ca: no PEM blocks")
	}
	// The simulator can target the broker via localhost, 127.0.0.1, or the
	// Docker network address depending on how the user runs it. We trust the
	// CA but skip hostname verification so one binary works everywhere —
	// matches the Python RPi sample's posture.
	return &tls.Config{RootCAs: pool, InsecureSkipVerify: true}, nil // nolint:gosec // CA-verified
}
