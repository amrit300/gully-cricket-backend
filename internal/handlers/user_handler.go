package handlers

import (
\t"crypto/hmac"
\t"crypto/sha256"
\t"database/sql"
\t"encoding/hex"
\t"encoding/json"
\t"log"
\t"net/url"
\t"os"
\t"sort"
\t"strings"
\t"time"

\t"github.com/golang-jwt/jwt/v4"
\t"github.com/gofiber/fiber/v2"
)

type RegisterRequest struct {
\tUsername string `json:"username"`
\tInitData string `json:"initData"`
}

func CreateUser(db *sql.DB) fiber.Handler {
\treturn func(c *fiber.Ctx) error {

\t\tvar req RegisterRequest
\t\tif err := c.BodyParser(&req); err != nil {
\t\t\treturn c.Status(400).JSON(fiber.Map{"error": "invalid request"})
\t\t}

\t\tif len(req.Username) < 3 {
\t\t\treturn c.Status(400).JSON(fiber.Map{"error": "username too short"})
\t\t}

\t\tbotToken := os.Getenv("TELEGRAM_BOT_TOKEN")
\t\tif botToken == "" {
\t\t\tlog.Println("BOT TOKEN MISSING")
\t\t\treturn c.Status(500).JSON(fiber.Map{"error": "server config error"})
\t\t}

\t\tif !verifyTelegram(req.InitData, botToken) {
\t\t\treturn c.Status(403).JSON(fiber.Map{"error": "telegram verification failed"})
\t\t}

\t\tvalues, err := url.ParseQuery(req.InitData)
\t\tif err != nil {
\t\t\treturn c.Status(400).JSON(fiber.Map{"error": "invalid init data"})
\t\t}

\t\tuserJSON := values.Get("user")
\t\tif userJSON == "" {
\t\t\treturn c.Status(400).JSON(fiber.Map{"error": "user missing"})
\t\t}

\t\tvar telegramUser struct {
\t\t\tID int64 `json:"id"`
\t\t}
\t\tif err := json.Unmarshal([]byte(userJSON), &telegramUser); err != nil {
\t\t\treturn c.Status(400).JSON(fiber.Map{"error": "invalid telegram user"})
\t\t}

\t\t// ✅ FIX: telegram_id not telegram
\t\tvar id int
\t\terr = db.QueryRow(`
\t\t\tINSERT INTO users (username, telegram_id)
\t\t\tVALUES ($1, $2)
\t\t\tON CONFLICT (telegram_id) DO UPDATE
\t\t\tSET username = EXCLUDED.username
\t\t\tRETURNING id
\t\t`, req.Username, telegramUser.ID).Scan(&id)

\t\tif err != nil {
\t\t\tlog.Println("USER INSERT ERROR:", err)
\t\t\treturn c.Status(500).JSON(fiber.Map{"error": err.Error()})
\t\t}

\t\t// ✅ FIX: return JWT token
\t\tjwtSecret := os.Getenv("JWT_SECRET")
\t\tif jwtSecret == "" {
\t\t\tjwtSecret = "change-this-in-production"
\t\t}

\t\ttoken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
\t\t\t"user_id": id,
\t\t\t"exp":     time.Now().Add(30 * 24 * time.Hour).Unix(),
\t\t})

\t\ttokenString, err := token.SignedString([]byte(jwtSecret))
\t\tif err != nil {
\t\t\treturn c.Status(500).JSON(fiber.Map{"error": "token generation failed"})
\t\t}

\t\treturn c.JSON(fiber.Map{
\t\t\t"user_id": id,
\t\t\t"token":   tokenString,
\t\t})
\t}
}

func verifyTelegram(initData string, botToken string) bool {
\tvalues, err := url.ParseQuery(initData)
\tif err != nil {
\t\treturn false
\t}

\thash := values.Get("hash")
\tvalues.Del("hash")

\tvar dataCheckArr []string
\tfor key, val := range values {
\t\tdataCheckArr = append(dataCheckArr, key+"="+val[0])
\t}
\tsort.Strings(dataCheckArr)
\tdataCheckString := strings.Join(dataCheckArr, "
")

\tsecretKey := sha256.Sum256([]byte(botToken))
\th := hmac.New(sha256.New, secretKey[:])
\th.Write([]byte(dataCheckString))
\tcalculatedHash := hex.EncodeToString(h.Sum(nil))

\treturn calculatedHash == hash
}
