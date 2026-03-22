package middleware

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func JWTProtected() fiber.Handler {
	return func(c *fiber.Ctx) error {

		authHeader := c.Get("Authorization")

		if authHeader == "" {
			return c.Status(401).JSON(fiber.Map{"error": "missing token"})
		}

		tokenString := strings.Split(authHeader, "Bearer ")
		if len(tokenString) != 2 {
			return c.Status(401).JSON(fiber.Map{"error": "invalid token format"})
		}

		secret := os.Getenv("JWT_SECRET")

		token, err := jwt.Parse(tokenString[1], func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(401).JSON(fiber.Map{"error": "invalid token"})
		}

		return c.Next()
	}
}
