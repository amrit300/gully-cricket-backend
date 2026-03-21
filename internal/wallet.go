package internal

import (
	"database/sql"
	"fmt"
)

/* =========================
   CREDIT WALLET
========================= */

func CreditWallet(db *sql.DB, userID int, amount float64, refID int, txType string) error {

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
	UPDATE users
	SET wallet_balance = wallet_balance + $1
	WHERE id=$2
	`, amount, userID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
	INSERT INTO transactions (user_id, amount, type, reference_id)
	VALUES ($1,$2,$3,$4)
	`, userID, amount, txType, refID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

/* =========================
   DEBIT WALLET
========================= */

func DebitWallet(db *sql.DB, userID int, amount float64, refID int) error {

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var balance float64

	err = tx.QueryRow(`
	SELECT wallet_balance FROM users
	WHERE id=$1 FOR UPDATE
	`, userID).Scan(&balance)

	if err != nil {
		return err
	}

	if balance < amount {
		return fmt.Errorf("insufficient balance")
	}

	_, err = tx.Exec(`
	UPDATE users
	SET wallet_balance = wallet_balance - $1
	WHERE id=$2
	`, amount, userID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
	INSERT INTO transactions (user_id, amount, type, reference_id)
	VALUES ($1,$2,'entry_fee',$3)
	`, userID, amount, refID)
	if err != nil {
		return err
	}

	return tx.Commit()
}
