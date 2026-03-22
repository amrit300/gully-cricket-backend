package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

func RateLimit() fiber.Handler {
	return limiter.New(limiter.Config{

		// 🔥 Better limits
		Max:        20,
		Expiration: 1 * time.Second,

		// 🔐 Use user_id if available, else fallback to IP
		KeyGenerator: func(c *fiber.Ctx) string {

			if userID := c.Locals("user_id"); userID != nil {
				return "user_" + fiber.Utils().ToString(userID)
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
