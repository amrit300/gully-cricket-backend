package services

import (
	"database/sql"
	"errors"
	"time"
)

func RequestWithdrawal(db *sql.DB, userID int, amount float64) error {

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	//////////////////////////////////////////////////////////////
	// 1. CHECK BALANCE (SUBSCRIPTION WALLET)
	//////////////////////////////////////////////////////////////

	var balance float64

	err = tx.QueryRow(`
		SELECT subscription_balance
		FROM users
		WHERE id=$1
		FOR UPDATE
	`, userID).Scan(&balance)

	if err != nil {
		return err
	}

	if balance < amount {
		return errors.New("insufficient balance")
	}

	//////////////////////////////////////////////////////////////
	// 2. DEDUCT BALANCE
	//////////////////////////////////////////////////////////////

	err = DeductSubscription(tx, userID, amount)
	if err != nil {
		return err
	}

	//////////////////////////////////////////////////////////////
	// 3. CREATE WITHDRAWAL REQUEST
	//////////////////////////////////////////////////////////////

	_, err = tx.Exec(`
		INSERT INTO withdrawals (user_id, amount, status, created_at)
		VALUES ($1,$2,'pending',$3)
	`, userID, amount, time.Now())

	if err != nil {
		return err
	}

	return tx.Commit()
}
