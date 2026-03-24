package services

import (
	"database/sql"
	"fmt"
)

func DeductBalance(tx *sql.Tx, userID int, amount float64) error {

	var balance float64

	err := tx.QueryRow(`
		SELECT wallet_balance FROM users WHERE id=$1 FOR UPDATE
	`, userID).Scan(&balance)

	if err != nil {
		return err
	}

	if balance < amount {
		return fmt.Errorf("insufficient balance")
	}

	_, err = tx.Exec(`
		UPDATE users SET wallet_balance = wallet_balance - $1 WHERE id=$2
	`, amount, userID)

	return err
}

func CreditBalance(tx *sql.Tx, userID int, amount float64, reference string) error {

	_, err := tx.Exec(`
		UPDATE users SET wallet_balance = wallet_balance + $1 WHERE id=$2
	`, amount, userID)

	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO transactions (user_id, amount, transaction_type, reference_id)
		VALUES ($1,$2,'credit',$3)
	`, userID, amount, reference)

	return err
}
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
