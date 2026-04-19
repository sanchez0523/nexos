package auth

import (
	"crypto/sha256"
	"crypto/subtle"
)

// VerifyAdminCredentials performs a constant-time comparison against the
// configured admin username and password. We hash both sides with SHA-256
// before comparing so the fixed-length byte slices are safe to pass to
// subtle.ConstantTimeCompare regardless of input length.
//
// Admin credentials are read from env vars on startup. bcrypt is intentionally
// avoided here because:
//   - Only ONE admin exists (single-user model per CLAUDE.md invariant).
//   - The plaintext password already lives in `.env` (operator-chosen).
//   - bcrypt would add dependency and offer no real uplift for a single
//     credential pair that never leaves memory.
func VerifyAdminCredentials(expectedUser, expectedPass, gotUser, gotPass string) bool {
	eu := sha256.Sum256([]byte(expectedUser))
	gu := sha256.Sum256([]byte(gotUser))
	ep := sha256.Sum256([]byte(expectedPass))
	gp := sha256.Sum256([]byte(gotPass))

	userOK := subtle.ConstantTimeCompare(eu[:], gu[:]) == 1
	passOK := subtle.ConstantTimeCompare(ep[:], gp[:]) == 1
	return userOK && passOK
}
