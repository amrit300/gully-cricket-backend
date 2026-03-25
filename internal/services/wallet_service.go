package services

import (
	"database/sql"
	"errors"
	"time"
)

//////////////////////////////////////////////////////////////
// 📊 GET SUBSCRIPTION BALANCE
//////////////////////////////////////////////////////////////

func GetBalance(db *sql.DB, userID int) (float64, error) {

	var balance float64

	err := db.QueryRow(`
		SELECT subscription_balance FROM users WHERE id=$1
	`, userID).Scan(&balance)

	if err != nil {
		return 0, err
	}

	return balance, nil
}

//////////////////////////////////////////////////////////////
// ➖ SUBSCRIPTION DEDUCTION (MONTHLY)
//////////////////////////////////////////////////////////////

func DeductSubscription(tx *sql.Tx, userID int, amount float64) error {

	var balance float64

	err := tx.QueryRow(`
		SELECT subscription_balance
		FROM users
		WHERE id=$1
		FOR UPDATE
	`, userID).Scan(&balance)

	if err != nil {
		return err
	}

	if balance < amount {
		return errors.New("subscription balance insufficient")
	}

	_, err = tx.Exec(`
		UPDATE users
		SET subscription_balance = subscription_balance - $1
		WHERE id=$2
	`, amount, userID)

	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO wallet_transactions (user_id, amount, type, created_at)
		VALUES ($1,$2,'subscription_debit',$3)
	`, userID, amount, time.Now())

	return err
}

//////////////////////////////////////////////////////////////
// ➕ ADD FUNDS (USER DEPOSIT)
//////////////////////////////////////////////////////////////

func CreditBalance(tx *sql.Tx, userID int, amount float64) error {

	_, err := tx.Exec(`
		UPDATE users
		SET subscription_balance = subscription_balance + $1
		WHERE id=$2
	`, amount, userID)

	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO wallet_transactions (user_id, amount, type, created_at)
		VALUES ($1,$2,'credit',$3)
	`, userID, amount, time.Now())

	return err
}
