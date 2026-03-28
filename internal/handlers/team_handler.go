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

		// Parse request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid request",
			})
		}

		// 🔐 SAFE JWT extraction (no panic)
		userIDVal := c.Locals("user_id")
		userID, ok := userIDVal.(int)
		if !ok || userID <= 0 {
			return c.Status(401).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}

		// Basic validation
		if req.MatchID <= 0 {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid match_id",
			})
		}

		if len(req.Players) != 11 {
			return c.Status(400).JSON(fiber.Map{
				"error": "team must have 11 players",
			})
		}

		// Default name
		if req.TeamName == "" {
			req.TeamName = "My Team"
		}

		// Call service (already context-safe inside)
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
