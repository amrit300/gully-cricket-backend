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
		SELECT COALESCE(subscription_balance, 0)
		FROM users
		WHERE id=$1
	`, userID).Scan(&balance)

	if err != nil {
		return 0, err
	}

	return balance, nil
}

//////////////////////////////////////////////////////////////
// ➖ SUBSCRIPTION DEDUCTION (ATOMIC + SAFE)
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

	//////////////////////////////////////////////////////////////
	// 🔐 ATOMIC DEDUCTION (NO RACE CONDITION)
	//////////////////////////////////////////////////////////////

	result, err := tx.ExecContext(ctx, `
		UPDATE users
		SET subscription_balance = subscription_balance - $1
		WHERE id=$2 AND subscription_balance >= $1
	`, amount, userID)

	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("insufficient balance")
	}

	//////////////////////////////////////////////////////////////
	// 📒 LEDGER ENTRY (STRICT TYPE)
	//////////////////////////////////////////////////////////////

	_, err = tx.ExecContext(ctx, `
		INSERT INTO wallet_transactions (user_id, amount, type, created_at)
		VALUES ($1,$2,'subscription_debit',$3)
	`, userID, amount, time.Now())

	return err
}

//////////////////////////////////////////////////////////////
// ➕ ADD FUNDS (WEBHOOK SAFE + IDEMPOTENT)
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
	// 🔐 IDEMPOTENCY LOCK (STRONG)
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
		return nil // ✅ idempotent success (important)
	}

	//////////////////////////////////////////////////////////////
	// 💰 ATOMIC CREDIT
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
