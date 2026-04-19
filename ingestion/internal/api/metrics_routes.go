package api

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/nexos-io/nexos/ingestion/internal/db"
)

// handleListMetrics serves historical time-series data. Required query
// parameters: device_id, sensor, from, to (ISO-8601). Optional: limit.
//
// Windows over 1 hour are downsampled to 1-minute buckets by the DB layer.
func (s *Server) handleListMetrics(c *fiber.Ctx) error {
	deviceID := c.Query("device_id")
	sensor := c.Query("sensor")
	fromStr := c.Query("from")
	toStr := c.Query("to")

	if deviceID == "" || sensor == "" || fromStr == "" || toStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "device_id, sensor, from, to are required",
		})
	}

	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "from must be RFC3339"})
	}
	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "to must be RFC3339"})
	}

	limit := 0
	if v := c.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "limit must be positive integer"})
		}
		limit = n
	}

	points, err := s.deps.DB.QueryMetrics(c.UserContext(), db.MetricsQuery{
		DeviceID: deviceID,
		Sensor:   sensor,
		From:     from,
		To:       to,
		Limit:    limit,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "query failed"})
	}

	bucketed := to.Sub(from) > time.Hour
	return c.JSON(fiber.Map{
		"device_id": deviceID,
		"sensor":    sensor,
		"bucketed":  bucketed,
		"points":    points,
	})
}
