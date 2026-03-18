package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

/* =========================
   STRUCT
========================= */

type Match struct {
	ID         int    `json:"id"`
	TeamA      string `json:"teamA"`
	TeamB      string `json:"teamB"`
	Venue      string `json:"venue"`
	AvgScore   int    `json:"avgScore"`
	SpinAssist int    `json:"spinAssist"`
	PaceAssist int    `json:"paceAssist"`
	StartTime  string `json:"startTime"`
}

/* =========================
   MAIN ENTRY
========================= */

func GetMatches() ([]Match, error) {

	matches, err := GetMatchesFromAPI()
	if err == nil && len(matches) > 0 {
		return matches, nil
	}

	return GetMatchesFromScraper()
}

/* =========================
   ENTITY API
========================= */

func GetMatchesFromAPI() ([]Match, error) {

	apiKey := os.Getenv("ENTITY_API_KEY")

	if apiKey == "" {
		return nil, fmt.Errorf("missing ENTITY_API_KEY")
	}

	url := fmt.Sprintf(
		"https://rest.entitysport.com/v2/matches/?token=%s&status=1",
		apiKey,
	)

	client := &http.Client{Timeout: 5 * time.Second}

	res, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var raw map[string]interface{}

	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return nil, err
	}

	response, ok := raw["response"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response")
	}

	var matches []Match

	for i, m := range response {

		item := m.(map[string]interface{})

		teama := item["teama"].(map[string]interface{})
		teamb := item["teamb"].(map[string]interface{})

		matches = append(matches, Match{
			ID:         i + 1,
			TeamA:      safeString(teama["name"]),
			TeamB:      safeString(teamb["name"]),
			Venue:      safeString(item["venue"]),
			AvgScore:   160,
			SpinAssist: 40,
			PaceAssist: 60,
			StartTime:  safeString(item["date_start"]),
		})
	}

	return matches, nil
}

/* =========================
   SCRAPER FALLBACK
========================= */

func GetMatchesFromScraper() ([]Match, error) {

	return []Match{
		{
			ID:         999,
			TeamA:      "Fallback XI",
			TeamB:      "Scraper XI",
			Venue:      "Fallback Ground",
			AvgScore:   150,
			SpinAssist: 50,
			PaceAssist: 50,
			StartTime:  time.Now().Format(time.RFC3339),
		},
	}, nil
}

/* =========================
   SAFE HELPER
========================= */

func safeString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return "Unknown"
}
