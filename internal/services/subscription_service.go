package services

import (
	"database/sql"
	"time"
)

func SubscribeUser(db *sql.DB, userID, planID int) error {

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var price float64
	var duration int

	err = tx.QueryRow(`
		SELECT price, duration_days FROM plans WHERE id=$1
	`, planID).Scan(&price, &duration)

	if err != nil {
		return err
	}

	err = DebitForSubscription(tx, userID, price)
	if err != nil {
		return err
	}

	now := time.Now()
	expiry := now.Add(time.Duration(duration) * 24 * time.Hour)

	_, err = tx.Exec(`
		INSERT INTO user_subscriptions (user_id, plan_id, status, started_at, expires_at)
		VALUES ($1,$2,'active',$3,$4)
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

	return tx.Commit()
}
