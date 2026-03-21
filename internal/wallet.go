package internal

import "database/sql"

func CreditWallet(db *sql.DB, userID int, amount float64, refID int, txType string) error {

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// update balance
	_, err = tx.Exec(`
	UPDATE users
	SET wallet_balance = wallet_balance + $1
	WHERE id=$2
	`, amount, userID)
	if err != nil {
		return err
	}

	// log transaction
	_, err = tx.Exec(`
	INSERT INTO transactions (user_id, amount, type, reference_id)
	VALUES ($1,$2,$3,$4)
	`, userID, amount, txType, refID)
	if err != nil {
		return err
	}

	return tx.Commit()
}
