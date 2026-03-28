package services

import (
	"context"
	"database/sql"
	"log"
	"time"
)

//////////////////////////////////////////////////////////////
// 🚀 WORKER (IMPROVED — CONTROLLED LOOP)
//////////////////////////////////////////////////////////////

func StartLeaderboardWorker(db *sql.DB) {

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		processLeaderboard(db)
	}
}

//////////////////////////////////////////////////////////////
// 🔥 CORE ENGINE
//////////////////////////////////////////////////////////////

func processLeaderboard(db *sql.DB) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, `
		SELECT DISTINCT l.contest_id, c.match_id
		FROM leaderboard l
		JOIN contests c ON c.id = l.contest_id
		WHERE c.status = 'upcoming'
		`)
	if err != nil {
		log.Println("worker query error:", err)
		return
	}
	defer rows.Close()

	for rows.Next() {

		var contestID int
		var matchID int

		if err := rows.Scan(&contestID, &matchID); err != nil {
			continue
		}

		// 🔐 isolate each contest execution
		runContestPipeline(db, contestID, matchID)
	}

	if err := rows.Err(); err != nil {
		log.Println("rows error:", err)
	}
}

//////////////////////////////////////////////////////////////
// 🔐 TRANSACTIONAL PIPELINE (CRITICAL)
//////////////////////////////////////////////////////////////

func runContestPipeline(db *sql.DB, contestID, matchID int) {

	tx, err := db.Begin()
if err != nil {
	log.Println("tx begin error:", err)
	return
}

// 🔐 LOCK CONTEST (PREVENT PARALLEL PIPELINE)
var lock int
err = tx.QueryRow(`
	SELECT id FROM contests WHERE id=$1 FOR UPDATE
`, contestID).Scan(&lock)

if err != nil {
	log.Println("lock error:", err)
	tx.Rollback()
	return
}
	defer tx.Rollback()

	if err := updateTeamPoints(tx, matchID); err != nil {
		log.Println("team points error:", err)
		return
	}

	if err := syncLeaderboardPoints(tx, contestID); err != nil {
		log.Println("sync error:", err)
		return
	}

	if err := updateDenseRanks(tx, contestID); err != nil {
		log.Println("rank error:", err)
		return
	}

	if err := assignWinnings(tx, contestID); err != nil {
		log.Println("winnings error:", err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Println("commit error:", err)
	}
}

//////////////////////////////////////////////////////////////
// STEP 1 → TEAM POINTS
//////////////////////////////////////////////////////////////

func updateTeamPoints(tx *sql.Tx, matchID int) error {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := tx.ExecContext(ctx, `
		UPDATE teams t
		SET total_points = sub.points
		FROM (
			SELECT 
			  t2.id as team_id,
			  COALESCE(SUM(
			    CASE 
			      WHEN tp.player_id = t2.captain_player_id THEN p.fantasy_points * 2
			      WHEN tp.player_id = t2.vice_captain_player_id THEN p.fantasy_points * 1.5
			      ELSE p.fantasy_points
			    END
			  ),0) as points
			FROM teams t2
			JOIN team_players tp ON tp.team_id = t2.id
			JOIN players p ON p.id = tp.player_id
			WHERE t2.match_id = $1
			GROUP BY t2.id
		) sub
		WHERE t.id = sub.team_id
		AND t.match_id = $1
	`, matchID)

	return err
}

//////////////////////////////////////////////////////////////
// STEP 2 → SYNC LEADERBOARD
//////////////////////////////////////////////////////////////

func syncLeaderboardPoints(tx *sql.Tx, contestID int) error {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := tx.ExecContext(ctx, `
		UPDATE leaderboard l
		SET points = t.total_points
		FROM teams t
		WHERE l.team_id = t.id
		AND l.contest_id = $1
	`, contestID)

	return err
}

//////////////////////////////////////////////////////////////
// STEP 3 → DENSE RANK
//////////////////////////////////////////////////////////////

func updateDenseRanks(tx *sql.Tx, contestID int) error {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := tx.ExecContext(ctx, `
		UPDATE leaderboard l
		SET rank = r.rank
		FROM (
			SELECT 
			  team_id,
			  DENSE_RANK() OVER (ORDER BY points DESC) as rank
			FROM leaderboard
			WHERE contest_id = $1
		) r
		WHERE l.team_id = r.team_id
		AND l.contest_id = $1
	`, contestID)

	return err
}

//////////////////////////////////////////////////////////////
// STEP 4 → ASSIGN WINNINGS (SAFE + IDEMPOTENT)
//////////////////////////////////////////////////////////////

func assignWinnings(tx *sql.Tx, contestID int) error {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := tx.QueryContext(ctx, `
		SELECT team_id, rank
		FROM leaderboard
		WHERE contest_id = $1
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {

		var teamID int
		var rank int

		if err := rows.Scan(&teamID, &rank); err != nil {
			continue
		}

		var amount float64

		err := tx.QueryRowContext(ctx, `
			SELECT amount
			FROM contest_prizes
			WHERE contest_id=$1
			AND $2 BETWEEN rank_start AND rank_end
			LIMIT 1
		`, contestID, rank).Scan(&amount)

		if err != nil {
			continue
		}

		// 🔐 update only if changed (prevents unnecessary writes)
		_, err = tx.ExecContext(ctx, `
			UPDATE leaderboard
			SET winnings = $1
			WHERE contest_id=$2 
			AND team_id=$3
			AND winnings IS DISTINCT FROM $1
		`, amount, contestID, teamID)

		if err != nil {
			return err
		}
	}

	return rows.Err()
}
