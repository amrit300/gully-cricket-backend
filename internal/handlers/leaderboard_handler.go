package handlers

import (
	"database/sql"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func GetLeaderboard(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		contestID, _ := strconv.Atoi(c.Params("contest_id"))

		rows, _ := db.Query(`
			SELECT team_id, points, rank
			FROM leaderboard
			WHERE contest_id=$1
			ORDER BY rank ASC
		`, contestID)

		defer rows.Close()

		var result []fiber.Map

		for rows.Next() {
			var teamID int
			var points float64
			var rank int

			rows.Scan(&teamID, &points, &rank)

			result = append(result, fiber.Map{
				"team_id": teamID,
				"points":  points,
				"rank":    rank,
			})
		}

		return c.JSON(result)
	}
}
