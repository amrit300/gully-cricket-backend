package services

import (
	"context"
	"database/sql"
	"errors"
	"time"

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

	//////////////////////////////////////////////////////////////
	// 🔐 DEFENSIVE CHECK
	//////////////////////////////////////////////////////////////

	if userID <= 0 || matchID <= 0 {
		return 0, errors.New("invalid input")
	}

	if len(playerIDs) != 11 {
		return 0, errors.New("team must have 11 players")
	}

	//////////////////////////////////////////////////////////////
	// 🔐 VALIDATIONS (FAST PRE-CHECKS)
	//////////////////////////////////////////////////////////////

	if err := validators.ValidateTeam(db, playerIDs, captainID, viceCaptainID); err != nil {
		return 0, err
	}

	if err := validators.ValidateMatchStatus(db, matchID); err != nil {
		return 0, err
	}

	if err := validators.ValidateTeamLimit(db, userID, matchID); err != nil {
		return 0, err
	}

	//////////////////////////////////////////////////////////////
	// 🚀 TRANSACTION START
	//////////////////////////////////////////////////////////////

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//////////////////////////////////////////////////////////////
	// 🔐 HARD LOCK (RACE CONDITION PROTECTION)
	//////////////////////////////////////////////////////////////

	// Lock user row to serialize team creation
	var maxTeams int
	err = tx.QueryRowContext(ctx, `
		SELECT max_teams_per_match
		FROM users
		WHERE id=$1
		FOR UPDATE
	`, userID).Scan(&maxTeams)

	if err != nil {
		return 0, err
	}

	// Count existing teams inside transaction (AFTER LOCK)
	var currentTeams int
	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM teams
		WHERE user_id=$1 AND match_id=$2
	`, userID, matchID).Scan(&currentTeams)

	if err != nil {
		return 0, err
	}

	if currentTeams >= maxTeams {
		return 0, errors.New("team limit reached")
	}

	//////////////////////////////////////////////////////////////
	// 🔐 INSERT TEAM (ATOMIC)
	//////////////////////////////////////////////////////////////

	var teamID int

	err = tx.QueryRowContext(ctx, `
		INSERT INTO teams (
			user_id,
			match_id,
			team_name,
			captain_player_id,
			vice_captain_player_id
		)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id
	`, userID, matchID, teamName, captainID, viceCaptainID).Scan(&teamID)

	if err != nil {
		return 0, err
	}

	//////////////////////////////////////////////////////////////
	// 🔐 INSERT TEAM PLAYERS
	//////////////////////////////////////////////////////////////

	for _, pid := range playerIDs {

		if pid <= 0 {
			return 0, errors.New("invalid player id")
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO team_players (team_id, player_id)
			VALUES ($1,$2)
		`, teamID, pid)

		if err != nil {
			return 0, err
		}
	}

	//////////////////////////////////////////////////////////////
	// ✅ COMMIT
	//////////////////////////////////////////////////////////////

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return teamID, nil
}
