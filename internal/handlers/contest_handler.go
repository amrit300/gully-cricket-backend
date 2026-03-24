package handlers

import (
	"database/sql"

	"gully-cricket/internal/services"

	"github.com/gofiber/fiber/v2"
)

type JoinRequest struct {
	ContestID int `json:"contest_id"`
	TeamID    int `json:"team_id"`
}

func JoinContest(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		var req JoinRequest

		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid request",
			})
		}

		userID := c.Locals("user_id").(int)

		err := services.JoinContest(
			db,
			userID,
			req.TeamID,
			req.ContestID,
		)

		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"status": "joined",
		})
	}
}
