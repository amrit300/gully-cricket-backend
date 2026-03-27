package handlers

import (
	"database/sql"
	"strconv"

	"gully-cricket/internal/services"

	"github.com/gofiber/fiber/v2"
)

//////////////////////////////////////////////////////////////
// JOIN CONTEST
//////////////////////////////////////////////////////////////

type JoinRequest struct {
	ContestID int `json:"contest_id"`
	TeamID    int `json:"team_id"`
}

func JoinContest(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		var req JoinRequest

		// Parse request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid request",
			})
		}

		// Validate input
		if req.ContestID <= 0 || req.TeamID <= 0 {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid contest_id or team_id",
			})
		}

		// Safe user extraction
		userID, ok := c.Locals("user_id").(int)
		if !ok || userID <= 0 {
			return c.Status(401).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}

		// Call service
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

//////////////////////////////////////////////////////////////
// GET CONTESTS
//////////////////////////////////////////////////////////////

func GetContests(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		// ✅ SAFE PARAM PARSE
		matchIDStr := c.Params("match_id")

		matchID, err := strconv.Atoi(matchIDStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid match id",
			})
		}

		rows, err := db.Query(`
			SELECT id, contest_name, prize_pool, total_spots, filled_spots, status
			FROM contests
			WHERE match_id = $1
		`, matchID)

		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "internal server error",
			})
		}
		defer rows.Close()

		var contests []fiber.Map

		for rows.Next() {
			var id int
			var name string
			var prize float64
			var total int
			var filled int
			var status string

			if err := rows.Scan(&id, &name, &prize, &total, &filled, &status); err != nil {
				return c.Status(500).JSON(fiber.Map{
					"error": "failed to read contests",
				})
			}

			contests = append(contests, fiber.Map{
				"id":           id,
				"contest_name": name,
				"prize_pool":   prize,
				"total_spots":  total,
				"filled_spots": filled,
				"status":       status,
			})
		}

		// ✅ rows error check
		if err := rows.Err(); err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "failed to process contests",
			})
		}

		if contests == nil {
			contests = []fiber.Map{}
		}

		return c.JSON(contests)
	}
}
