package handlers

import (
	"database/sql"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func GetLeaderboard(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		// ✅ SAFE PARAM PARSE
		contestIDStr := c.Params("contestId")
		contestID, err := strconv.Atoi(contestIDStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid contest id",
			})
		}

		ctx, cancel := dbutil.Ctx()
		defer cancel()
		rows, err := db.QueryContext(ctx, `
		SELECT
		l.rank,
		l.points,
		l.winnings,
		u.username,
		t.team_name
	FROM leaderboard l
	JOIN teams t ON t.id = l.team_id
	JOIN users u ON u.id = t.user_id
	WHERE l.contest_id = $1
	ORDER BY l.rank ASC
	LIMIT 100
`, contestID)

		if err != nil {
			log.Println("LEADERBOARD QUERY ERROR:", err)
			return c.Status(500).JSON(fiber.Map{
				"error": "internal server error",
			})
		}
		defer rows.Close()

		type Entry struct {
			Rank     int     `json:"rank"`
			Points   float64 `json:"points"`
			Winnings float64 `json:"winnings"`
			Username string  `json:"username"`
			TeamName string  `json:"team_name"`
		}

		var entries []Entry

		for rows.Next() {
			var e Entry

			if err := rows.Scan(
				&e.Rank,
				&e.Points,
				&e.Winnings,
				&e.Username,
				&e.TeamName,
			); err != nil {
				log.Println("SCAN ERROR:", err)
				continue
			}

			entries = append(entries, e)
		}

		if entries == nil {
	entries = []Entry{}
}

if err := rows.Err(); err != nil {
	return c.Status(500).JSON(fiber.Map{
		"error": "failed to read leaderboard",
	})
}

		return c.JSON(entries)
	}
}
