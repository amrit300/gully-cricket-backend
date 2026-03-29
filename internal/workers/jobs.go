package workers

import (
	"database/sql"
	"log"

	dbutil "gully-cricket/internal/db"
	"gully-cricket/internal/services"
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
			continue
		}

		//////////////////////////////////////////////////////////////
		// ✅ PASS CONTEXT DOWNSTREAM (IMPORTANT)
		//////////////////////////////////////////////////////////////

		points, err := services.CalculateTeamPointsWithCtx(ctx, DB, teamID)
		if err != nil {
			log.Println("POINT CALC ERROR:", err)
			continue
		}

		err = services.UpdateLeaderboardScore(contestID, teamID, points)
		if err != nil {
			log.Println("REDIS UPDATE ERROR:", err)
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
