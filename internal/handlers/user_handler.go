package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/gofiber/fiber/v2"
)

type RegisterRequest struct {
	Username string `json:"username"`
	InitData string `json:"initData"`
}

func CreateUser(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if len(req.Username) > 50 {
	return c.Status(400).JSON(fiber.Map{"error": "username too long"})
}

		var req RegisterRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}

		if len(req.Username) < 3 {
			return c.Status(400).JSON(fiber.Map{"error": "username too short"})
		}

		botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
		if botToken == "" {
			log.Println("BOT TOKEN MISSING")
			return c.Status(500).JSON(fiber.Map{"error": "server config error"})
		}

		// 🔐 Verify Telegram data
		if !verifyTelegram(req.InitData, botToken) {
			return c.Status(403).JSON(fiber.Map{"error": "telegram verification failed"})
		}

		values, err := url.ParseQuery(req.InitData)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid init data"})
		}

		// ⏱️ Check auth_date freshness (max 1 day)
		authDateStr := values.Get("auth_date")
		authDate, _ := strconv.ParseInt(authDateStr, 10, 64)

		if time.Now().Unix()-authDate > 86400 {
			return c.Status(403).JSON(fiber.Map{"error": "init data expired"})
		}

		userJSON := values.Get("user")
		if userJSON == "" {
			return c.Status(400).JSON(fiber.Map{"error": "user missing"})
		}

		var telegramUser struct {
			ID int64 `json:"id"`
		}

		if err := json.Unmarshal([]byte(userJSON), &telegramUser); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid telegram user"})
		}

		var id int

		err = db.QueryRow(`
			INSERT INTO users (username, telegram_id)
			VALUES ($1, $2)
			ON CONFLICT (telegram_id) DO UPDATE
			SET username = EXCLUDED.username
			RETURNING id
		`, req.Username, telegramUser.ID).Scan(&id)

		if err != nil {
			log.Println("USER INSERT ERROR:", err)
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		// 🔐 JWT
		jwtSecret := os.Getenv("JWT_SECRET")
		if jwtSecret == "" {
			return c.Status(500).JSON(fiber.Map{"error": "jwt not configured"})
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id": id,
			"exp":     time.Now().Add(30 * 24 * time.Hour).Unix(),
		})

		tokenString, err := token.SignedString([]byte(jwtSecret))
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "token generation failed"})
		}

		return c.JSON(fiber.Map{
			"user_id": id,
			"token":   tokenString,
		})
	}
}

//////////////////////////////////////////////////////////////
// 🔐 TELEGRAM VERIFICATION (HARDENED)
//////////////////////////////////////////////////////////////

func verifyTelegram(initData string, botToken string) bool {

	values, err := url.ParseQuery(initData)
	if err != nil {
		return false
	}

	hash := values.Get("hash")
	values.Del("hash")

	var dataCheckArr []string

	for key, val := range values {
		dataCheckArr = append(dataCheckArr, key+"="+val[0])
	}

	sort.Strings(dataCheckArr)
	dataCheckString := strings.Join(dataCheckArr, "\n")

	secretKey := sha256.Sum256([]byte(botToken))

	h := hmac.New(sha256.New, secretKey[:])
	h.Write([]byte(dataCheckString))

	calculatedHash := hex.EncodeToString(h.Sum(nil))

	// 🔥 Constant-time compare (prevents timing attacks)
	return subtle.ConstantTimeCompare(
		[]byte(calculatedHash),
		[]byte(hash),
	) == 1
}
