package services

import "database/sql"

func GetBalance(db *sql.DB, userID int) (float64, error) {

	var balance float64

	err := db.QueryRow(`
		SELECT wallet_balance FROM users WHERE id=$1
	`, userID).Scan(&balance)

	if err != nil {
		return 0, err
	}

	return balance, nil
}
