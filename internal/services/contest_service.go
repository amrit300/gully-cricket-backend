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
// 🔥 SHADOW BAN CHECK
//////////////////////////////////////////////////////////////

var shadow bool

_ = tx.QueryRowContext(ctx, `
	SELECT shadow_banned FROM users WHERE id=$1
`, userID).Scan(&shadow)

if shadow {
	// silent success (do nothing)
	return nil
}

	//////////////////////////////////////////////////////////////
	// 🔥 FRAUD CHECK — RISK SCORE
	//////////////////////////////////////////////////////////////

	var riskScore float64

	_ = tx.QueryRowContext(ctx, `
		SELECT risk_score FROM user_risk_profiles WHERE user_id=$1
	`, userID).Scan(&riskScore)

	if riskScore > 80 {
		return errors.New("account restricted due to risk")
	}

	

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
// 🔥 SUBSCRIPTION CHECK (PLACE HERE)
//////////////////////////////////////////////////////////////

var status string

err = tx.QueryRowContext(ctx, `
	SELECT status FROM user_subscriptions WHERE user_id=$1
`, userID).Scan(&status)

if err != nil {
	return errors.New("subscription required")
}

if status != "active" {
	return errors.New("subscription inactive")
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
// 🔥 VELOCITY / BOT ABUSE CHECK (PLACE HERE)
//////////////////////////////////////////////////////////////

var count int

_ = tx.QueryRowContext(ctx, `
	SELECT COUNT(*)
	FROM contest_entries
	WHERE user_id=$1
	AND created_at > NOW() - INTERVAL '1 hour'
`, userID).Scan(&count)

if count > 20 {
	// increase risk score (silent)
	_, _ = tx.ExecContext(ctx, `
		INSERT INTO user_risk_profiles (user_id, risk_score)
		VALUES ($1,15)
		ON CONFLICT (user_id)
		DO UPDATE SET risk_score = user_risk_profiles.risk_score + 15
	`, userID)
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
	// UPDATE SPOTS (RACE SAFE)
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
	// LEADERBOARD ENTRY
	//////////////////////////////////////////////////////////////

	_, err = tx.ExecContext(ctx, `
		INSERT INTO leaderboard (contest_id, team_id, user_id, points, rank)
		VALUES ($1,$2,$3,0,0)
	`, contestID, teamID, userID)

	if err != nil {
		return err
	}

	//////////////////////////////////////////////////////////////
	// 🔥 AUDIT LOG (FORENSIC)
	//////////////////////////////////////////////////////////////

	_, _ = tx.ExecContext(ctx, `
		INSERT INTO audit_logs (user_id, action, metadata)
		VALUES ($1,$2,$3)
	`, userID, "join_contest", `{"contest_id":`+strconv.Itoa(contestID)+`}`)

	//////////////////////////////////////////////////////////////
	// 🔥 SMALL RISK UPDATE (BEHAVIOR TRACK)
	//////////////////////////////////////////////////////////////

	_, _ = tx.ExecContext(ctx, `
		INSERT INTO user_risk_profiles (user_id, risk_score)
		VALUES ($1,0.5)
		ON CONFLICT (user_id)
		DO UPDATE SET risk_score = user_risk_profiles.risk_score + 0.5
	`, userID)

	//////////////////////////////////////////////////////////////
	// COMMIT
	//////////////////////////////////////////////////////////////

	if err := tx.Commit(); err != nil {
		return err
	}

	//////////////////////////////////////////////////////////////
	// CACHE INVALIDATION
	//////////////////////////////////////////////////////////////

	cache.Rdb.Del(cache.Ctx, "contests:"+strconv.Itoa(matchID))

	go func() {
		cache.Rdb.Del(cache.Ctx, "leaderboard:"+strconv.Itoa(contestID))
	}()

	return nil
}

//////////////////////////////////////////////////////////////
// RETRY WRAPPER (COCKROACH SAFE)
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

	if err == nil {
		return false
	}

	msg := err.Error()

	return strings.Contains(msg, "restart transaction") ||
		strings.Contains(msg, "deadlock") ||
		strings.Contains(msg, "serialization")
}
