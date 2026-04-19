package api

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
)

// handleHealth reports readiness across the critical backing services.
// Returns 200 when all components are reachable, 503 otherwise so container
// orchestrators can act on the response code alone.
func (s *Server) handleHealth(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
	defer cancel()

	dbStatus := "ok"
	if err := s.deps.DB.Ping(ctx); err != nil {
		dbStatus = "down"
	}

	status := fiber.StatusOK
	if dbStatus != "ok" {
		status = fiber.StatusServiceUnavailable
	}
	return c.Status(status).JSON(fiber.Map{
		"status": map[string]string{
			"db": dbStatus,
		},
	})
}
