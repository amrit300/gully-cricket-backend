package handlers

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"gully-cricket/internal/services"
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

		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid request",
			})
		}

		// Basic validation only (keep handler light)
		if req.UserID == 0 || req.MatchID == 0 {
			return c.Status(400).JSON(fiber.Map{
				"error": "user_id and match_id required",
			})
		}

		if req.TeamName == "" {
			req.TeamName = "My Team"
		}

		teamID, err := services.CreateTeam(
			db,
			req.UserID,
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
