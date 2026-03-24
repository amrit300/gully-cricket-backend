package middleware

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/gofiber/fiber/v2"
)

func JWTProtected() fiber.Handler {
	return func(c *fiber.Ctx) error {

		//////////////////////////////////////////////////////////////
		// 🔐 AUTH HEADER VALIDATION
		//////////////////////////////////////////////////////////////

		authHeader := c.Get("Authorization")

		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(401).JSON(fiber.Map{
				"error": "missing or invalid authorization header",
			})
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		//////////////////////////////////////////////////////////////
		// 🔐 SECRET (STRICT — NO FALLBACK)
		//////////////////////////////////////////////////////////////

		jwtSecret := os.Getenv("JWT_SECRET")
		if jwtSecret == "" {
			return c.Status(500).JSON(fiber.Map{
				"error": "server misconfigured",
			})
		}

		//////////////////////////////////////////////////////////////
		// 🔐 PARSE TOKEN WITH STRICT METHOD CHECK
		//////////////////////////////////////////////////////////////

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {

			// Enforce HMAC signing only
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("invalid signing method")
			}

			return []byte(jwtSecret), nil
		})

		if err != nil || token == nil || !token.Valid {
			return c.Status(401).JSON(fiber.Map{
				"error": "invalid or expired token",
			})
		}

		//////////////////////////////////////////////////////////////
		// 🔐 CLAIM EXTRACTION (SAFE)
		//////////////////////////////////////////////////////////////

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(401).JSON(fiber.Map{
				"error": "invalid token claims",
			})
		}

		// ✅ user_id extraction
		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			return c.Status(401).JSON(fiber.Map{
				"error": "invalid user_id",
			})
		}

		userID := int(userIDFloat)

		//////////////////////////////////////////////////////////////
		// 🔐 EXPIRY VALIDATION (IMPORTANT)
		//////////////////////////////////////////////////////////////

		if exp, ok := claims["exp"].(float64); ok {
			if time.Now().Unix() > int64(exp) {
				return c.Status(401).JSON(fiber.Map{
					"error": "token expired",
				})
			}
		} else {
			return c.Status(401).JSON(fiber.Map{
				"error": "missing expiry in token",
			})
		}

		//////////////////////////////////////////////////////////////
		// 🔐 SET CONTEXT
		//////////////////////////////////////////////////////////////

		c.Locals("user_id", userID)

		return c.Next()
	}
}
