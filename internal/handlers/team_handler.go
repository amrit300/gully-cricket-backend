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

		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid request",
			})
		}

		// BASIC VALIDATION
		if req.UserID == 0 || req.MatchID == 0 {
			return c.Status(400).JSON(fiber.Map{
				"error": "user_id and match_id required",
			})
		}

		if req.Captain == req.ViceCaptain {
			return c.Status(400).JSON(fiber.Map{
				"error": "captain and vice captain cannot be same",
			})
		}

		// BUSINESS VALIDATIONS
		if err := checkDailyTeamLimit(db, req.UserID); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		if err := validateTeam(db, req.Players); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		// 🔥 TRANSACTION (CRITICAL)
		tx, err := db.Begin()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "db error"})
		}
		defer tx.Rollback()

		var teamID int

		err = tx.QueryRow(`
			INSERT INTO teams
			(user_id,match_id,team_name,captain_player_id,vice_captain_player_id)
			VALUES ($1,$2,$3,$4,$5)
			RETURNING id
		`,
			req.UserID,
			req.MatchID,
			req.TeamName,
			req.Captain,
			req.ViceCaptain,
		).Scan(&teamID)

		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		for _, playerID := range req.Players {

			_, err = tx.Exec(`
				INSERT INTO team_players (team_id,player_id)
				VALUES ($1,$2)
			`, teamID, playerID)

			if err != nil {
				return c.Status(500).JSON(fiber.Map{"error": "player insert failed"})
			}
		}

		// ✅ COMMIT
		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "commit failed"})
		}

		return c.JSON(fiber.Map{
			"team_id": teamID,
		})
	}
}

//////////////////////////////////////////////////////////////
// 🔒 VALIDATIONS
//////////////////////////////////////////////////////////////

func checkDailyTeamLimit(db *sql.DB, userID int) error {

	var teamLimit int

	err := db.QueryRow(`
		SELECT max_teams_per_match
		FROM users
		WHERE id=$1
	`, userID).Scan(&teamLimit)

	if err != nil {
		return fmt.Errorf("failed to fetch user plan")
	}

	var teamsToday int

	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM teams
		WHERE user_id=$1
		AND created_at >= CURRENT_DATE
	`, userID).Scan(&teamsToday)

	if err != nil {
		return fmt.Errorf("failed to count teams")
	}

	if teamsToday >= teamLimit {
		return fmt.Errorf("daily team limit reached")
	}

	return nil
}

func validateTeam(db *sql.DB, playerIDs []int) error {

	if len(playerIDs) != 11 {
		return fmt.Errorf("team must contain 11 players")
	}

	rows, err := db.Query(`
		SELECT team,role,credit
		FROM players
		WHERE id = ANY($1)
	`, pq.Array(playerIDs))

	if err != nil {
		return err
	}
	defer rows.Close()

	teamCount := map[string]int{}
	roleCount := map[string]int{}

	totalCredit := 0.0
	playerCount := 0

	for rows.Next() {

		var team string
		var role string
		var credit float64

		err := rows.Scan(&team, &role, &credit)
		if err != nil {
			return err
		}

		playerCount++
		teamCount[team]++
		roleCount[role]++
		totalCredit += credit
	}

	if playerCount != 11 {
		return fmt.Errorf("invalid player selection")
	}

	if totalCredit > 100 {
		return fmt.Errorf("credit limit exceeded")
	}

	for _, count := range teamCount {
		if count > 7 {
			return fmt.Errorf("max 7 players allowed from one team")
		}
	}

	if roleCount["WK"] < 1 || roleCount["WK"] > 4 {
		return fmt.Errorf("invalid wicketkeeper count")
	}

	if roleCount["BAT"] < 3 || roleCount["BAT"] > 6 {
		return fmt.Errorf("invalid batsman count")
	}

	if roleCount["ALL"] < 1 || roleCount["ALL"] > 4 {
		return fmt.Errorf("invalid allrounder count")
	}

	if roleCount["BOWL"] < 3 || roleCount["BOWL"] > 6 {
		return fmt.Errorf("invalid bowler count")
	}

	return nil
}
