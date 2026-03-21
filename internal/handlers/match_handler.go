package handlers

import (
	"database/sql"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func GetMatches(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		rows, err := db.Query(`
			SELECT team_a, team_b, start_time, status, venue
			FROM matches_master
			ORDER BY start_time DESC
			LIMIT 50
		`)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()

		live := []fiber.Map{}
		upcoming := []fiber.Map{}
		recent := []fiber.Map{}

		for rows.Next() {
			var teamA, teamB, status, venue string
			var startTime string

			_ = rows.Scan(&teamA, &teamB, &startTime, &status, &venue)

			match := fiber.Map{
				"teamA":     teamA,
				"teamB":     teamB,
				"startTime": startTime,
				"status":    status,
				"venue":     venue,
			}

			if strings.Contains(status, "Live") || strings.Contains(status, "Stumps") {
				live = append(live, match)
			} else if strings.Contains(status, "Starts") || strings.Contains(status, "Upcoming") {
				upcoming = append(upcoming, match)
			} else {
				recent = append(recent, match)
			}
		}

		return c.JSON(fiber.Map{
			"live":     live,
			"upcoming": upcoming,
			"recent":   recent,
		})
	}
}
