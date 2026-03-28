package ingestion

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	dbutil "gully-cricket/internal/db"
	"gully-cricket/internal/providers"
)

func SyncMatchesToDB(db *sql.DB) error {

	matches, err := providers.FetchMatchesFromEntityAPI()
	if err != nil {
		return err
	}

	log.Println("MATCHES RECEIVED:", len(matches))

	if len(matches) == 0 {
		log.Println("⚠️ NO MATCHES FROM API")
		return nil
	}

	for _, m := range matches {

		// ✅ SAFE EXTRACTION

		matchID := fmt.Sprintf("%v", m["match_id"])

		teama, okA := m["teama"].(map[string]interface{})
		teamb, okB := m["teamb"].(map[string]interface{})
		venueObj, okV := m["venue"].(map[string]interface{})

		if !okA || !okB || !okV {
			log.Println("❌ INVALID MATCH STRUCTURE, SKIPPING")
			continue
		}

		teamA := fmt.Sprintf("%v", teama["name"])
		teamB := fmt.Sprintf("%v", teamb["name"])
		venue := fmt.Sprintf("%v", venueObj["name"])
		status := fmt.Sprintf("%v", m["status_str"])

		startTimeStr := fmt.Sprintf("%v", m["date_start"])

		startTime, err := time.Parse("2006-01-02 15:04:05", startTimeStr)
		if err != nil {
			log.Println("❌ TIME PARSE ERROR:", err)
			continue
		}

		// ✅ INSERT
		ctx, cancel := dbutil.Ctx()
		defer cancel()
		
		_, err = db.ExecContext(ctx, `
			INSERT INTO matches_master
			(external_id, team_a, team_b, venue, start_time, status)
			VALUES ($1,$2,$3,$4,$5,$6)
			ON CONFLICT (external_id)
			DO UPDATE SET
				team_a = EXCLUDED.team_a,
				team_b = EXCLUDED.team_b,
				venue = EXCLUDED.venue,
				start_time = EXCLUDED.start_time,
				status = EXCLUDED.status
		`,
			matchID,
			teamA,
			teamB,
			venue,
			startTime,
			status,
		)

		if err != nil {
			log.Println("❌ MATCH INSERT ERROR:", err)
			continue
		}

		log.Println("✅ INSERTED:", teamA, "vs", teamB)
	}

	log.Println("✅ MATCH SYNC COMPLETED")

	return nil
}
