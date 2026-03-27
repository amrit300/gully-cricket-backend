package services

import (
	"database/sql"

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

	// Defensive Check 
	if userID <= 0 || matchID <= 0 {
	return 0, errors.New("invalid input")
}

	// VALIDATIONS
	if err := validators.ValidateTeam(db, playerIDs, captainID, viceCaptainID); err != nil {
		return 0, err
	}

	if err := validators.ValidateMatchStatus(db, matchID); err != nil {
		return 0, err
	}

	if err := validators.ValidateTeamLimit(db, userID, matchID); err != nil {
		return 0, err
	}

	// TRANSACTION
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var teamID int

	err = tx.QueryRow(`
		INSERT INTO teams (user_id, match_id, team_name, captain_player_id, vice_captain_player_id)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id
	`, userID, matchID, teamName, captainID, viceCaptainID).Scan(&teamID)

	if err != nil {
		return 0, err
	}

	for _, pid := range playerIDs {
		_, err = tx.Exec(`
			INSERT INTO team_players (team_id, player_id)
			VALUES ($1,$2)
		`, teamID, pid)

		if err != nil {
			return 0, err
		}
	}

	return teamID, tx.Commit()
}
