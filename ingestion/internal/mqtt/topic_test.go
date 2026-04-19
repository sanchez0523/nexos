package mqtt

import (
	"errors"
	"testing"
)

func TestParseTopic(t *testing.T) {
	tests := []struct {
		name     string
		topic    string
		wantID   string
		wantSens string
		wantErr  error
	}{
		{"valid basic", "devices/esp32-01/temperature", "esp32-01", "temperature", nil},
		{"valid with numeric suffix", "devices/rpi02/cpu_temp", "rpi02", "cpu_temp", nil},
		{"valid with dash and dot", "devices/sensor-1.a/humidity", "sensor-1.a", "humidity", nil},

		{"wrong prefix", "sensors/esp32-01/temperature", "", "", ErrInvalidTopic},
		{"too few parts", "devices/esp32-01", "", "", ErrInvalidTopic},
		{"too many parts", "devices/esp32-01/temp/inner", "", "", ErrInvalidTopic},
		{"empty device_id", "devices//temperature", "", "", ErrInvalidTopic},
		{"empty sensor", "devices/esp32-01/", "", "", ErrInvalidTopic},
		{"plus wildcard in device", "devices/+/temperature", "", "", ErrInvalidTopic},
		{"hash wildcard in sensor", "devices/esp32-01/#", "", "", ErrInvalidTopic},
		{"empty topic", "", "", "", ErrInvalidTopic},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deviceID, sensor, err := ParseTopic(tt.topic)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if deviceID != tt.wantID {
				t.Errorf("deviceID = %q, want %q", deviceID, tt.wantID)
			}
			if sensor != tt.wantSens {
				t.Errorf("sensor = %q, want %q", sensor, tt.wantSens)
			}
		})
	}
}
