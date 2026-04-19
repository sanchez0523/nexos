package api

import (
	"context"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/nexos-io/nexos/ingestion/internal/api/ws"
	"github.com/nexos-io/nexos/ingestion/internal/auth"
	"github.com/nexos-io/nexos/ingestion/internal/db"
)

// Deps bundles everything a Server needs from main.go. Keeping it small
// prevents the API package from reaching into config structs it shouldn't know
// about.
type Deps struct {
	DB            *db.DB
	Issuer        *auth.Issuer
	Hub           *ws.Hub
	AdminUsername string
	AdminPassword string
}

// Server owns the Fiber app plus handler dependencies. Construct once in
// main.go via New() and call Listen()/Shutdown() to manage the lifecycle.
type Server struct {
	app  *fiber.App
	deps Deps
}

// New wires middleware and registers every route. Returns a ready-to-listen
// server. The caller is responsible for Listen + Shutdown lifecycle.
func New(deps Deps) *Server {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ReadTimeout:           10 * time.Second,
		WriteTimeout:          10 * time.Second,
		AppName:               "nexos-ingestion",
		ErrorHandler:          errorHandler,
	})

	app.Use(recover.New())
	app.Use(requestLogger())
	app.Use(cors.New(cors.Config{
		// Nexos is same-origin by design; the listed origins cover the two
		// supported local-install entry points (HTTP port 80 and HTTPS port
		// 443 via Caddy). AllowCredentials is required so httpOnly auth
		// cookies attach on fetch() calls from the dashboard.
		AllowOrigins:     "http://localhost,https://localhost",
		AllowHeaders:     "Origin,Content-Type,Accept",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowCredentials: true,
	}))

	s := &Server{app: app, deps: deps}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	// Public: health + auth
	s.app.Get("/health", s.handleHealth)
	s.app.Post("/api/auth/login", s.handleLogin)
	s.app.Post("/api/auth/refresh", s.handleRefresh)
	s.app.Post("/api/auth/logout", s.handleLogout)

	// WebSocket uses query-param token (browsers can't set headers on `new WebSocket`).
	s.app.Get("/ws", s.handleWebSocket())

	// Authenticated API surface.
	authed := s.app.Group("/api", requireAuth(s.deps.Issuer))
	authed.Get("/devices", s.handleListDevices)
	authed.Get("/devices/:id/sensors", s.handleListSensors)
	authed.Get("/metrics", s.handleListMetrics)
	s.registerAlertRoutes(authed)
}

// Listen blocks until the server stops or ctx triggers Shutdown externally.
func (s *Server) Listen(addr string) error { return s.app.Listen(addr) }

// Shutdown drains active connections with a deadline.
func (s *Server) Shutdown(ctx context.Context) error { return s.app.ShutdownWithContext(ctx) }

// ── middleware helpers ───────────────────────────────────────────────────────

func requestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		slog.Info("http",
			"method", c.Method(),
			"path", c.Path(),
			"status", c.Response().StatusCode(),
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return err
	}
}

func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}
	slog.Error("http: unhandled", "err", err, "path", c.Path())
	return c.Status(code).JSON(fiber.Map{"error": "internal error"})
}
