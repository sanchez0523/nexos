package api

import (
	"errors"
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/nexos-io/nexos/ingestion/internal/db"
)

// registerAlertRoutes is called by registerRoutes (server.go) once on startup.
// Split out so the CRUD surface lives with the engine it feeds.
func (s *Server) registerAlertRoutes(r fiber.Router) {
	r.Get("/alerts", s.handleListAlerts)
	r.Post("/alerts", s.handleCreateAlert)
	r.Put("/alerts/:id", s.handleUpdateAlert)
	r.Delete("/alerts/:id", s.handleDeleteAlert)
}

func (s *Server) handleListAlerts(c *fiber.Ctx) error {
	rules, err := s.deps.DB.ListAlertRules(c.UserContext())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "list alerts failed"})
	}
	return c.JSON(fiber.Map{"alerts": rules})
}

func (s *Server) handleCreateAlert(c *fiber.Ctx) error {
	var in db.AlertRuleInput
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json body"})
	}
	if msg := validateAlertInput(in); msg != "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": msg})
	}

	rule, err := s.deps.DB.CreateAlertRule(c.UserContext(), in)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "create alert failed"})
	}
	return c.Status(fiber.StatusCreated).JSON(rule)
}

func (s *Server) handleUpdateAlert(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var in db.AlertRuleInput
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json body"})
	}
	if msg := validateAlertInput(in); msg != "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": msg})
	}

	rule, err := s.deps.DB.UpdateAlertRule(c.UserContext(), id, in)
	if errors.Is(err, db.ErrAlertNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "alert not found"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "update alert failed"})
	}
	return c.JSON(rule)
}

func (s *Server) handleDeleteAlert(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := s.deps.DB.DeleteAlertRule(c.UserContext(), id); errors.Is(err, db.ErrAlertNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "alert not found"})
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "delete alert failed"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// validateAlertInput returns a non-empty error message when the payload is
// invalid. Kept separate from the handlers so the same rule set applies to
// create and update.
func validateAlertInput(in db.AlertRuleInput) string {
	if in.DeviceID == "" {
		return "device_id required"
	}
	if in.Sensor == "" {
		return "sensor required"
	}
	if !in.Condition.Valid() {
		return `condition must be "above" or "below"`
	}
	if in.WebhookURL == "" {
		return "webhook_url required"
	}
	u, err := url.Parse(in.WebhookURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "webhook_url must be an http(s) URL"
	}
	return ""
}
