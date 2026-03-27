func NowPaymentsWebhook(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		var payload struct {
			PaymentID string  `json:"payment_id"`
			PayAmount float64 `json:"pay_amount"`
			PayCurrency string `json:"pay_currency"`
			PaymentStatus string `json:"payment_status"`
		}

		if err := c.BodyParser(&payload); err != nil {
			return c.SendStatus(400)
		}

		// ✅ only accept confirmed payments
		if payload.PaymentStatus != "finished" {
			return c.SendStatus(200)
		}

		tx, _ := db.Begin()
		defer tx.Rollback()

		// prevent duplicate
		var exists bool
		tx.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM wallet_transactions WHERE source=$1
			)
		`, payload.PaymentID).Scan(&exists)

		if exists {
			return c.SendStatus(200)
		}

		// TODO: map payment_id → user_id (store when creating payment)

		userID := getUserFromPayment(payload.PaymentID)

		err := AddFunds(tx, userID, payload.PayAmount, payload.PaymentID)
		if err != nil {
			return c.SendStatus(500)
		}

		tx.Commit()

		return c.SendStatus(200)
	}
}
