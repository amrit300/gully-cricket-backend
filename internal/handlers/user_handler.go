package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type RegisterRequest struct {
	Username string `json:"username"`
	InitData string `json:"initData"`
}

func CreateUser(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		var req RegisterRequest

		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}

		botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
		if botToken == "" {
			log.Println("BOT TOKEN MISSING")
			return c.Status(500).JSON(fiber.Map{"error": "server config error"})
		}

		if !verifyTelegram(req.InitData, botToken) {
			return c.Status(403).JSON(fiber.Map{"error": "telegram verification failed"})
		}

		values, err := url.ParseQuery(req.InitData)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid init data"})
		}

		userJSON := values.Get("user")
		if userJSON == "" {
			return c.Status(400).JSON(fiber.Map{"error": "user missing"})
		}

		var telegramUser struct {
			ID int `json:"id"`
		}

		if err := json.Unmarshal([]byte(userJSON), &telegramUser); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid telegram user"})
		}

		var id int

		err = db.QueryRow(`
			INSERT INTO users (username, telegram)
			VALUES ($1,$2)
			ON CONFLICT (telegram)
			DO UPDATE SET username=EXCLUDED.username
			RETURNING id
		`, req.Username, telegramUser.ID).Scan(&id)

		if err != nil {
			log.Println("USER INSERT ERROR:", err)
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"user_id": id})
	}
}

func verifyTelegram(initData string, botToken string) bool {

	values, err := url.ParseQuery(initData)
	if err != nil {
		return false
	}

	hash := values.Get("hash")
	values.Del("hash")

	dataCheckArr := []string{}

	for key, val := range values {
		dataCheckArr = append(dataCheckArr, key+"="+val[0])
	}

	sort.Strings(dataCheckArr)

	dataCheckString := strings.Join(dataCheckArr, "\n")

	secretKey := sha256.Sum256([]byte(botToken))

	h := hmac.New(sha256.New, secretKey[:])
	h.Write([]byte(dataCheckString))

	calculatedHash := hex.EncodeToString(h.Sum(nil))

	return calculatedHash == hash
}
