package handlers

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"gully-cricket/internal/providers"
)

func SyncPlayers(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		matchID := c.Params("match_id")
		externalID := c.Params("external_id")

		if matchID == "" || externalID == "" {
			return c.Status(400).JSON(fiber.Map{
				"error": "match_id and external_id required",
			})
		}

		players, err := providers.FetchPlayersFromEntityAPI(externalID)
		if err != nil {
			log.Println("PLAYER FETCH ERROR:", err)
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		for _, p := range players {

			name := fmt.Sprintf("%v", p["name"])
			role := fmt.Sprintf("%v", p["role"])
			team := fmt.Sprintf("%v", p["team"])

			_, err := db.Exec(`
				INSERT INTO players (name, team, role, credit, match_id)
				VALUES ($1,$2,$3,8.5,$4)
				ON CONFLICT DO NOTHING
			`, name, team, role, matchID)

			if err != nil {
				log.Println("PLAYER INSERT ERROR:", err)
			}
		}

		return c.JSON(fiber.Map{
			"status": "players synced",
		})
	}
}
func GetPlayers(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		matchID := c.Params("match_id")

		rows, err := db.Query(`
			SELECT id, name, team, role, credit, fantasy_points
			FROM players
			WHERE match_id = $1
		`, matchID)

		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		defer rows.Close()

		var players []map[string]interface{}

		for rows.Next() {
			var id int
			var name, team, role string
			var credit, fantasyPoints float64

			err := rows.Scan(&id, &name, &team, &role, &credit, &fantasyPoints)
			if err != nil {
				continue
			}

			players = append(players, fiber.Map{
				"id": id,
				"name": name,
				"team": team,
				"role": role,
				"credit": credit,
				"fantasy_points": fantasyPoints,
			})
		}

		return c.JSON(players)
	}
}
