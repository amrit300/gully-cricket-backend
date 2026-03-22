package services

import "database/sql"

func CheckUserContestLimit(tx *sql.Tx, userID, contestID int, maxTeams int) error {

	var count int

	err := tx.QueryRow(`
		SELECT COUNT(*)
		FROM contest_entries
		WHERE user_id=$1 AND contest_id=$2
	`, userID, contestID).Scan(&count)

	if err != nil {
		return err
	}

	if count >= maxTeams {
		return fmt.Errorf("max teams reached in this contest")
	}

	return nil
}
