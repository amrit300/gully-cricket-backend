package ingestion

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"gully-cricket/internal/cache"
	"gully-cricket/internal/providers"
	"gully-cricket/internal/queue"
)

func SyncMatchesToDBWithCtx(ctx context.Context, db *sql.DB) error {

	matches, err := providers.FetchMatches()
	if err != nil {
		return err
	}

	log.Println("MATCHES RECEIVED:", len(matches))

	if len(matches) == 0 {
		log.Println("⚠️ NO MATCHES FROM API")
		return nil
	}

	for _, m := range matches {

		var teamA, teamB, venue, status, matchID string
		var startTime time.Time

		//////////////////////////////////////////////////////////////
		// 🔥 DETECT SOURCE (STRICT)
		//////////////////////////////////////////////////////////////

		if _, ok := m["teama"]; ok {
			// ✅ ENTITY API

			teama := m["teama"].(map[string]interface{})
			teamb := m["teamb"].(map[string]interface{})
			venueObj := m["venue"].(map[string]interface{})

			teamA = fmt.Sprintf("%v", teama["name"])
			teamB = fmt.Sprintf("%v", teamb["name"])
			venue = fmt.Sprintf("%v", venueObj["name"])
			status = fmt.Sprintf("%v", m["status_str"])
			matchID = fmt.Sprintf("%v", m["match_id"])

			startTimeStr := fmt.Sprintf("%v", m["date_start"])
			startTime, err = time.Parse("2006-01-02 15:04:05", startTimeStr)

		} else if _, ok := m["teams"]; ok {
			// ✅ CRIC API

			teams, _ := m["teams"].([]interface{})

			if len(teams) < 2 {
				continue
			}

			teamA = fmt.Sprintf("%v", teams[0])
			teamB = fmt.Sprintf("%v", teams[1])
			venue = fmt.Sprintf("%v", m["venue"])
			matchID = fmt.Sprintf("%v", m["id"])

			rawStatus := fmt.Sprintf("%v", m["status"])

			// 🔥 NORMALIZE STATUS (CRITICAL)
			lower := strings.ToLower(rawStatus)

			if strings.Contains(lower, "won") {
				status = "completed"
			} else {
				status = "live"
			}

			startTimeStr := fmt.Sprintf("%v", m["dateTimeGMT"])
			startTime, err = time.Parse(time.RFC3339, startTimeStr)
		}

		//////////////////////////////////////////////////////////////
		// 🚨 VALIDATION
		//////////////////////////////////////////////////////////////

		if teamA == "" || teamB == "" {
			log.Println("❌ INVALID MATCH, SKIPPING")
			continue
		}

		if err != nil {
			log.Println("❌ TIME PARSE ERROR:", err)
			continue
		}

		//////////////////////////////////////////////////////////////
		// 🔥 UPSERT
		//////////////////////////////////////////////////////////////

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
			log.Println("❌ DB ERROR:", err)
			continue
		}

		//////////////////////////////////////////////////////////////
		// 🔥 EVENT TRIGGER (QUEUE USED → NO BUILD ERROR)
		//////////////////////////////////////////////////////////////

		if status == "completed" {
			queue.Enqueue(queue.Job{
				Type: "match_complete",
				Data: matchID,
				Key:  matchID,
			})
		}

		log.Println("✅ SYNCED:", teamA, "vs", teamB)
	}

	//////////////////////////////////////////////////////////////
	// 🔥 SAFE CACHE CLEAR
	//////////////////////////////////////////////////////////////

	if cache.Rdb != nil {
		cache.Rdb.Del(cache.Ctx, "matches:v1")
	}

	log.Println("✅ MATCH SYNC COMPLETED")

	return nil
}

func SyncMatchesToDB(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return SyncMatchesToDBWithCtx(ctx, db)
}
