package services

import (
	"database/sql"
	"errors"
	"time"
)

//////////////////////////////////////////////////////////////
// SUBSCRIBE USER (CORE FUNCTION)
//////////////////////////////////////////////////////////////

func SubscribeUser(db *sql.DB, userID, planID int) error {

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	//////////////////////////////////////////////////////////////
	// GET PLAN
	//////////////////////////////////////////////////////////////

	var price float64
	var duration int

	err = tx.QueryRow(`
		SELECT price, duration_days
		FROM plans
		WHERE id=$1
	`, planID).Scan(&price, &duration)

	if err != nil {
		return err
	}

	//////////////////////////////////////////////////////////////
	// WALLET CHECK
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
	// DEDUCT BALANCE
	//////////////////////////////////////////////////////////////

	_, err = tx.Exec(`
		UPDATE wallets
		SET balance = balance - $1
		WHERE user_id=$2
	`, price, userID)

	if err != nil {
		return err
	}

	//////////////////////////////////////////////////////////////
	// WALLET TRANSACTION LOG
	//////////////////////////////////////////////////////////////

	_, err = tx.Exec(`
		INSERT INTO wallet_transactions (user_id, amount, type)
		VALUES ($1,$2,'subscription')
	`, userID, price)

	if err != nil {
		return err
	}

	//////////////////////////////////////////////////////////////
	// CREATE / UPDATE SUBSCRIPTION
	//////////////////////////////////////////////////////////////

	now := time.Now()
	expiry := now.Add(time.Duration(duration) * 24 * time.Hour)

	_, err = tx.Exec(`
		INSERT INTO user_subscriptions
		(user_id, plan_id, status, started_at, expires_at, auto_renew)
		VALUES ($1,$2,'active',$3,$4,TRUE)
		ON CONFLICT (user_id)
		DO UPDATE SET
			plan_id=$2,
			status='active',
			started_at=$3,
			expires_at=$4
	`, userID, planID, now, expiry)

	if err != nil {
		return err
	}

	//////////////////////////////////////////////////////////////
	// COMMIT
	//////////////////////////////////////////////////////////////

	return tx.Commit()
}
