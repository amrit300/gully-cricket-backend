package services

import (
	"database/sql"
	"log"
)

func ProcessContestPayout(db *sql.DB, contestID int) {

	tx, err := db.Begin()
	if err != nil {
		log.Println(err)
		return
	}

	rows, err := tx.Query(`
		SELECT team_id, rank
		FROM leaderboard
		WHERE contest_id = $1
	`)
	if err != nil {
		tx.Rollback()
		return
	}
	defer rows.Close()

	for rows.Next() {

		var teamID int
		var rank int

		rows.Scan(&teamID, &rank)

		var amount float64

		err := tx.QueryRow(`
			SELECT amount
			FROM contest_prizes
			WHERE contest_id=$1
			AND $2 BETWEEN rank_start AND rank_end
			LIMIT 1
		`, contestID, rank).Scan(&amount)

		if err != nil {
			continue
		}

		// update winnings
		_, err = tx.Exec(`
			UPDATE leaderboard
			SET winnings = $1
			WHERE contest_id=$2 AND team_id=$3
		`, amount, contestID, teamID)

		if err != nil {
			tx.Rollback()
			return
		}

		// credit wallet
		_, err = tx.Exec(`
			UPDATE users u
			SET wallet_balance = wallet_balance + $1
			FROM teams t
			WHERE t.id=$2 AND u.id=t.user_id
		`, amount, teamID)

		if err != nil {
			tx.Rollback()
			return
		}
	}

	// mark contest completed
	tx.Exec(`
		UPDATE contests SET status='completed'
		WHERE id=$1
	`, contestID)

	tx.Commit()
}
