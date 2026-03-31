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
)

//////////////////////////////////////////////////////////////
// MAIN SYNC FUNCTION
//////////////////////////////////////////////////////////////

func SyncMatchesToDBWithCtx(ctx context.Context, db *sql.DB) error {

	matches, err := providers.FetchMatches()
	if err != nil {
		return err
	}

	log.Println("📦 MATCHES RECEIVED:", len(matches))

	if len(matches) == 0 {
		log.Println("⚠️ NO MATCHES FROM API")
		return nil
	}

	for _, m := range matches {

		var (
			teamA, teamB string
			venue        string
			status       string
			matchID      string
			startTime    time.Time
			parseErr     error
		)

		//////////////////////////////////////////////////////////////
		// ✅ CASE 1: ENTITY API STRUCTURE
		//////////////////////////////////////////////////////////////

		if teama, ok := m["teama"].(map[string]interface{}); ok {

			teamb, okB := m["teamb"].(map[string]interface{})
			venueObj, okV := m["venue"].(map[string]interface{})

			if !okB || !okV {
				log.Println("❌ INVALID ENTITY STRUCTURE")
				continue
			}

			teamA = fmt.Sprintf("%v", teama["name"])
			teamB = fmt.Sprintf("%v", teamb["name"])
			venue = fmt.Sprintf("%v", venueObj["name"])
			matchID = fmt.Sprintf("%v", m["match_id"])

			// 🔥 STATUS NORMALIZATION
			rawStatus := fmt.Sprintf("%v", m["status_str"])
			status = normalizeStatus(rawStatus, false, false)

			// 🔥 TIME PARSE
			startTimeStr := fmt.Sprintf("%v", m["date_start"])
			startTime, parseErr = time.Parse("2006-01-02 15:04:05", startTimeStr)
		}

		//////////////////////////////////////////////////////////////
		// ✅ CASE 2: CRIC API STRUCTURE
		//////////////////////////////////////////////////////////////

		if teams, ok := m["teams"].([]interface{}); ok {

			if len(teams) >= 2 {
				teamA = fmt.Sprintf("%v", teams[0])
				teamB = fmt.Sprintf("%v", teams[1])
			}

			venue = fmt.Sprintf("%v", m["venue"])
			matchID = fmt.Sprintf("%v", m["id"])

			startTimeStr := fmt.Sprintf("%v", m["dateTimeGMT"])
			startTime, parseErr = time.Parse(time.RFC3339, startTimeStr)

			rawStatus := fmt.Sprintf("%v", m["status"])
			matchStarted, _ := m["matchStarted"].(bool)
			matchEnded, _ := m["matchEnded"].(bool)

			status = normalizeStatus(rawStatus, matchStarted, matchEnded)
		}

		//////////////////////////////////////////////////////////////
		// 🚨 VALIDATION (STRICT)
		//////////////////////////////////////////////////////////////

		if teamA == "" || teamB == "" {
			log.Println("❌ SKIPPING INVALID MATCH (NO TEAMS)")
			continue
		}

		if matchID == "" {
			log.Println("❌ SKIPPING INVALID MATCH (NO ID)")
			continue
		}

		if parseErr != nil {
			log.Println("❌ TIME PARSE ERROR:", parseErr)
			continue
		}

		//////////////////////////////////////////////////////////////
		// 🔥 UPSERT INTO DB
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

		log.Printf("✅ SYNCED: %s vs %s [%s]", teamA, teamB, status)
	}

	//////////////////////////////////////////////////////////////
	// 🔥 CACHE INVALIDATION
	//////////////////////////////////////////////////////////////

	if cache.Rdb != nil {
		cache.Rdb.Del(cache.Ctx, "matches:v1")
	}

	log.Println("✅ MATCH SYNC COMPLETED")

	return nil
}

//////////////////////////////////////////////////////////////
// FALLBACK WRAPPER
//////////////////////////////////////////////////////////////

func SyncMatchesToDB(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return SyncMatchesToDBWithCtx(ctx, db)
}

//////////////////////////////////////////////////////////////
// 🔥 STATUS NORMALIZATION (CORE LOGIC)
//////////////////////////////////////////////////////////////

func normalizeStatus(raw string, started bool, ended bool) string {

	raw = strings.ToLower(raw)

	// ✅ COMPLETED
	if ended ||
		strings.Contains(raw, "won") ||
		strings.Contains(raw, "match ended") ||
		strings.Contains(raw, "result") {
		return "completed"
	}

	// ✅ LIVE
	if started {
		return "live"
	}

	// ✅ UPCOMING
	return "upcoming"
}
