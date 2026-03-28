package handlers

import (
	"database/sql"

	"gully-cricket/internal/services"

	"github.com/gofiber/fiber/v2"
)

func GetMatches(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		data, err := services.GetMatches(db)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "failed to fetch matches",
			})
		}

		return c.JSON(data)
	}
}
