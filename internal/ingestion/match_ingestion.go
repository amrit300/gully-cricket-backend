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

func SyncMatchesToDB(db *sql.DB) error {

	apiKey := os.Getenv("ENTITY_API_KEY")

	if apiKey == "" {
		return fmt.Errorf("missing ENTITY_API_KEY")
	}

	url := fmt.Sprintf(
		"https://rest.entitysport.com/v2/matches/?token=%s&per_page=50",
		apiKey,
	)

	client := &http.Client{Timeout: 8 * time.Second}

	res, err := client.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)

	fmt.Println("ENTITY API RAW:", string(body))

	if res.StatusCode != 200 {
		return fmt.Errorf("API failed: %s", string(body))
	}

	var raw map[string]interface{}

	if err := json.Unmarshal(body, &raw); err != nil {
		return err
	}

	response, ok := raw["response"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid response structure")
	}

	items, ok := response["items"].([]interface{})
	if !ok {
		return fmt.Errorf("no matches found")
	}

	now := time.Now().UTC()

	for _, m := range items {

		item := m.(map[string]interface{})

		status := fmt.Sprintf("%v", item["status"])

		// keep only live/upcoming
		if status != "1" && status != "2" {
			continue
		}

		dateStr := fmt.Sprintf("%v", item["date_start"])

		matchTime, err := time.Parse("2006-01-02 15:04:05", dateStr)
		if err != nil {
			continue
		}

		matchTime = matchTime.UTC()

		// ignore very old matches
		if matchTime.Before(now.Add(-6 * time.Hour)) {
			continue
		}

		teama := item["teama"].(map[string]interface{})
		teamb := item["teamb"].(map[string]interface{})

		teamA := fmt.Sprintf("%v", teama["name"])
		teamB := fmt.Sprintf("%v", teamb["name"])
		venue := fmt.Sprintf("%v", item["venue"])
		externalID := fmt.Sprintf("%v", item["match_id"])

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

		fmt.Println("INSERTED:", teamA, "vs", teamB)
	}

	return nil
}
