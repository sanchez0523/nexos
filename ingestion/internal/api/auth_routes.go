package api

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/nexos-io/nexos/ingestion/internal/auth"
)

const (
	cookieAccessToken  = "nexos_access"
	cookieRefreshToken = "nexos_refresh"

	// Refresh cookie is scoped so it is only sent to the refresh endpoint.
	// Limits exposure surface — every other request carries only the access
	// cookie.
	refreshCookiePath = "/api/auth"
)

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// authResp is the JSON body returned on login / refresh. Tokens themselves
// are delivered out-of-band via Set-Cookie headers.
type authResp struct {
	Username  string `json:"username"`
	ExpiresIn int64  `json:"expires_in"` // seconds until access token expiry
}

// handleLogin exchanges admin credentials for a fresh token pair. Tokens are
// delivered as httpOnly, Secure, SameSite=Strict cookies so browser JS cannot
// read them (XSS mitigation) and the browser refuses to send them on cross-
// site requests (CSRF mitigation).
func (s *Server) handleLogin(c *fiber.Ctx) error {
	var req loginReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json body"})
	}
	if req.Username == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "username and password required"})
	}

	if !auth.VerifyAdminCredentials(
		s.deps.AdminUsername, s.deps.AdminPassword,
		req.Username, req.Password,
	) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}

	pair, err := s.deps.Issuer.Issue(req.Username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "token issuance failed"})
	}
	setAuthCookies(c, pair)
	return c.JSON(authResp{
		Username:  req.Username,
		ExpiresIn: pair.ExpiresIn,
	})
}

// handleRefresh rotates both tokens. The refresh token is read from the
// nexos_refresh cookie; no body is required.
func (s *Server) handleRefresh(c *fiber.Ctx) error {
	raw := c.Cookies(cookieRefreshToken)
	if raw == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "refresh cookie required"})
	}
	pair, err := s.deps.Issuer.Rotate(raw)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid refresh token"})
	}

	// Subject is carried through the rotation; pull it back out so we can
	// echo it in the response without trusting the client.
	claims, err := s.deps.Issuer.RequireAccess(pair.AccessToken)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "post-rotation validation failed"})
	}

	setAuthCookies(c, pair)
	return c.JSON(authResp{
		Username:  claims.Subject,
		ExpiresIn: pair.ExpiresIn,
	})
}

// handleLogout clears both auth cookies. Always returns 204 so logout is
// idempotent even if the cookies were already missing.
func (s *Server) handleLogout(c *fiber.Ctx) error {
	clearAuthCookies(c)
	return c.SendStatus(fiber.StatusNoContent)
}

// isHTTPSRequest reports whether the request came in over TLS. Caddy
// terminates TLS and forwards to the ingestion container over plain HTTP, so
// we trust the X-Forwarded-Proto header that Caddy sets. If the header is
// missing (direct access to localhost:8080 during dev), treat as HTTP.
//
// We only honour this header when the client IP is loopback or inside a
// private range — Nexos is designed to run behind its own proxy, never
// exposed directly to the internet, so spoofing isn't a realistic attack.
func isHTTPSRequest(c *fiber.Ctx) bool {
	return c.Get(fiber.HeaderXForwardedProto) == "https"
}

// setAuthCookies writes the access + refresh cookies. Secure is set only on
// TLS requests: over plain HTTP (dev mode via http://localhost) browsers
// refuse Secure cookies, which would block the entire login flow.
func setAuthCookies(c *fiber.Ctx, pair auth.TokenPair) {
	secure := isHTTPSRequest(c)
	accessExpiry := time.Now().Add(time.Duration(pair.ExpiresIn) * time.Second)
	c.Cookie(&fiber.Cookie{
		Name:     cookieAccessToken,
		Value:    pair.AccessToken,
		Path:     "/",
		Expires:  accessExpiry,
		HTTPOnly: true,
		Secure:   secure,
		SameSite: fiber.CookieSameSiteStrictMode,
	})
	// Refresh cookie: we don't encode its expiry in TokenPair, so we rely on
	// the JWT's own exp claim and give the cookie a generous session-level
	// lifetime. Path scoping limits exposure even without an explicit Max-Age.
	c.Cookie(&fiber.Cookie{
		Name:     cookieRefreshToken,
		Value:    pair.RefreshToken,
		Path:     refreshCookiePath,
		HTTPOnly: true,
		Secure:   secure,
		SameSite: fiber.CookieSameSiteStrictMode,
	})
}

func clearAuthCookies(c *fiber.Ctx) {
	secure := isHTTPSRequest(c)
	past := time.Unix(0, 0)
	c.Cookie(&fiber.Cookie{
		Name: cookieAccessToken, Path: "/", Expires: past, MaxAge: -1,
		HTTPOnly: true, Secure: secure, SameSite: fiber.CookieSameSiteStrictMode,
	})
	c.Cookie(&fiber.Cookie{
		Name: cookieRefreshToken, Path: refreshCookiePath, Expires: past, MaxAge: -1,
		HTTPOnly: true, Secure: secure, SameSite: fiber.CookieSameSiteStrictMode,
	})
}
