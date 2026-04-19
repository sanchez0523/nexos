package api

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// onlineThreshold is how recently a device must have published a message to
// be considered "online". Matches the convention in ROADMAP Phase 3-3.
const onlineThreshold = 60 * time.Second

type deviceDTO struct {
	DeviceID  string    `json:"device_id"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
	Online    bool      `json:"online"`
}

// handleListDevices returns every registered device with an online flag
// derived at response time (not stored). Cheap even on large device counts
// because it's a single table scan with no joins.
func (s *Server) handleListDevices(c *fiber.Ctx) error {
	devices, err := s.deps.DB.ListDevices(c.UserContext())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "list devices failed"})
	}

	now := time.Now().UTC()
	out := make([]deviceDTO, 0, len(devices))
	for _, d := range devices {
		out = append(out, deviceDTO{
			DeviceID:  d.DeviceID,
			FirstSeen: d.FirstSeen,
			LastSeen:  d.LastSeen,
			Online:    now.Sub(d.LastSeen) <= onlineThreshold,
		})
	}
	return c.JSON(fiber.Map{"devices": out})
}

// handleListSensors returns the sensors ever observed for a device. Used by
// the dashboard to render the list of cards under a device.
func (s *Server) handleListSensors(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "device id required"})
	}
	sensors, err := s.deps.DB.ListSensorsByDevice(c.UserContext(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "list sensors failed"})
	}
	return c.JSON(fiber.Map{
		"device_id": id,
		"sensors":   sensors,
	})
}
