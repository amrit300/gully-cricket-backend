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
type AddFundsRequest struct {
	Amount float64 `json:"amount"`
	TxHash string  `json:"tx_hash"`
}

func AddFundsHandler(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		userID := c.Locals("user_id").(int)

		var req AddFundsRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid request",
			})
		}

		if req.Amount <= 0 {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid amount",
			})
		}

		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		err = services.AddFunds(tx, userID, req.Amount, "crypto")
		if err != nil {
			return err
		}

		return tx.Commit()
	}
}
