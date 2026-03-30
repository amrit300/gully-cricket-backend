package handlers

import (
	"database/sql"

	"gully-cricket/internal/services"
	"github.com/gofiber/fiber/v2"
)

func Subscribe(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		var req struct {
			PlanID int `json:"plan_id"`
		}

		// Parse request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid request",
			})
		}

		// Validate
		if req.PlanID <= 0 {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid plan_id",
			})
		}

		// Get user
		userID, ok := c.Locals("user_id").(int)
		if !ok || userID <= 0 {
			return c.Status(401).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}

		// 🔥 FIX: use req.PlanID
		err := services.SubscribeUser(db, userID, req.PlanID)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"status": "subscribed",
		})
	}
}
