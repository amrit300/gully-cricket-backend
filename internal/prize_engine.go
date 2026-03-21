package internal

import (
	"database/sql"
	"log"
)

func DistributePrizes(db *sql.DB, contestID int) error {

	rows, err := db.Query(`
	SELECT rank, COUNT(*) 
	FROM leaderboard
	WHERE contest_id=$1
	GROUP BY rank
	ORDER BY rank ASC
	`, contestID)

	if err != nil {
		return err
	}
	defer rows.Close()

	currentRank := 1

	for rows.Next() {

		var rank int
		var count int

		err := rows.Scan(&rank, &count)
		if err != nil {
			return err
		}

		startRank := currentRank
		endRank := currentRank + count - 1

		// get total prize for this range
		var totalPrize float64

		err = db.QueryRow(`
		SELECT COALESCE(SUM(amount),0)
		FROM contest_prizes
		WHERE contest_id=$1
		AND rank_start >= $2
		AND rank_end <= $3
		`, contestID, startRank, endRank).Scan(&totalPrize)

		if err != nil {
			return err
		}

		if totalPrize == 0 {
			currentRank += count
			continue
		}

		perUser := totalPrize / float64(count)

		// update winnings
		_, err = db.Exec(`
		UPDATE leaderboard
		SET winnings = $1
		WHERE contest_id=$2 AND rank=$3
		`, perUser, contestID, rank)

		if err != nil {
			log.Println("payout error:", err)
		}

		currentRank += count
	}

	return nil
}
