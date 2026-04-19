package mqtt

import (
	"errors"
	"strings"
)

// ErrInvalidTopic indicates the topic does not match the required convention
// `devices/{device_id}/{sensor}`. Callers should drop the message and log.
var ErrInvalidTopic = errors.New("mqtt: invalid topic format")

// ParseTopic extracts device_id and sensor from a topic that must match
// `devices/{device_id}/{sensor}`. Both identifiers must be non-empty and must
// not contain '/', '+', or '#' (MQTT wildcards).
func ParseTopic(topic string) (deviceID, sensor string, err error) {
	parts := strings.Split(topic, "/")
	if len(parts) != 3 {
		return "", "", ErrInvalidTopic
	}
	if parts[0] != "devices" {
		return "", "", ErrInvalidTopic
	}

	deviceID, sensor = parts[1], parts[2]
	if deviceID == "" || sensor == "" {
		return "", "", ErrInvalidTopic
	}
	if containsWildcard(deviceID) || containsWildcard(sensor) {
		return "", "", ErrInvalidTopic
	}

	return deviceID, sensor, nil
}

func containsWildcard(s string) bool {
	return strings.ContainsAny(s, "+#")
}
