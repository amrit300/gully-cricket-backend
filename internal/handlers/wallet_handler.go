package handlers

import (
	"database/sql"
	"strconv"

	"gully-cricket/internal/services"

	"github.com/gofiber/fiber/v2"
)

func GetBalance(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		userID, _ := strconv.Atoi(c.Params("user_id"))

		balance, err := services.GetWalletBalance(db, userID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{
			"balance": balance,
		})
	}
}
