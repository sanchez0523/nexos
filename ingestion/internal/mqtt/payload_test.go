package mqtt

import (
	"errors"
	"math"
	"testing"
)

func TestParsePayload(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    float64
		wantErr error
	}{
		// Bare number
		{"integer", "42", 42, nil},
		{"float", "23.5", 23.5, nil},
		{"negative float", "-5.7", -5.7, nil},
		{"zero", "0", 0, nil},
		{"scientific", "1e3", 1000, nil},

		// Object form
		{"object float", `{"value": 23.5}`, 23.5, nil},
		{"object integer", `{"value": 42}`, 42, nil},
		{"object negative", `{"value": -17}`, -17, nil},
		{"object with whitespace", `  {"value": 1.23}  `, 1.23, nil},
		{"object with extra fields", `{"value": 9.9, "unit": "C"}`, 9.9, nil},

		// Invalid
		{"empty", "", 0, ErrInvalidPayload},
		{"whitespace only", "   ", 0, ErrInvalidPayload},
		{"string value", `"hello"`, 0, ErrInvalidPayload},
		{"null", "null", 0, ErrInvalidPayload},
		{"array", "[1,2,3]", 0, ErrInvalidPayload},
		{"object missing value", `{"temp": 23.5}`, 0, ErrInvalidPayload},
		{"object value as string", `{"value": "23.5"}`, 0, ErrInvalidPayload},
		{"object value null", `{"value": null}`, 0, ErrInvalidPayload},
		{"malformed json", `{"value":`, 0, ErrInvalidPayload},
		{"not a number", "abc", 0, ErrInvalidPayload},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePayload([]byte(tt.raw))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if err == nil && math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("got = %v, want %v", got, tt.want)
			}
		})
	}
}
