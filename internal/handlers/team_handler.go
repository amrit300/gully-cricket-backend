package handlers

import (
	"database/sql"

	"gully-cricket/internal/services"

	"github.com/gofiber/fiber/v2"
)

type TeamRequest struct {
	UserID      int    `json:"user_id"`
	MatchID     int    `json:"match_id"`
	TeamName    string `json:"team_name"`
	Captain     int    `json:"captain"`
	ViceCaptain int    `json:"vice_captain"`
	Players     []int  `json:"players"`
}

func CreateTeam(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		var req TeamRequest

		// Parse
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid request",
			})
		}

		// 🔐 TRUST JWT OVER BODY
		userID := c.Locals("user_id").(int)

		// Default name
		if req.TeamName == "" {
			req.TeamName = "My Team"
		}

		// Call service
		teamID, err := services.CreateTeam(
			db,
			userID,
			req.MatchID,
			req.TeamName,
			req.Players,
			req.Captain,
			req.ViceCaptain,
		)

		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"team_id": teamID,
		})
	}
}
