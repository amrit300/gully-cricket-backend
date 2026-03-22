package handlers

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"gully-cricket/internal/services"
)

type TeamRequest struct {
	MatchID     int   `json:"match_id"`
	Captain     int   `json:"captain"`
	ViceCaptain int   `json:"vice_captain"`
	Players     []int `json:"players"`
}

func CreateTeam(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		var req TeamRequest

		// Parse body
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid request",
			})
		}

		// ✅ Get user from JWT middleware
		userIDVal := c.Locals("user_id")
		if userIDVal == nil {
			return c.Status(401).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}

		userID := userIDVal.(int)

		// Basic validation
		if req.MatchID == 0 {
			return c.Status(400).JSON(fiber.Map{
				"error": "match_id required",
			})
		}

		if len(req.Players) == 0 {
			return c.Status(400).JSON(fiber.Map{
				"error": "players required",
			})
		}

		// ✅ Call service (FIXED SIGNATURE)
		err := services.CreateTeam(
			db,
			userID,
			req.MatchID,
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
			"status": "team created",
		})
	}
}
