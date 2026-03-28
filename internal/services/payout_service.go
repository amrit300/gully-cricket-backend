package services

import (
	"database/sql"
	"errors"
	"log"
	"time"
)

func ProcessContestPayout(db *sql.DB, contestID int) error {

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	//////////////////////////////////////////////////////////////
	// 1. CHECK IF ALREADY PROCESSED (IDEMPOTENCY)
	//////////////////////////////////////////////////////////////

	var status string
	err = tx.QueryRow(`
		SELECT status FROM contests WHERE id=$1
		FOR UPDATE
	`, contestID).Scan(&status)

	if err != nil {
		return err
	}

	if status == "completed" {
		return errors.New("payout already processed")
	}

	//////////////////////////////////////////////////////////////
	// 2. FETCH LEADERBOARD
	//////////////////////////////////////////////////////////////

	rows, err := tx.Query(`
		SELECT team_id, rank
		FROM leaderboard
		WHERE contest_id = $1
	`, contestID)

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

		//////////////////////////////////////////////////////////////
		// 3. GET PRIZE
		//////////////////////////////////////////////////////////////

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

		//////////////////////////////////////////////////////////////
		// 4. UPDATE LEADERBOARD
		//////////////////////////////////////////////////////////////

		_, err = tx.Exec(`
			UPDATE leaderboard
			SET winnings = $1
			WHERE contest_id=$2 AND team_id=$3
		`, amount, contestID, teamID)

		if err != nil {
			return err
		}

		//////////////////////////////////////////////////////////////
		// 5. GET USER ID
		//////////////////////////////////////////////////////////////

		var userID int
		err = tx.QueryRow(`
			SELECT user_id FROM teams WHERE id=$1
		`, teamID).Scan(&userID)

		if err != nil {
			continue
		}

		//////////////////////////////////////////////////////////////
		// 6. CREDIT WALLET (SAFE — LEDGER BASED)
		//////////////////////////////////////////////////////////////

		_, err = tx.Exec(`
			UPDATE users
			SET subscription_balance = subscription_balance + $1
			WHERE id=$2
		`, amount, userID)

		if err != nil {
			return err
		}

		//////////////////////////////////////////////////////////////
		// 7. LEDGER ENTRY (CRITICAL)
		//////////////////////////////////////////////////////////////

		_, err = tx.Exec(`
			INSERT INTO wallet_transactions (user_id, amount, type, source, created_at)
			VALUES ($1,$2,'winnings',$3,$4)
		`, userID, amount, "contest_"+string(rune(contestID)), time.Now())

		if err != nil {
			return err
		}
	}

	//////////////////////////////////////////////////////////////
	// 8. CHECK LOOP ERROR
	//////////////////////////////////////////////////////////////

	if err := rows.Err(); err != nil {
		return err
	}

	//////////////////////////////////////////////////////////////
	// 9. MARK COMPLETED
	//////////////////////////////////////////////////////////////

	_, err = tx.Exec(`
		UPDATE contests SET status='completed'
		WHERE id=$1
	`, contestID)

	if err != nil {
		return err
	}

	//////////////////////////////////////////////////////////////
	// 10. COMMIT
	//////////////////////////////////////////////////////////////

	return tx.Commit()
}
