package handlers

import (
	"database/sql"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type Player struct {
	ID     int     `json:"id"`
	Name   string  `json:"name"`
	Team   string  `json:"team"`
	Role   string  `json:"role"`
	Credit float64 `json:"credit"`
}

func GetPlayers(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		matchID, err := strconv.Atoi(c.Params("match_id"))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid match id"})
		}

		rows, err := db.Query(`
			SELECT id,name,team,role,credit
			FROM players
			WHERE match_id=$1
			ORDER BY role,credit DESC
		`, matchID)

		if err != nil {
			log.Println(err)
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()

		var players []Player

		for rows.Next() {
			var p Player
			rows.Scan(&p.ID, &p.Name, &p.Team, &p.Role, &p.Credit)
			players = append(players, p)
		}

		return c.JSON(players)
	}
}
