package services

import (
	"database/sql"

	"gully-cricket/internal/validators"
)

func CreateTeam(db *sql.DB, userID int, matchID int, players [] int, captainID int, viceCaptainID int) error {

	// Validate team
	if err := validators.ValidateTeam(players, captainID, viceCaptainID); err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	var teamID int

	err = tx.QueryRow(`
		INSERT INTO teams (user_id, match_id, captain_player_id, vice_captain_player_id)
		VALUES ($1,$2,$3,$4)
		RETURNING id
	`, userID, matchID, captainID, viceCaptainID).Scan(&teamID)

	if err != nil {
		tx.Rollback()
		return err
	}

	for _, p := range players {
		_, err := tx.Exec(`
			INSERT INTO team_players (team_id, player_id)
			VALUES ($1,$2)
		`, teamID, p.ID)

		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
