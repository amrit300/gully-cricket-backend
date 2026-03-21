package ingestion

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

/* =========================
   ENTRY POINT
========================= */

func SyncMatchesToDB(db *sql.DB) error {

	apiKey := os.Getenv("CRIC_API_KEY")

	url := fmt.Sprintf(
		"https://api.cricapi.com/v1/currentMatches?apikey=%s",
		apiKey,
	)

	client := &http.Client{Timeout: 8 * time.Second}

	res, err := client.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	var raw map[string]interface{}

	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return err
	}

	data, ok := raw["data"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid response format")
	}

	now := time.Now().UTC()
	pastLimit := now.Add(-3 * time.Hour)
	futureLimit := now.Add(7 * 24 * time.Hour)

	for _, m := range data {

		item := m.(map[string]interface{})

		teams, ok := item["teams"].([]interface{})
		if !ok || len(teams) < 2 {
			continue
		}

		matchTimeStr := fmt.Sprintf("%v", item["dateTimeGMT"])

		matchTime, err := time.Parse(time.RFC3339, matchTimeStr)
		if err != nil {
			continue
		}

		matchTime = matchTime.UTC()

		// ✅ TIME FILTER
		if matchTime.Before(pastLimit) || matchTime.After(futureLimit) {
			continue
		}

		teamA := fmt.Sprintf("%v", teams[0])
		teamB := fmt.Sprintf("%v", teams[1])
		venue := fmt.Sprintf("%v", item["venue"])
		externalID := fmt.Sprintf("%v", item["id"])
		status := fmt.Sprintf("%v", item["status"])

		// ✅ UPSERT INTO DB
		_, err = db.Exec(`
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
			externalID,
			teamA,
			teamB,
			venue,
			matchTime,
			status,
		)

		if err != nil {
			fmt.Println("INSERT ERROR:", err)
			continue
		}

		fmt.Println("MATCH STORED:", teamA, "vs", teamB)
	}

	return nil
}
