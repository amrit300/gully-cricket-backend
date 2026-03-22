package handlers

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
)

func GetBalance(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		// ✅ Get user from JWT
		userIDVal := c.Locals("user_id")
		if userIDVal == nil {
			return c.Status(401).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}

		userID := userIDVal.(int)

		var balance float64

		err := db.QueryRow(`
			SELECT wallet_balance FROM users WHERE id=$1
		`, userID).Scan(&balance)

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
