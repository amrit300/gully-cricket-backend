package services

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

//////////////////////////////////////////////////////////////
// SAFE SUBSCRIPTION (IDEMPOTENT + TRANSACTIONAL)
//////////////////////////////////////////////////////////////

func RenewSubscription(db *sql.DB, userID, planID int) error {

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	//////////////////////////////////////////////////////////////
	// LOCK SUBSCRIPTION (PREVENT RACE)
	//////////////////////////////////////////////////////////////

	var locked bool
	err = tx.QueryRow(`
		SELECT renewal_lock FROM user_subscriptions
		WHERE user_id=$1 FOR UPDATE
	`, userID).Scan(&locked)

	if err != nil {
		return err
	}

	if locked {
		return errors.New("already processing")
	}

	_, _ = tx.Exec(`
		UPDATE user_subscriptions SET renewal_lock = TRUE
		WHERE user_id=$1
	`, userID)

	//////////////////////////////////////////////////////////////
	// GET PLAN
	//////////////////////////////////////////////////////////////

	var price float64
	var duration int

	err = tx.QueryRow(`
		SELECT price, duration_days FROM plans WHERE id=$1
	`, planID).Scan(&price, &duration)

	if err != nil {
		return err
	}

	//////////////////////////////////////////////////////////////
	// WALLET LOCK + BALANCE CHECK
	//////////////////////////////////////////////////////////////

	var balance float64

	err = tx.QueryRow(`
		SELECT balance FROM wallets WHERE user_id=$1 FOR UPDATE
	`, userID).Scan(&balance)

	if err != nil {
		return err
	}

	if balance < price {
		return errors.New("insufficient balance")
	}

	//////////////////////////////////////////////////////////////
	// IDEMPOTENCY KEY (PREVENT DOUBLE CHARGE)
	//////////////////////////////////////////////////////////////

	idemKey := fmt.Sprintf("renew_%d_%d", userID, time.Now().Unix()/3600)

	var exists int
	_ = tx.QueryRow(`
		SELECT 1 FROM payments WHERE idempotency_key=$1
	`, idemKey).Scan(&exists)

	if exists == 1 {
		return nil // already processed
	}

	//////////////////////////////////////////////////////////////
	// DEBIT WALLET (LEDGER FIRST)
	//////////////////////////////////////////////////////////////

	_, err = tx.Exec(`
		INSERT INTO wallet_transactions (user_id, amount, type, status, reference)
		VALUES ($1,$2,'subscription_debit','success',$3)
	`, userID, price, idemKey)

	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		UPDATE wallets SET balance = balance - $1 WHERE user_id=$2
	`, price, userID)

	if err != nil {
		return err
	}

	//////////////////////////////////////////////////////////////
	// UPDATE SUBSCRIPTION
	//////////////////////////////////////////////////////////////

	now := time.Now()
	expiry := now.Add(time.Duration(duration) * 24 * time.Hour)

	_, err = tx.Exec(`
		UPDATE user_subscriptions
		SET status='active',
		    expires_at=$1,
		    renewal_lock=FALSE
		WHERE user_id=$2
	`, expiry, userID)

	if err != nil {
		return err
	}

	//////////////////////////////////////////////////////////////
	// PAYMENT LOG
	//////////////////////////////////////////////////////////////

	_, err = tx.Exec(`
		INSERT INTO payments (user_id, amount, status, idempotency_key)
		VALUES ($1,$2,'success',$3)
	`, userID, price, idemKey)

	if err != nil {
		return err
	}

	return tx.Commit()
}
