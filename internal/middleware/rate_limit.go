package middleware

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

func RateLimit() fiber.Handler {
	return limiter.New(limiter.Config{

		Max:        20,
		Expiration: 1 * time.Second,

		KeyGenerator: func(c *fiber.Ctx) string {
			if userID := c.Locals("user_id"); userID != nil {
				return fmt.Sprintf("user_%v", userID)
			}
			return c.IP()
		},

		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(429).JSON(fiber.Map{
				"error": "too many requests",
			})
		},
	})
}
