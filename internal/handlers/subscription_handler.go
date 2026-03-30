package handlers

import (
	"database/sql"

	"gully-cricket/internal/services"
	"github.com/gofiber/fiber/v2"
)

func Subscribe(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		var req struct {
			PlanID    int `json:"plan_id"`
			TeamCount int `json:"team_count"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid"})
		}

		userID := c.Locals("user_id").(int)
		err := services.SubscribeUser(db, userID, planID)

		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"status": "subscribed"})
	}
}
