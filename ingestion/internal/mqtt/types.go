package mqtt

import "time"

// Metric is a single sensor reading extracted from an incoming MQTT message.
// It flows from the subscriber goroutine through a buffered channel to the DB
// writer and the WebSocket hub.
type Metric struct {
	Time     time.Time `json:"time"`
	DeviceID string    `json:"device_id"`
	Sensor   string    `json:"sensor"`
	Value    float64   `json:"value"`
}
