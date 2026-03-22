package handlers

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"gully-cricket/internal/services"
)

type WithdrawRequest struct {
	UserID int     `json:"user_id"`
	Amount float64 `json:"amount"`
}

func RequestWithdrawal(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		var req WithdrawRequest

		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}

		if req.Amount <= 0 {
			return c.Status(400).JSON(fiber.Map{"error": "invalid amount"})
		}

		err := services.RequestWithdrawal(db, req.UserID, req.Amount)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{
			"status": "withdrawal requested",
		})
	}
}
