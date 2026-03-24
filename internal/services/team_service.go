package services

import (
	"database/sql"
	"fmt"

	"gully-cricket/internal/validators"
)

func CreateTeam(
	db *sql.DB,
	userID int,
	matchID int,
	teamName string,
	playerIDs []int,
	captainID int,
	viceCaptainID int,
) (int, error) {

	// validations
	if err := validators.ValidateTeam(db, playerIDs, captainID, viceCaptainID); err != nil {
		return 0, err
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}

	var teamID int

	err = tx.QueryRow(`
		INSERT INTO teams (user_id, match_id, team_name, captain_player_id, vice_captain_player_id)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id
	`, userID, matchID, teamName, captainID, viceCaptainID).Scan(&teamID)

	if err != nil {
		tx.Rollback()
		return 0, err
	}

	for _, pid := range playerIDs {
		_, err := tx.Exec(`
			INSERT INTO team_players (team_id, player_id)
			VALUES ($1,$2)
		`, teamID, pid)

		if err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return teamID, nil
}
