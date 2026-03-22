package services

import (
	"database/sql"
	"fmt"
)

func RequestWithdrawal(db *sql.DB, userID int, amount float64) error {

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// Deduct balance safely
	err = DeductBalance(tx, userID, amount)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Insert withdrawal request
	_, err = tx.Exec(`
		INSERT INTO withdrawals (user_id, amount, status)
		VALUES ($1,$2,'pending')
	`, userID, amount)

	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func ProcessWithdrawal(db *sql.DB, withdrawalID int) error {

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	var userID int
	var amount float64

	err = tx.QueryRow(`
		SELECT user_id, amount
		FROM withdrawals
		WHERE id=$1 AND status='pending'
		FOR UPDATE
	`, withdrawalID).Scan(&userID, &amount)

	if err != nil {
		tx.Rollback()
		return fmt.Errorf("invalid withdrawal")
	}

	_, err = tx.Exec(`
		UPDATE withdrawals SET status='completed'
		WHERE id=$1
	`, withdrawalID)

	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
