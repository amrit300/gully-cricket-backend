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
func GetContests(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		matchID := c.Params("match_id")

		rows, err := db.Query(`
			SELECT id, contest_name, prize_pool, total_spots, filled_spots, status
			FROM contests
			WHERE match_id = $1
		`, matchID)

		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
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

			rows.Scan(&id, &name, &prize, &total, &filled, &status)

			contests = append(contests, fiber.Map{
				"id": id,
				"contest_name": name,
				"prize_pool": prize,
				"total_spots": total,
				"filled_spots": filled,
				"status": status,
			})
		}

		return c.JSON(contests)
	}
}
