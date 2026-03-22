package services

import (
	"database/sql"
	"fmt"

	"gully-cricket/internal/validators"
)

func CreateTeam(db *sql.DB, userID int, matchID int, playerIDs []int, captainID int, viceCaptainID int) error {

	// 1. VALIDATIONS
	if err := validators.ValidateMatchStatus(db, matchID); err != nil {
		return err
	}

	if err := validators.ValidateTeamLimit(db, userID, matchID); err != nil {
		return err
	}

	if err := validators.ValidateTeam(db, playerIDs, captainID, viceCaptainID); err != nil {
		return err
	}

	// 2. TRANSACTION
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("db error")
	}
	defer tx.Rollback()

	var teamID int

	// 3. INSERT TEAM
	err = tx.QueryRow(`
		INSERT INTO teams (user_id, match_id, captain_player_id, vice_captain_player_id)
		VALUES ($1,$2,$3,$4)
		RETURNING id
	`, userID, matchID, captainID, viceCaptainID).Scan(&teamID)

	if err != nil {
		return err
	}

	// 4. INSERT TEAM PLAYERS
	for _, pid := range playerIDs {
		_, err = tx.Exec(`
			INSERT INTO team_players (team_id, player_id)
			VALUES ($1,$2)
		`, teamID, pid)

		if err != nil {
			return err
		}
	}

	// 5. COMMIT
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit failed")
	}

	return nil
}
