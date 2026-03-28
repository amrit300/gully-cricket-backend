package handlers

import (
	"database/sql"

	dbutil "gully-cricket/internal/db"
	"gully-cricket/internal/services"

	"github.com/gofiber/fiber/v2"
)

//////////////////////////////////////////////////////////////
// 📊 GET BALANCE
//////////////////////////////////////////////////////////////

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

//////////////////////////////////////////////////////////////
// ➕ ADD FUNDS
//////////////////////////////////////////////////////////////

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

		// 🔐 VALIDATION
		if req.Amount <= 0 {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid amount",
			})
		}

		if len(req.TxHash) < 10 {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid transaction",
			})
		}

		// ⏱️ GLOBAL TIMEOUT CONTEXT
		ctx, cancel := dbutil.Ctx()
		defer cancel()

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "failed to start transaction",
			})
		}
		defer tx.Rollback()

		err = services.AddFunds(tx, userID, req.Amount, "crypto")
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "failed to commit transaction",
			})
		}

		return c.JSON(fiber.Map{
			"status": "funds added",
		})
	}
}
