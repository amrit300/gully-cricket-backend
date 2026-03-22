package handlers

import (
	"database/sql"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func GetLeaderboard(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		contestID, err := strconv.Atoi(c.Params("contest_id"))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid contest_id",
			})
		}

		rows, err := db.Query(`
			SELECT l.team_id, l.points, l.rank, l.winnings
			FROM leaderboard l
			WHERE l.contest_id = $1
			ORDER BY l.rank ASC
		`, contestID)

		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		defer rows.Close()

		var result []fiber.Map

		for rows.Next() {
			var teamID int
			var points float64
			var rank int
			var winnings float64

			if err := rows.Scan(&teamID, &points, &rank, &winnings); err != nil {
				continue
			}

			result = append(result, fiber.Map{
				"team_id":  teamID,
				"points":   points,
				"rank":     rank,
				"winnings": winnings,
			})
		}

		return c.JSON(result)
	}
}
