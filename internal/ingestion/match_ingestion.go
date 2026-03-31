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

//////////////////////////////////////////////////////////////
// 🔥 ROBUST TIME PARSER (MULTI-FORMAT SUPPORT)
//////////////////////////////////////////////////////////////

func parseTimeSafe(input string) (time.Time, error) {

	input = strings.TrimSpace(input)

	// 🔥 FIX: add Z if missing
	if len(input) == 19 && strings.Contains(input, "T") {
		input = input + "Z"
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	var lastErr error

	for _, layout := range layouts {
		if t, err := time.Parse(layout, input); err == nil {
			return t.UTC(), nil
		} else {
			lastErr = err
		}
	}

	return time.Time{}, fmt.Errorf("unsupported time format: %s | %v", input, lastErr)
}
//////////////////////////////////////////////////////////////
// 🔥 SAFE STRING NORMALIZER
//////////////////////////////////////////////////////////////

func normalizeStatus(status string) string {
	s := strings.ToLower(status)

	switch {
	case strings.Contains(s, "won"),
		strings.Contains(s, "completed"),
		strings.Contains(s, "result"):
		return "completed"

	case strings.Contains(s, "live"),
		strings.Contains(s, "progress"):
		return "live"

	case strings.Contains(s, "upcoming"),
		strings.Contains(s, "scheduled"):
		return "upcoming"
	}

	return "completed" // fallback (safe default)
}

//////////////////////////////////////////////////////////////
// 🚀 MAIN SYNC FUNCTION (PRODUCTION SAFE)
//////////////////////////////////////////////////////////////

func SyncMatchesToDBWithCtx(ctx context.Context, db *sql.DB) error {

	start := time.Now()

	matches, err := providers.FetchMatches()
	if err != nil {
		return fmt.Errorf("fetch failed: %w", err)
	}

	log.Println("📡 MATCHES RECEIVED:", len(matches))

	if len(matches) == 0 {
		log.Println("⚠️ NO MATCHES FROM API")
		return nil
	}

	success := 0
	failed := 0
	skipped := 0

	for _, m := range matches {

		var teamA, teamB, venue, status, matchID string
		var startTime time.Time

		//////////////////////////////////////////////////////////////
		// ✅ CASE 1: ENTITY API
		//////////////////////////////////////////////////////////////

		if teama, ok := m["teama"].(map[string]interface{}); ok {

			teamb, okB := m["teamb"].(map[string]interface{})
			venueObj, okV := m["venue"].(map[string]interface{})

			if !okB || !okV {
				log.Println("❌ ENTITY INVALID STRUCTURE")
				skipped++
				continue
			}

			teamA = fmt.Sprintf("%v", teama["name"])
			teamB = fmt.Sprintf("%v", teamb["name"])
			venue = fmt.Sprintf("%v", venueObj["name"])
			status = normalizeStatus(fmt.Sprintf("%v", m["status_str"]))
			matchID = fmt.Sprintf("%v", m["match_id"])

			startTimeStr := fmt.Sprintf("%v", m["date_start"])

			startTime, err = parseTimeSafe(startTimeStr)
			if err != nil {
				log.Println("❌ ENTITY TIME ERROR:", err)
				skipped++
				continue
			}
		}

		//////////////////////////////////////////////////////////////
		// ✅ CASE 2: CRIC API
		//////////////////////////////////////////////////////////////

		if teams, ok := m["teams"].([]interface{}); ok {

			if len(teams) < 2 {
				log.Println("❌ CRIC INVALID TEAMS")
				skipped++
				continue
			}

			teamA = fmt.Sprintf("%v", teams[0])
			teamB = fmt.Sprintf("%v", teams[1])
			venue = fmt.Sprintf("%v", m["venue"])
			status = normalizeStatus(fmt.Sprintf("%v", m["status"]))
			matchID = fmt.Sprintf("%v", m["id"])

			startTimeStr := fmt.Sprintf("%v", m["dateTimeGMT"])

			startTime, err = parseTimeSafe(startTimeStr)
			if err != nil {
				log.Println("❌ CRIC TIME ERROR:", err)
				skipped++
				continue
			} 
			log.Printf("DEBUG MATCH → A:%s B:%s ID:%s TIME:%v\n",
					   teamA, teamB, matchID, startTime)
		}

		//////////////////////////////////////////////////////////////
		// 🚨 FINAL VALIDATION
		//////////////////////////////////////////////////////////////

		if teamA == "" || teamB == "" || matchID == "" {
			log.Println("❌ INVALID MATCH DATA")
			skipped++
			continue
		}

		//////////////////////////////////////////////////////////////
		// 🔥 DB UPSERT (STRONG)
		//////////////////////////////////////////////////////////////

		res, err := db.ExecContext(ctx, `
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
			failed++
			continue
		}

		rows, _ := res.RowsAffected()

		log.Printf("✅ UPSERT OK | %s vs %s | status=%s | rows=%d\n",
			teamA, teamB, status, rows)

		//////////////////////////////////////////////////////////////
		// 🔥 EVENT QUEUE (OPTIONAL)
		//////////////////////////////////////////////////////////////

		if status == "completed" {
			queue.Enqueue(queue.Job{
				Type: "match_complete",
				Data: matchID,
				Key:  matchID,
			})
		}

		success++
	}

	//////////////////////////////////////////////////////////////
	// 🔥 CACHE INVALIDATION
	//////////////////////////////////////////////////////////////

	cache.Rdb.Del(cache.Ctx, "matches:v1")

	//////////////////////////////////////////////////////////////
	// 📊 FINAL METRICS
	//////////////////////////////////////////////////////////////

	log.Println("===================================")
	log.Println("📊 INGESTION SUMMARY")
	log.Println("✅ Success:", success)
	log.Println("❌ Failed:", failed)
	log.Println("⚠️ Skipped:", skipped)
	log.Println("⏱ Duration:", time.Since(start))
	log.Println("===================================")

	return nil
}

//////////////////////////////////////////////////////////////
// 🚀 WRAPPER
//////////////////////////////////////////////////////////////

func SyncMatchesToDB(db *sql.DB) error {

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	return SyncMatchesToDBWithCtx(ctx, db)
}
