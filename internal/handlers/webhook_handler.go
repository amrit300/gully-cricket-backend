package handlers

import (
	"crypto/hmac"
	"crypto/sha512"
	"database/sql"
	"encoding/hex"
	"log"
	"os"

	"gully-cricket/internal/services"

	"github.com/gofiber/fiber/v2"
)

func NowPaymentsWebhook(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		//////////////////////////////////////////////////////////////
		// 🔐 1. RAW BODY (MANDATORY)
		//////////////////////////////////////////////////////////////

		body := c.Body()

		//////////////////////////////////////////////////////////////
		// 🔐 2. SIGNATURE CHECK
		//////////////////////////////////////////////////////////////

		signature := c.Get("x-nowpayments-sig")
		if signature == "" {
			log.Println("❌ Missing signature")
			return c.SendStatus(403)
		}

		//////////////////////////////////////////////////////////////
		// 🔐 3. VERIFY SIGNATURE (HMAC SHA512)
		//////////////////////////////////////////////////////////////

		secret := os.Getenv("NOWPAYMENTS_IPN_SECRET")
		if secret == "" {
			log.Println("❌ IPN secret not configured")
			return c.SendStatus(500)
		}

		mac := hmac.New(sha512.New, []byte(secret))
		mac.Write(body)
		expectedHash := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(expectedHash), []byte(signature)) {
			log.Println("❌ Invalid webhook signature")
			return c.SendStatus(403)
		}

		//////////////////////////////////////////////////////////////
		// ✅ 4. PARSE PAYLOAD (SAFE NOW)
		//////////////////////////////////////////////////////////////

		var payload struct {
			PaymentID     string  `json:"payment_id"`
			PayAmount     float64 `json:"pay_amount"`
			PayCurrency   string  `json:"pay_currency"`
			PaymentStatus string  `json:"payment_status"`
		}

		if err := c.BodyParser(&payload); err != nil {
			log.Println("❌ Body parse failed:", err)
			return c.SendStatus(400)
		}

		//////////////////////////////////////////////////////////////
		// 🔐 5. STRICT VALIDATION
		//////////////////////////////////////////////////////////////

		if payload.PaymentID == "" {
			return c.SendStatus(400)
		}

		if payload.PayAmount <= 0 {
			return c.SendStatus(400)
		}

		// Accept only USDT TRC20 (adjust if needed)
		if payload.PayCurrency != "usdttrc20" {
			log.Println("❌ Invalid currency:", payload.PayCurrency)
			return c.SendStatus(400)
		}

		//////////////////////////////////////////////////////////////
		// ✅ 6. ONLY PROCESS SUCCESS PAYMENTS
		//////////////////////////////////////////////////////////////

		if payload.PaymentStatus != "finished" {
			return c.SendStatus(200)
		}

		//////////////////////////////////////////////////////////////
		// 🔐 7. DB TRANSACTION (ATOMIC)
		//////////////////////////////////////////////////////////////

		tx, err := db.Begin()
		if err != nil {
			log.Println("❌ DB begin error:", err)
			return c.SendStatus(500)
		}
		defer tx.Rollback()

		//////////////////////////////////////////////////////////////
		// 🔐 8. DUPLICATE PROTECTION (IDEMPOTENCY)
		//////////////////////////////////////////////////////////////

		var exists bool
		err = tx.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM wallet_transactions WHERE source=$1
			)
		`, payload.PaymentID).Scan(&exists)

		if err != nil {
			log.Println("❌ Duplicate check failed:", err)
			return c.SendStatus(500)
		}

		if exists {
			log.Println("⚠️ Duplicate webhook:", payload.PaymentID)
			return c.SendStatus(200)
		}

		//////////////////////////////////////////////////////////////
		// 🔐 9. GET USER FROM PAYMENT TABLE (MANDATORY)
		//////////////////////////////////////////////////////////////

		var userID int
		err = tx.QueryRow(`
			SELECT user_id FROM payments WHERE payment_id=$1
		`, payload.PaymentID).Scan(&userID)

		if err != nil {
			log.Println("❌ Payment mapping not found:", payload.PaymentID)
			return c.SendStatus(400)
		}

		//////////////////////////////////////////////////////////////
		// 💰 10. CREDIT WALLET
		//////////////////////////////////////////////////////////////

		err = services.AddFunds(tx, userID, payload.PayAmount, payload.PaymentID)
		if err != nil {
			log.Println("❌ Wallet credit failed:", err)
			return c.SendStatus(500)
		}

		//////////////////////////////////////////////////////////////
		// ✅ 11. MARK PAYMENT SUCCESS
		//////////////////////////////////////////////////////////////

		_, err = tx.Exec(`
			UPDATE payments
			SET status='completed'
			WHERE payment_id=$1
		`, payload.PaymentID)

		if err != nil {
			log.Println("❌ Payment update failed:", err)
			return c.SendStatus(500)
		}

		//////////////////////////////////////////////////////////////
		// ✅ 12. COMMIT
		//////////////////////////////////////////////////////////////

		if err := tx.Commit(); err != nil {
			log.Println("❌ Commit failed:", err)
			return c.SendStatus(500)
		}

		log.Println("✅ Payment credited:", payload.PaymentID)

		return c.SendStatus(200)
	}
}
