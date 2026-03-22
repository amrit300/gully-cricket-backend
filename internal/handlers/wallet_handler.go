package handlers

import (
	"database/sql"
	"strconv"

	"gully-cricket/internal/services"

	"github.com/gofiber/fiber/v2"
)

func GetBalance(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		userID := c.Locals("user_id").(int)

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
