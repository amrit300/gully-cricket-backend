package ingestion

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

/*
SyncMatchesToDB
→ Fetch matches from EntitySport API
→ Parse safely
→ Insert/Upsert into DB
*/

func SyncMatchesToDB(db *sql.DB) error {

	apiKey := os.Getenv("ENTITY_API_KEY")

	if apiKey == "" {
		return fmt.Errorf("ENTITY_API_KEY missing")
	}

	url := fmt.Sprintf(
		"https://rest.entitysport.com/v2/matches/?token=%s",
		apiKey,
	)

	client := &http.Client{Timeout: 10 * time.Second}

	res, err := client.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	/* =========================
	   READ RAW RESPONSE
	========================= */

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	fmt.Println("API STATUS:", res.StatusCode)

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("API failed: %s", string(body))
	}

	/* =========================
	   PARSE JSON
	========================= */

	var raw map[string]interface{}

	if err := json.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("JSON parse error: %v", err)
	}

	response, ok := raw["response"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid response structure")
	}

	items, ok := response["items"].([]interface{})
	if !ok {
		return fmt.Errorf("missing items array")
	}

	fmt.Println("TOTAL MATCHES RECEIVED:", len(items))

	/* =========================
	   LOOP + INSERT
	========================= */

	for _, m := range items {

		item := m.(map[string]interface{})

		// ✅ SAFE EXTRACTION

		teama, okA := item["teama"].(map[string]interface{})
		teamb, okB := item["teamb"].(map[string]interface{})

		if !okA || !okB {
			fmt.Println("TEAM PARSE FAILED")
			continue
		}

		teamA := fmt.Sprintf("%v", teama["name"])
		teamB := fmt.Sprintf("%v", teamb["name"])

		matchID := fmt.Sprintf("%v", item["match_id"])
		status := fmt.Sprintf("%v", item["status_str"])

		dateStr := fmt.Sprintf("%v", item["date_start"])

		matchTime, err := time.Parse("2006-01-02 15:04:05", dateStr)
		if err != nil {
			fmt.Println("TIME PARSE ERROR:", err)
			continue
		}

		// ✅ VENUE SAFE PARSE
		venue := "Unknown"
		if v, ok := item["venue"].(map[string]interface{}); ok {
			venue = fmt.Sprintf("%v", v["name"])
		}

		/* =========================
		   UPSERT INTO DB
		========================= */

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
			matchID,
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

		fmt.Println("INSERTED:", teamA, "vs", teamB)
	}

	return nil
}
