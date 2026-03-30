package workers

import (
	"log"
	"time"

	"gully-cricket/internal/services"
)

//////////////////////////////////////////////////////////////
// AUTO RENEW LOOP
//////////////////////////////////////////////////////////////

func StartSubscriptionWorker() {

	go func() {
		for {

			ProcessRenewals()

			time.Sleep(1 * time.Hour)
		}
	}()
}

//////////////////////////////////////////////////////////////
// RENEWAL LOGIC
//////////////////////////////////////////////////////////////

func ProcessRenewals() {

	rows, err := DB.Query(`
		SELECT user_id, plan_id
		FROM user_subscriptions
		WHERE expires_at < NOW()
		AND auto_renew = TRUE
	`)

	if err != nil {
		log.Println("RENEW QUERY ERROR:", err)
		return
	}
	defer rows.Close()

	for rows.Next() {

		var userID, planID int
		if err := rows.Scan(&userID, &planID); err != nil {
			continue
		}

		err := services.SubscribeUser(DB, userID, planID)

		if err != nil {
			log.Println("RENEW FAILED:", userID)

			DB.Exec(`
				UPDATE user_subscriptions
				SET status='grace'
				WHERE user_id=$1
			`, userID)

		} else {
			log.Println("RENEW SUCCESS:", userID)
		}
	}
}
