package services

import (
	"database/sql"
	"log"
	"time"
)

func StartLeaderboardWorker(db *sql.DB) {

	for {
		time.Sleep(10 * time.Second)

		processLeaderboard(db)
	}
}

//////////////////////////////////////////////////////////////
// CORE ENGINE
//////////////////////////////////////////////////////////////

func processLeaderboard(db *sql.DB) {

	rows, err := db.Query(`
		SELECT DISTINCT contest_id, match_id
		FROM leaderboard
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

		updateTeamPoints(db, matchID)
		syncLeaderboardPoints(db, contestID)
		updateDenseRanks(db, contestID)
		assignWinnings(db, contestID)
	}
}

//////////////////////////////////////////////////////////////
// STEP 1 → TEAM POINTS
//////////////////////////////////////////////////////////////

func updateTeamPoints(db *sql.DB, matchID int) {

	_, err := db.Exec(`
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

	if err != nil {
		log.Println("team points error:", err)
	}
}

//////////////////////////////////////////////////////////////
// STEP 2 → SYNC LEADERBOARD
//////////////////////////////////////////////////////////////

func syncLeaderboardPoints(db *sql.DB, contestID int) {

	_, err := db.Exec(`
		UPDATE leaderboard l
		SET points = t.total_points
		FROM teams t
		WHERE l.team_id = t.id
		AND l.contest_id = $1
	`, contestID)

	if err != nil {
		log.Println("leaderboard sync error:", err)
	}
}

//////////////////////////////////////////////////////////////
// STEP 3 → DENSE RANK (FIXED)
//////////////////////////////////////////////////////////////

func updateDenseRanks(db *sql.DB, contestID int) {

	_, err := db.Exec(`
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

	if err != nil {
		log.Println("rank update error:", err)
	}
}

//////////////////////////////////////////////////////////////
// STEP 4 → WINNINGS DISTRIBUTION
//////////////////////////////////////////////////////////////

func assignWinnings(db *sql.DB, contestID int) {

	rows, err := db.Query(`
		SELECT team_id, rank
		FROM leaderboard
		WHERE contest_id = $1
	`, contestID)

	if err != nil {
		log.Println("winnings fetch error:", err)
		return
	}
	defer rows.Close()

	for rows.Next() {

		var teamID int
		var rank int

		if err := rows.Scan(&teamID, &rank); err != nil {
			continue
		}

		var amount float64

		err := db.QueryRow(`
			SELECT amount
			FROM contest_prizes
			WHERE contest_id=$1
			AND $2 BETWEEN rank_start AND rank_end
			LIMIT 1
		`, contestID, rank).Scan(&amount)

		if err != nil {
			continue
		}

		_, err = db.Exec(`
			UPDATE leaderboard
			SET winnings = $1
			WHERE contest_id=$2 AND team_id=$3
		`, amount, contestID, teamID)

		if err != nil {
			log.Println("winnings update error:", err)
		}
	}
}
