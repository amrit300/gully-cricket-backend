package handlers

import (
	"database/sql"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/lib/pq"
)

type TeamRequest struct {
	UserID      int    `json:"user_id"`
	MatchID     int    `json:"match_id"`
	TeamName    string `json:"team_name"`
	Captain     int    `json:"captain"`
	ViceCaptain int    `json:"vice_captain"`
	Players     []int  `json:"players"`
}

func CreateTeam(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		var req TeamRequest
		c.BodyParser(&req)

		if req.UserID == 0 || req.MatchID == 0 {
			return c.Status(400).JSON(fiber.Map{"error": "invalid"})
		}

		err := validateTeam(db, req.Players)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		var teamID int

		err = db.QueryRow(`
			INSERT INTO teams (user_id,match_id,team_name,captain_player_id,vice_captain_player_id)
			VALUES ($1,$2,$3,$4,$5)
			RETURNING id
		`, req.UserID, req.MatchID, req.TeamName, req.Captain, req.ViceCaptain).Scan(&teamID)

		if err != nil {
			return err
		}

		for _, p := range req.Players {
			db.Exec(`INSERT INTO team_players (team_id,player_id) VALUES ($1,$2)`, teamID, p)
		}

		return c.JSON(fiber.Map{"team_id": teamID})
	}
}

func validateTeam(db *sql.DB, ids []int) error {

	rows, err := db.Query(`
		SELECT team,role,credit
		FROM players
		WHERE id = ANY($1)
	`, pq.Array(ids))

	if err != nil {
		return err
	}
	defer rows.Close()

	if len(ids) != 11 {
		return fmt.Errorf("11 players required")
	}

	return nil
}
