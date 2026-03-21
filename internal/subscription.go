package internal

import "database/sql"

var PlanLimits = map[string]int{
	"basic":   1,
	"pro":     3,
	"elite":   5,
	"legends": 20,
}

func ActivateSubscription(db *sql.DB, userID int, plan string, price float64) error {

	_, err := db.Exec(`
	INSERT INTO subscriptions
	(user_id, plan, price, start_date, end_date, status)
	VALUES ($1,$2,$3,NOW(),NOW()+INTERVAL '30 days','active')
	`, userID, plan, price)

	if err != nil {
		return err
	}

	_, err = db.Exec(`
	UPDATE users
	SET plan=$1,
	    max_teams_per_match=$2
	WHERE id=$3
	`, plan, PlanLimits[plan], userID)

	return err
}
