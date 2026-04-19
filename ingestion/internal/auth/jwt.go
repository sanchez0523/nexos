package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenType distinguishes access tokens (short-lived, used on every request)
// from refresh tokens (long-lived, only exchanged at /api/auth/refresh).
type TokenType string

const (
	TokenAccess  TokenType = "access"
	TokenRefresh TokenType = "refresh"

	issuer = "nexos"
)

// ErrInvalidToken is returned whenever a token fails signature, expiry, or
// claim validation. Always returned opaquely to avoid leaking which step
// failed.
var ErrInvalidToken = errors.New("auth: invalid token")

// Claims is the typed payload baked into both access and refresh tokens.
type Claims struct {
	Subject   string    `json:"sub"`
	TokenType TokenType `json:"typ"`
	jwt.RegisteredClaims
}

// Issuer signs tokens using a shared secret. It is stateless — revocation
// requires rotating the secret.
type Issuer struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	now        func() time.Time
}

// NewIssuer validates the secret length (≥32 bytes per CLAUDE.md security
// constraint) and returns a ready-to-use issuer.
func NewIssuer(secret string, accessTTL, refreshTTL time.Duration) (*Issuer, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("auth: jwt secret must be at least 32 bytes (got %d)", len(secret))
	}
	if accessTTL <= 0 || refreshTTL <= 0 {
		return nil, errors.New("auth: token TTLs must be positive")
	}
	if refreshTTL <= accessTTL {
		return nil, errors.New("auth: refresh TTL must exceed access TTL")
	}
	return &Issuer{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		now:        time.Now,
	}, nil
}

// TokenPair is the response body from /api/auth/login and /api/auth/refresh.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // seconds until access token expiry
}

// Issue mints a fresh (access, refresh) pair for the given subject (admin
// username). Both tokens are signed with the shared secret.
func (i *Issuer) Issue(subject string) (TokenPair, error) {
	now := i.now()
	access, err := i.sign(subject, TokenAccess, now, i.accessTTL)
	if err != nil {
		return TokenPair{}, err
	}
	refresh, err := i.sign(subject, TokenRefresh, now, i.refreshTTL)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(i.accessTTL.Seconds()),
	}, nil
}

// Rotate validates a refresh token and mints a brand-new pair (both tokens
// rotate — old refresh is implicitly invalidated once consumed, enforced by
// the single-admin model where the admin stores only the latest pair).
func (i *Issuer) Rotate(refreshToken string) (TokenPair, error) {
	claims, err := i.Parse(refreshToken)
	if err != nil {
		return TokenPair{}, err
	}
	if claims.TokenType != TokenRefresh {
		return TokenPair{}, ErrInvalidToken
	}
	return i.Issue(claims.Subject)
}

// Parse validates signature, expiry, issuer, and returns typed claims.
func (i *Issuer) Parse(raw string) (*Claims, error) {
	tok, err := jwt.ParseWithClaims(raw, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return i.secret, nil
	}, jwt.WithIssuer(issuer), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !tok.Valid {
		return nil, ErrInvalidToken
	}
	claims, ok := tok.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// RequireAccess validates the token and additionally enforces that it is an
// access token (not a refresh). Used by the cookie-based auth middleware and
// the WebSocket upgrade handler.
func (i *Issuer) RequireAccess(raw string) (*Claims, error) {
	claims, err := i.Parse(raw)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != TokenAccess {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func (i *Issuer) sign(subject string, typ TokenType, now time.Time, ttl time.Duration) (string, error) {
	claims := Claims{
		Subject:   subject,
		TokenType: typ,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   subject,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(i.secret)
	if err != nil {
		return "", fmt.Errorf("auth: sign %s token: %w", typ, err)
	}
	return signed, nil
}
