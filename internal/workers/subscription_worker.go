package workers

import (
	"log"

	"gully-cricket/internal/services"
)

func ProcessRenewals() {

	rows, err := DB.Query(`
		SELECT user_id, plan_id
		FROM user_subscriptions
		WHERE expires_at < NOW()
		AND auto_renew = TRUE
		AND renewal_lock = FALSE
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

		err := services.RenewSubscription(DB, userID, planID)

		if err != nil {
			log.Println("RENEW FAILED:", userID, err)

			_, _ = DB.Exec(`
				UPDATE user_subscriptions
				SET status='grace'
				WHERE user_id=$1
			`, userID)

		} else {
			log.Println("RENEW SUCCESS:", userID)
		}
	}
}
