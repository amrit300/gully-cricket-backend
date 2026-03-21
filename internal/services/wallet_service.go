package services

import (
	"database/sql"
	"fmt"
)

//////////////////////////////////////////////////////////////
// CREDIT WALLET (WINNINGS / REFERRALS)
//////////////////////////////////////////////////////////////

func CreditWallet(tx *sql.Tx, userID int, amount float64, reason string) error {

	if amount <= 0 {
		return fmt.Errorf("invalid amount")
	}

	// 1. UPDATE BALANCE
	_, err := tx.Exec(`
		UPDATE users
		SET wallet_balance = wallet_balance + $1
		WHERE id=$2
	`, amount, userID)

	if err != nil {
		return err
	}

	// 2. LOG TRANSACTION
	_, err = tx.Exec(`
		INSERT INTO transactions (user_id, amount, transaction_type, status)
		VALUES ($1,$2,$3,'success')
	`, userID, amount, reason)

	return err
}

//////////////////////////////////////////////////////////////
// DEBIT WALLET (CONTEST ENTRY)
//////////////////////////////////////////////////////////////

func DebitWallet(tx *sql.Tx, userID int, amount float64, reason string) error {

	if amount <= 0 {
		return fmt.Errorf("invalid amount")
	}

	// 🔥 ATOMIC CHECK + DEDUCT
	res, err := tx.Exec(`
		UPDATE users
		SET wallet_balance = wallet_balance - $1
		WHERE id=$2 AND wallet_balance >= $1
	`, amount, userID)

	if err != nil {
		return err
	}

	rows, _ := res.RowsAffected()

	if rows == 0 {
		return fmt.Errorf("insufficient balance")
	}

	// LOG TRANSACTION
	_, err = tx.Exec(`
		INSERT INTO transactions (user_id, amount, transaction_type, status)
		VALUES ($1,$2,$3,'success')
	`, userID, -amount, reason)

	return err
}

//////////////////////////////////////////////////////////////
// GET BALANCE
//////////////////////////////////////////////////////////////

func GetWalletBalance(db *sql.DB, userID int) (float64, error) {

	var balance float64

	err := db.QueryRow(`
		SELECT wallet_balance FROM users WHERE id=$1
	`, userID).Scan(&balance)

	return balance, err
}
