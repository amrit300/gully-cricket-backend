package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/time/rate"
)

var limiter = rate.NewLimiter(1, 5) // 1 req/sec, burst 5

func RateLimit() fiber.Handler {
	return func(c *fiber.Ctx) error {

		if !limiter.Allow() {
			return c.Status(429).JSON(fiber.Map{
				"error": "too many requests",
			})
		}

		return c.Next()
	}
}
