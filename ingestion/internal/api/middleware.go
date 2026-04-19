package api

import (
	"github.com/gofiber/fiber/v2"

	"github.com/nexos-io/nexos/ingestion/internal/auth"
)

const ctxKeySubject = "nexos.subject"

// requireAuth returns a Fiber middleware that validates the access token from
// the nexos_access httpOnly cookie. Stores the subject claim on the Fiber
// context under ctxKeySubject for downstream handlers.
//
// Cookie-based auth means:
//   - Browser JS cannot read the token (XSS safer).
//   - Cross-site requests never attach the cookie (CSRF safer thanks to
//     SameSite=Strict).
//   - No fallback to Authorization: Bearer — scripting/CLI callers should
//     obtain a session via /api/auth/login and reuse the cookie jar.
func requireAuth(iss *auth.Issuer) fiber.Handler {
	return func(c *fiber.Ctx) error {
		raw := c.Cookies(cookieAccessToken)
		if raw == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "authentication required"})
		}
		claims, err := iss.RequireAccess(raw)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}
		c.Locals(ctxKeySubject, claims.Subject)
		return c.Next()
	}
}
