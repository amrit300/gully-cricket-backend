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
