package handlers

import (
	"database/sql"

	"gully-cricket/internal/services"

	"github.com/gofiber/fiber/v2"
)

func GetBalance(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		userID := c.Locals("user_id").(int)

		balance, err := services.GetBalance(db, userID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "failed to fetch balance",
			})
		}

		return c.JSON(fiber.Map{
			"balance": balance,
		})
	}
}
