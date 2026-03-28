package services

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

//////////////////////////////////////////////////////////////
// 📊 GET BALANCE
//////////////////////////////////////////////////////////////

func GetBalance(db *sql.DB, userID int) (float64, error) {

	if userID <= 0 {
		return 0, errors.New("invalid user id")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var balance float64

	err := db.QueryRowContext(ctx, `
		SELECT subscription_balance
		FROM users
		WHERE id=$1
	`, userID).Scan(&balance)

	if err != nil {
		return 0, err
	}

	return balance, nil
}

//////////////////////////////////////////////////////////////
// ➖ SUBSCRIPTION DEDUCTION
//////////////////////////////////////////////////////////////

func DeductSubscription(tx *sql.Tx, userID int, amount float64) error {

	if userID <= 0 {
		return errors.New("invalid user id")
	}

	if amount <= 0 {
		return errors.New("invalid amount")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var balance float64

	err := tx.QueryRowContext(ctx, `
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
	// UPDATE BALANCE
	//////////////////////////////////////////////////////////////

	_, err = tx.ExecContext(ctx, `
		UPDATE users
		SET subscription_balance = subscription_balance - $1
		WHERE id=$2
	`, amount, userID)

	if err != nil {
		return err
	}

	//////////////////////////////////////////////////////////////
	// LEDGER ENTRY
	//////////////////////////////////////////////////////////////

	_, err = tx.ExecContext(ctx, `
		INSERT INTO wallet_transactions (user_id, amount, type, created_at)
		VALUES ($1,$2,'subscription_debit',$3)
	`, userID, amount, time.Now())

	return err
}

//////////////////////////////////////////////////////////////
// ➕ ADD FUNDS (WEBHOOK ONLY)
//////////////////////////////////////////////////////////////

func AddFunds(tx *sql.Tx, userID int, amount float64, source string) error {

	if userID <= 0 {
		return errors.New("invalid user id")
	}

	if amount <= 0 {
		return errors.New("invalid amount")
	}

	if source == "" {
		return errors.New("transaction id required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	//////////////////////////////////////////////////////////////
	// 🔐 PREVENT DUPLICATE TRANSACTION
	//////////////////////////////////////////////////////////////

	var exists bool

	err := tx.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM wallet_transactions WHERE source=$1
		)
	`, source).Scan(&exists)

	if err != nil {
		return err
	}

	if exists {
		return errors.New("transaction already processed")
	}

	//////////////////////////////////////////////////////////////
	// 💰 CREDIT BALANCE
	//////////////////////////////////////////////////////////////

	_, err = tx.ExecContext(ctx, `
		UPDATE users
		SET subscription_balance = subscription_balance + $1
		WHERE id=$2
	`, amount, userID)

	if err != nil {
		return err
	}

	//////////////////////////////////////////////////////////////
	// 📒 LEDGER ENTRY
	//////////////////////////////////////////////////////////////

	_, err = tx.ExecContext(ctx, `
		INSERT INTO wallet_transactions (user_id, amount, type, source, created_at)
		VALUES ($1,$2,'credit',$3,$4)
	`, userID, amount, source, time.Now())

	return err
}
