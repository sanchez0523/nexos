package mqtt

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
)

// ErrInvalidPayload indicates the payload cannot be decoded into a numeric
// value. Callers should drop the message and log.
var ErrInvalidPayload = errors.New("mqtt: invalid payload")

// ParsePayload accepts two payload shapes:
//
//  1. A bare JSON number, e.g. `23.5` or `42`.
//  2. A JSON object with a numeric "value" field, e.g. `{"value": 23.5}`.
//
// Any other shape (strings, arrays, nested objects, missing "value" field,
// non-numeric "value") returns ErrInvalidPayload.
func ParsePayload(raw []byte) (float64, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return 0, ErrInvalidPayload
	}

	// Fast path: bare number
	if trimmed[0] != '{' {
		v, err := strconv.ParseFloat(string(trimmed), 64)
		if err != nil {
			return 0, ErrInvalidPayload
		}
		return v, nil
	}

	// Object path: require "value" as a JSON number
	var obj struct {
		Value *json.Number `json:"value"`
	}
	dec := json.NewDecoder(bytes.NewReader(trimmed))
	dec.UseNumber()
	if err := dec.Decode(&obj); err != nil {
		return 0, ErrInvalidPayload
	}
	if obj.Value == nil {
		return 0, ErrInvalidPayload
	}
	v, err := obj.Value.Float64()
	if err != nil {
		return 0, ErrInvalidPayload
	}
	return v, nil
}
