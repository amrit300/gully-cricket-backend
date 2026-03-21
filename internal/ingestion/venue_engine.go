package ingestion

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
	"database/sql"
)

type VenueData struct {
	Venue        string
	TotalRuns    int
	PaceWickets  int
	SpinWickets  int
	Matches      int
}

/*
MAIN FUNCTION
*/
func UpdateVenueStats(db *sql.DB) error {

	apiKey := os.Getenv("ENTITY_API_KEY")

	url := fmt.Sprintf(
		"https://rest.entitysport.com/v2/matches/?token=%s&per_page=100&status=3",
		apiKey,
	)

	client := &http.Client{Timeout: 10 * time.Second}

	res, err := client.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	var raw map[string]interface{}

	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return err
	}

	response := raw["response"].(map[string]interface{})
	items := response["items"].([]interface{})

	venueMap := map[string]*VenueData{}

	for _, m := range items {

		item := m.(map[string]interface{})

		venue := safeString(item["venue"])

		if venue == "" || venue == "Unknown" {
			continue
		}

		if _, ok := venueMap[venue]; !ok {
			venueMap[venue] = &VenueData{Venue: venue}
		}

		v := venueMap[venue]

		// 🔥 SCORE (approx — can improve later)
		totalScore := int(getFloat(item["total_runs"]))
		v.TotalRuns += totalScore

		// 🔥 WICKETS (approx logic)
		p := int(getFloat(item["pace_wkts"]))
		s := int(getFloat(item["spin_wkts"]))

		v.PaceWickets += p
		v.SpinWickets += s
		v.Matches++
	}

	// SAVE TO DB
	for _, v := range venueMap {

		if v.Matches == 0 {
			continue
		}

		avg := v.TotalRuns / v.Matches

		_, err := db.Exec(`
		INSERT INTO venue_stats (venue, avg_score, pace_wickets, spin_wickets, total_matches)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (venue) DO UPDATE SET
		avg_score=$2,
		pace_wickets=$3,
		spin_wickets=$4,
		total_matches=$5,
		last_updated=NOW()
		`,
			v.Venue,
			avg,
			v.PaceWickets,
			v.SpinWickets,
			v.Matches,
		)

		if err != nil {
			fmt.Println("DB ERROR:", err)
		}
	}

	fmt.Println("✅ Venue stats updated")

	return nil
}

/*
HELPERS
*/

func safeString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func getFloat(v interface{}) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}
