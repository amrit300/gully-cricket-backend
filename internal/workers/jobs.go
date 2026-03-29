package workers

import (
	"database/sql"
	"log"

	dbutil "gully-cricket/internal/db"
	"gully-cricket/internal/services"
	"gully-cricket/internal/queue"
)

var DB *sql.DB // injected from main

func handleLeaderboard(data interface{}) error {

	contestID, ok := data.(int)
	if !ok {
		return nil
	}

	//////////////////////////////////////////////////////////////
	// ✅ CONTEXT (MANDATORY)
	//////////////////////////////////////////////////////////////

	ctx, cancel := dbutil.Ctx()
	defer cancel()

	rows, err := DB.QueryContext(ctx, `
		SELECT team_id
		FROM contest_entries
		WHERE contest_id = $1
	`, contestID)

	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {

		var teamID int
		if err := rows.Scan(&teamID); err != nil {
			log.Println("SCAN ERROR:", err)
			continue
		}

		//////////////////////////////////////////////////////////////
		// 🔥 FRAUD / ANOMALY HOOK (FUTURE ML / GRAPH)
		//////////////////////////////////////////////////////////////

		queue.Enqueue(queue.Job{
			Type: "fraud_check",
			Data: teamID,
		})

		//////////////////////////////////////////////////////////////
		// ✅ PASS CONTEXT DOWNSTREAM (IMPORTANT)
		//////////////////////////////////////////////////////////////

		points, err := services.CalculateTeamPointsWithCtx(ctx, DB, teamID)
		if err != nil {
			log.Println("POINT CALC ERROR:", err)

			// 🔥 RETRY SAFE
			queue.Retry(queue.Job{
				Type: "leaderboard_update",
				Data: contestID,
			})

			continue
		}

		//////////////////////////////////////////////////////////////
		// ✅ REDIS LEADERBOARD UPDATE
		//////////////////////////////////////////////////////////////

		err = services.UpdateLeaderboardScore(contestID, teamID, points)
		if err != nil {
			log.Println("REDIS UPDATE ERROR:", err)

			// 🔥 RETRY SAFE
			queue.Retry(queue.Job{
				Type: "leaderboard_update",
				Data: contestID,
			})
		}
	}

	//////////////////////////////////////////////////////////////
	// ✅ ROWS ERROR CHECK
	//////////////////////////////////////////////////////////////

	if err := rows.Err(); err != nil {
		return err
	}

	return nil
}
func handleFraudCheck(data interface{}) {

	userID, ok := data.(int)
	if !ok {
		return
	}

	log.Println("🔍 Fraud check triggered for user/team:", userID)

	// future:
	// → ML scoring
	// → graph detection
	// → anomaly detection
}
