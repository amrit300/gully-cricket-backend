package handlers

import (
	"database/sql"
	"time"

	"github.com/gofiber/fiber/v2"
)

func GetMatches(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		rows, err := db.Query(`
			SELECT id, team_a, team_b, venue, start_time, status
			FROM matches_master
			ORDER BY start_time ASC
			LIMIT 50
		`)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		defer rows.Close()

		var matches []fiber.Map

		for rows.Next() {

			var id int
			var teamA, teamB, venue, status string
			var start time.Time

			err := rows.Scan(
				&id,
				&teamA,
				&teamB,
				&venue,
				&start,
				&status,
			)
			if err != nil {
				continue
			}

			matches = append(matches, fiber.Map{
				"id":        id,
				"teamA":     teamA,
				"teamB":     teamB,
				"venue":     venue,
				"startTime": start.Format(time.RFC3339),
				"status":    status,
			})
		}

		return c.JSON(matches)
	}
}
