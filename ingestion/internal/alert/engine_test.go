package alert

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/nexos-io/nexos/ingestion/internal/db"
)

func TestTriggered(t *testing.T) {
	tests := []struct {
		name      string
		condition db.AlertCondition
		threshold float64
		value     float64
		want      bool
	}{
		{"above strict greater", db.ConditionAbove, 10, 11, true},
		{"above exactly equal", db.ConditionAbove, 10, 10, false},
		{"above smaller", db.ConditionAbove, 10, 9, false},
		{"below strict smaller", db.ConditionBelow, 10, 9, true},
		{"below exactly equal", db.ConditionBelow, 10, 10, false},
		{"below larger", db.ConditionBelow, 10, 11, false},
		{"above negative threshold", db.ConditionAbove, -5, -4, true},
		{"below negative threshold", db.ConditionBelow, -5, -6, true},
		{"unknown condition", db.AlertCondition("weird"), 10, 15, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := triggered(db.AlertRule{
				Condition: tt.condition,
				Threshold: tt.threshold,
			}, tt.value)
			if got != tt.want {
				t.Errorf("triggered = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldFire_Cooldown(t *testing.T) {
	e := &Engine{
		cooldown:    60 * time.Second,
		lastFiredAt: make(map[uuid.UUID]time.Time),
	}
	id := uuid.New()
	base := time.Now()

	if !e.shouldFire(id, base) {
		t.Fatal("first fire should be allowed")
	}
	if e.shouldFire(id, base.Add(30*time.Second)) {
		t.Fatal("30s after first fire should still be on cooldown (cooldown=60s)")
	}
	if !e.shouldFire(id, base.Add(61*time.Second)) {
		t.Fatal("61s after first fire should be allowed (cooldown=60s)")
	}
}

func TestShouldFire_ZeroCooldown(t *testing.T) {
	e := &Engine{
		cooldown:    0,
		lastFiredAt: make(map[uuid.UUID]time.Time),
	}
	id := uuid.New()
	now := time.Now()

	if !e.shouldFire(id, now) || !e.shouldFire(id, now) {
		t.Fatal("zero cooldown must allow every call")
	}
}

func TestShouldFire_PerRuleIndependent(t *testing.T) {
	e := &Engine{
		cooldown:    60 * time.Second,
		lastFiredAt: make(map[uuid.UUID]time.Time),
	}
	idA, idB := uuid.New(), uuid.New()
	now := time.Now()

	if !e.shouldFire(idA, now) {
		t.Fatal("rule A first fire")
	}
	if !e.shouldFire(idB, now) {
		t.Fatal("rule B cooldown should not carry over from rule A")
	}
}

func TestValidateWebhookURL(t *testing.T) {
	tests := []struct {
		url     string
		wantErr bool
	}{
		{"https://example.com/hook", false},
		{"http://localhost:3000/hook", false},
		{"http://192.168.1.10/webhook", false}, // private IPs explicitly allowed
		{"ftp://example.com/hook", true},
		{"https://", true},
		{"", true},
		{"not a url at all", true},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			err := validateWebhookURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateWebhookURL(%q) err = %v, wantErr = %v", tt.url, err, tt.wantErr)
			}
		})
	}
}
