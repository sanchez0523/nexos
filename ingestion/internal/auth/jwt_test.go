package auth

import (
	"errors"
	"strings"
	"testing"
	"time"
)

const testSecret = "abcdefghijklmnopqrstuvwxyz0123456789" // 36 bytes

func newIssuer(t *testing.T) *Issuer {
	t.Helper()
	iss, err := NewIssuer(testSecret, 24*time.Hour, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("NewIssuer: %v", err)
	}
	return iss
}

func TestNewIssuer_ValidatesSecretLength(t *testing.T) {
	_, err := NewIssuer("too-short", time.Hour, time.Hour*2)
	if err == nil {
		t.Fatal("expected error for short secret")
	}
}

func TestNewIssuer_ValidatesTTLs(t *testing.T) {
	if _, err := NewIssuer(testSecret, 0, time.Hour); err == nil {
		t.Fatal("expected error for zero access TTL")
	}
	if _, err := NewIssuer(testSecret, time.Hour, time.Minute); err == nil {
		t.Fatal("expected error when refresh TTL <= access TTL")
	}
}

func TestIssueAndParse(t *testing.T) {
	iss := newIssuer(t)

	pair, err := iss.Issue("admin")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("expected both tokens populated")
	}
	if pair.ExpiresIn != int64((24 * time.Hour).Seconds()) {
		t.Errorf("ExpiresIn = %d, want %d", pair.ExpiresIn, int64((24*time.Hour).Seconds()))
	}

	access, err := iss.Parse(pair.AccessToken)
	if err != nil {
		t.Fatalf("Parse access: %v", err)
	}
	if access.Subject != "admin" || access.TokenType != TokenAccess {
		t.Errorf("unexpected access claims: %+v", access)
	}

	refresh, err := iss.Parse(pair.RefreshToken)
	if err != nil {
		t.Fatalf("Parse refresh: %v", err)
	}
	if refresh.TokenType != TokenRefresh {
		t.Errorf("refresh TokenType = %q, want %q", refresh.TokenType, TokenRefresh)
	}
}

func TestRequireAccess_RejectsRefreshToken(t *testing.T) {
	iss := newIssuer(t)
	pair, _ := iss.Issue("admin")

	if _, err := iss.RequireAccess(pair.AccessToken); err != nil {
		t.Fatalf("RequireAccess on access token: %v", err)
	}
	if _, err := iss.RequireAccess(pair.RefreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("RequireAccess on refresh token: got %v, want ErrInvalidToken", err)
	}
}

func TestRotate_RejectsAccessToken(t *testing.T) {
	iss := newIssuer(t)
	pair, _ := iss.Issue("admin")

	if _, err := iss.Rotate(pair.AccessToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Rotate on access token: got %v, want ErrInvalidToken", err)
	}

	newPair, err := iss.Rotate(pair.RefreshToken)
	if err != nil {
		t.Fatalf("Rotate on refresh token: %v", err)
	}
	if newPair.AccessToken == pair.AccessToken {
		t.Error("expected new access token to differ from original")
	}
	if newPair.RefreshToken == pair.RefreshToken {
		t.Error("expected new refresh token to differ from original")
	}
}

func TestParse_RejectsExpired(t *testing.T) {
	iss := newIssuer(t)
	// Manually construct an issuer with past time
	iss.now = func() time.Time { return time.Now().Add(-48 * time.Hour) }
	pair, _ := iss.Issue("admin")

	// Restore real time → token has already expired
	iss.now = time.Now
	if _, err := iss.Parse(pair.AccessToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Parse expired: got %v, want ErrInvalidToken", err)
	}
}

func TestParse_RejectsTamperedSignature(t *testing.T) {
	iss := newIssuer(t)
	pair, _ := iss.Issue("admin")

	tampered := pair.AccessToken[:len(pair.AccessToken)-3] + "xyz"
	if _, err := iss.Parse(tampered); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Parse tampered: got %v, want ErrInvalidToken", err)
	}
}

func TestParse_RejectsWrongSecret(t *testing.T) {
	issA := newIssuer(t)
	issB, _ := NewIssuer(strings.Repeat("z", 32), time.Hour, 24*time.Hour)

	pair, _ := issA.Issue("admin")
	if _, err := issB.Parse(pair.AccessToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("cross-secret parse: got %v, want ErrInvalidToken", err)
	}
}

func TestParse_RejectsNoneAlgorithm(t *testing.T) {
	iss := newIssuer(t)
	// Classic JWT 'none' algorithm attack: a token claiming alg=none
	// must be rejected outright by jwt.WithValidMethods.
	maliciousToken := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiJhZG1pbiJ9."
	if _, err := iss.Parse(maliciousToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Parse alg=none: got %v, want ErrInvalidToken", err)
	}
}
