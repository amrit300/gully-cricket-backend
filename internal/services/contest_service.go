package services

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

func JoinContest(db *sql.DB, userID, teamID, contestID int) error {

	//////////////////////////////////////////////////////////////
	// 0. DEFENSIVE VALIDATION
	//////////////////////////////////////////////////////////////

	if userID <= 0 || teamID <= 0 || contestID <= 0 {
		return errors.New("invalid input")
	}

	//////////////////////////////////////////////////////////////
	// CONTEXT (GLOBAL FOR THIS TX)
	//////////////////////////////////////////////////////////////

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	//////////////////////////////////////////////////////////////
	// 1. LOCK USER (PREVENT PLAN RACE)
	//////////////////////////////////////////////////////////////

	var maxTeams int

	err = tx.QueryRowContext(ctx, `
		SELECT max_teams_per_match
		FROM users
		WHERE id=$1
		FOR UPDATE
	`, userID).Scan(&maxTeams)

	if err != nil {
		return err
	}

	//////////////////////////////////////////////////////////////
	// 2. LOCK CONTEST + GET MATCH_ID
	//////////////////////////////////////////////////////////////

	var filled, total int
	var status string
	var matchID int

	err = tx.QueryRowContext(ctx, `
		SELECT filled_spots, total_spots, status, match_id
		FROM contests
		WHERE id=$1
		FOR UPDATE
	`, contestID).Scan(&filled, &total, &status, &matchID)

	if err != nil {
		return err
	}

	if status != "upcoming" {
		return errors.New("contest locked")
	}

	if filled >= total {
		return errors.New("contest full")
	}

	//////////////////////////////////////////////////////////////
	// 3. COUNT USER TEAMS FOR THIS MATCH
	//////////////////////////////////////////////////////////////

	var currentTeams int

	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM contest_entries ce
		JOIN teams t ON t.id = ce.team_id
		WHERE ce.user_id=$1 AND t.match_id=$2
	`, userID, matchID).Scan(&currentTeams)

	if err != nil {
		return err
	}

	if currentTeams >= maxTeams {
		return errors.New("team limit reached for your plan")
	}

	//////////////////////////////////////////////////////////////
	// 4. PREVENT DUPLICATE ENTRY
	//////////////////////////////////////////////////////////////

	var exists bool

	err = tx.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM contest_entries
			WHERE contest_id=$1 AND user_id=$2 AND team_id=$3
		)
	`, contestID, userID, teamID).Scan(&exists)

	if err != nil {
		return err
	}

	if exists {
		return errors.New("already joined with this team")
	}

	//////////////////////////////////////////////////////////////
	// 5. INSERT ENTRY (ATOMIC)
	//////////////////////////////////////////////////////////////

	_, err = tx.ExecContext(ctx, `
		INSERT INTO contest_entries (contest_id, user_id, team_id)
		VALUES ($1,$2,$3)
	`, contestID, userID, teamID)

	if err != nil {
		return err
	}

	//////////////////////////////////////////////////////////////
	// 6. SAFE SPOT UPDATE (NO OVERFLOW)
	//////////////////////////////////////////////////////////////

	result, err := tx.ExecContext(ctx, `
		UPDATE contests
		SET filled_spots = filled_spots + 1
		WHERE id=$1 AND filled_spots < total_spots
	`, contestID)

	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("contest just got full")
	}

	//////////////////////////////////////////////////////////////
	// 7. INSERT INTO LEADERBOARD
	//////////////////////////////////////////////////////////////

	_, err = tx.ExecContext(ctx, `
		INSERT INTO leaderboard (contest_id, team_id, user_id, points, rank)
		VALUES ($1,$2,$3,0,0)
	`, contestID, teamID, userID)

	if err != nil {
		return err
	}

	//////////////////////////////////////////////////////////////
	// COMMIT
	//////////////////////////////////////////////////////////////

	return tx.Commit()
}
