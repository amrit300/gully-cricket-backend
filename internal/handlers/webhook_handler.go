func NowPaymentsWebhook(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		//////////////////////////////////////////////////////////////
		// 🔐 1. GET RAW BODY (MANDATORY)
		//////////////////////////////////////////////////////////////

		body := c.Body()

		//////////////////////////////////////////////////////////////
		// 🔐 2. GET SIGNATURE
		//////////////////////////////////////////////////////////////

		signature := c.Get("x-nowpayments-sig")
		if signature == "" {
			return c.SendStatus(403)
		}

		//////////////////////////////////////////////////////////////
		// 🔐 3. VERIFY SIGNATURE
		//////////////////////////////////////////////////////////////

		secret := os.Getenv("NOWPAYMENTS_IPN_SECRET")

		mac := hmac.New(sha512.New, []byte(secret))
		mac.Write(body)
		expectedHash := hex.EncodeToString(mac.Sum(nil))

		if expectedHash != signature {
			return c.SendStatus(403)
		}

		//////////////////////////////////////////////////////////////
		// ✅ 4. NOW SAFE TO PARSE
		//////////////////////////////////////////////////////////////

		var payload struct {
			PaymentID     string  `json:"payment_id"`
			PayAmount     float64 `json:"pay_amount"`
			PaymentStatus string  `json:"payment_status"`
		}

		if err := c.BodyParser(&payload); err != nil {
			return c.SendStatus(400)
		}

		//////////////////////////////////////////////////////////////
		// ✅ 5. PROCESS ONLY SUCCESS
		//////////////////////////////////////////////////////////////

		if payload.PaymentStatus != "finished" {
			return c.SendStatus(200)
		}

		//////////////////////////////////////////////////////////////
		// ✅ 6. DB LOGIC
		//////////////////////////////////////////////////////////////

		tx, _ := db.Begin()
		defer tx.Rollback()

		var exists bool
		tx.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM wallet_transactions WHERE source=$1
			)
		`, payload.PaymentID).Scan(&exists)

		if exists {
			return c.SendStatus(200)
		}

		userID := getUserFromPayment(payload.PaymentID)

		err := services.AddFunds(tx, userID, payload.PayAmount, payload.PaymentID)
		if err != nil {
			return c.SendStatus(500)
		}

		tx.Commit()

		return c.SendStatus(200)
	}
}
