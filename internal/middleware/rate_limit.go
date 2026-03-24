package middleware

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

func RateLimit() fiber.Handler {
	return limiter.New(limiter.Config{

		//////////////////////////////////////////////////////////////
		// 🔥 LIMIT CONFIG (BALANCED FOR PROD)
		//////////////////////////////////////////////////////////////

		Max:        60,              // max requests
		Expiration: 1 * time.Minute, // per minute window

		//////////////////////////////////////////////////////////////
		// 🔐 KEY GENERATOR (USER > IP)
		//////////////////////////////////////////////////////////////

		KeyGenerator: func(c *fiber.Ctx) string {

			// Prefer authenticated user
			if userID := c.Locals("user_id"); userID != nil {
				return fmt.Sprintf("user_%v", userID)
			}

			// Fallback → IP (unauthenticated)
			return fmt.Sprintf("ip_%s", c.IP())
		},

		//////////////////////////////////////////////////////////////
		// 🚫 LIMIT REACHED RESPONSE
		//////////////////////////////////////////////////////////////

		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(429).JSON(fiber.Map{
				"error": "rate limit exceeded, slow down",
			})
		},

		//////////////////////////////////////////////////////////////
		// ⚡ SKIP HEALTH CHECKS (IMPORTANT)
		//////////////////////////////////////////////////////////////

		Next: func(c *fiber.Ctx) bool {
			return c.Path() == "/health"
		},
	})
}
