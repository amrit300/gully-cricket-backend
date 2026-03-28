package services

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"strconv"
	"gully-cricket/internal/cache"
)

//////////////////////////////////////////////////////////////
// MAIN FUNCTION
//////////////////////////////////////////////////////////////

func JoinContest(db *sql.DB, userID, teamID, contestID int) error {

	if userID <= 0 || teamID <= 0 || contestID <= 0 {
		return errors.New("invalid input")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	//////////////////////////////////////////////////////////////
	// TEAM VALIDATION
	//////////////////////////////////////////////////////////////

	var ownerID int
	var teamMatchID int

	err = tx.QueryRowContext(ctx, `
		SELECT user_id, match_id
		FROM teams
		WHERE id=$1
	`, teamID).Scan(&ownerID, &teamMatchID)

	if err != nil {
		return err
	}

	if ownerID != userID {
		return errors.New("unauthorized team")
	}

	//////////////////////////////////////////////////////////////
	// LOCK USER
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
	// LOCK CONTEST
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

	if teamMatchID != matchID {
		return errors.New("team does not belong to this match")
	}

	if status != "upcoming" {
		return errors.New("contest locked")
	}

	if filled >= total {
		return errors.New("contest full")
	}

	//////////////////////////////////////////////////////////////
	// COUNT USER TEAMS
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
	// DUPLICATE CHECK
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
	// INSERT ENTRY
	//////////////////////////////////////////////////////////////

	_, err = tx.ExecContext(ctx, `
		INSERT INTO contest_entries (contest_id, user_id, team_id)
		VALUES ($1,$2,$3)
	`, contestID, userID, teamID)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			return errors.New("already joined with this team")
		}
		return err
	}

	//////////////////////////////////////////////////////////////
	// UPDATE SPOTS
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
	// LEADERBOARD
	//////////////////////////////////////////////////////////////

	_, err = tx.ExecContext(ctx, `
		INSERT INTO leaderboard (contest_id, team_id, user_id, points, rank)
		VALUES ($1,$2,$3,0,0)
	`, contestID, teamID, userID)

	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
	return err
}

// 🔥 CACHE INVALIDATION (SYNC — REQUIRED)
cache.Rdb.Del(cache.Ctx, "contests:"+strconv.Itoa(matchID))

// 🔥 OPTIONAL ASYNC (LEADERBOARD)
go func() {
	cache.Rdb.Del(cache.Ctx, "leaderboard:"+strconv.Itoa(contestID))
}()

return nil

//////////////////////////////////////////////////////////////
// RETRY WRAPPER
//////////////////////////////////////////////////////////////

func JoinContestWithRetry(db *sql.DB, userID, teamID, contestID int) error {

	for i := 0; i < 3; i++ {

		err := JoinContest(db, userID, teamID, contestID)

		if err == nil {
			return nil
		}

		if !isRetryableError(err) {
			return err
		}

		time.Sleep(50 * time.Millisecond)
	}

	return errors.New("transaction failed after retries")
}

//////////////////////////////////////////////////////////////
// RETRY DETECTOR
//////////////////////////////////////////////////////////////

func isRetryableError(err error) bool {
	return strings.Contains(err.Error(), "restart transaction") ||
		strings.Contains(err.Error(), "deadlock") ||
		strings.Contains(err.Error(), "serialization")
}
