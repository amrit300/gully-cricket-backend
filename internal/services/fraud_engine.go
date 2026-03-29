package services

import (
	"database/sql"
	"encoding/json"
)

//////////////////////////////////////////////////////////////
// UPDATE RISK SCORE
//////////////////////////////////////////////////////////////

func UpdateRiskScore(db *sql.DB, userID int, delta float64) error {

	_, err := db.Exec(`
		INSERT INTO user_risk_profiles (user_id, risk_score, last_updated)
		VALUES ($1,$2,NOW())
		ON CONFLICT (user_id)
		DO UPDATE SET
			risk_score = user_risk_profiles.risk_score + $2,
			last_updated = NOW()
	`, userID, delta)

	return err
}

//////////////////////////////////////////////////////////////
// GET RISK SCORE
//////////////////////////////////////////////////////////////

func GetRiskScore(db *sql.DB, userID int) (float64, error) {

	var score float64

	err := db.QueryRow(`
		SELECT risk_score FROM user_risk_profiles WHERE user_id=$1
	`, userID).Scan(&score)

	if err != nil {
		return 0, nil // default safe
	}

	return score, nil
}

//////////////////////////////////////////////////////////////
// AUDIT LOG
//////////////////////////////////////////////////////////////

func LogAction(db *sql.DB, userID int, action string, meta interface{}) {

	bytes, _ := json.Marshal(meta)

	_, _ = db.Exec(`
		INSERT INTO audit_logs (user_id, action, metadata)
		VALUES ($1,$2,$3)
	`, userID, action, bytes)
}
