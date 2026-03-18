package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

/* =========================
   DOMAIN MODEL
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
	Status     string `json:"status"`
}

/* =========================
   ENTRY POINT
========================= */

func GetMatches() ([]Match, error) {

	// PRIMARY
	matches, err := fetchFromEntityAPI()
	if err == nil && len(matches) > 0 {
		return matches, nil
	}

	fmt.Println("⚠️ Entity API failed:", err)

	// SECONDARY
	matches, err = fetchFromCricAPI()
	if err == nil && len(matches) > 0 {
		return matches, nil
	}

	fmt.Println("❌ Secondary API also failed:", err)

	// FINAL: return empty (NO FAKE DATA)
	return []Match{}, nil
}

/* =========================
   ENTITY API
========================= */

func fetchFromEntityAPI() ([]Match, error) {

	apiKey := os.Getenv("ENTITY_API_KEY")

	url := fmt.Sprintf(
		"https://rest.entitysport.com/v2/matches/?token=%s&per_page=50",
		apiKey,
	)

	client := &http.Client{Timeout: 6 * time.Second}

	res, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var raw map[string]interface{}

	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return nil, err
	}

	response := raw["response"].(map[string]interface{})
	items := response["items"].([]interface{})

	var matches []Match

	now := time.Now()
	maxTime := now.Add(7 * 24 * time.Hour)

	for i, m := range items {
for i, m := range items {

	item := m.(map[string]interface{})

	status := safeString(item["status"])

	if status != "1" && status != "2" {
		continue
	}

	matchTimeStr := safeString(item["date_start"])

	matchTime, err := time.Parse("2006-01-02 15:04:05", matchTimeStr)
	if err != nil {
		continue
	}

	now := time.Now().UTC()
	maxTime := now.Add(7 * 24 * time.Hour)

	// ✅ DEBUG LOG (prevents unused variable error)
	fmt.Println("MATCH TIME:", matchTime, "NOW:", now, "MAX:", maxTime)

	// ❌ TEMP DISABLED FILTER
	// if matchTime.Before(now) || matchTime.After(maxTime) {
	//     continue
	// }

	teama := safeMap(item["teama"])
	teamb := safeMap(item["teamb"])

	matches = append(matches, Match{
		ID:         i + 1,
		TeamA:      safeString(teama["name"]),
		TeamB:      safeString(teamb["name"]),
		Venue:      safeString(item["venue"]),
		AvgScore:   160,
		SpinAssist: 40,
		PaceAssist: 60,
		StartTime:  matchTimeStr,
		Status:     status,
	})
}

	return matches, nil
}

/* =========================
   SECONDARY API (CRICAPI)
========================= */

func fetchFromCricAPI() ([]Match, error) {

	apiKey := os.Getenv("CRIC_API_KEY")

	if apiKey == "" {
		return nil, fmt.Errorf("missing CRIC_API_KEY")
	}

	url := fmt.Sprintf(
		"https://api.cricapi.com/v1/currentMatches?apikey=%s",
		apiKey,
	)

	client := &http.Client{Timeout: 6 * time.Second}

	res, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var raw map[string]interface{}

	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return nil, err
	}

	data, ok := raw["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid cricapi response")
	}

	var matches []Match

	now := time.Now()
	maxTime := now.Add(7 * 24 * time.Hour)

	for i, m := range data {

		item := m.(map[string]interface{})

		teams, ok := item["teams"].([]interface{})
		if !ok || len(teams) < 2 {
			continue
		}

		matchTimeStr := safeString(item["dateTimeGMT"])

		matchTime, err := time.Parse(time.RFC3339, matchTimeStr)
		if err != nil {
			continue
		}

		// TEMP DEBUG — disable filtering

		//if matchTime.Before(now) || matchTime.After(maxTime) {
			//continue
		// }

		matches = append(matches, Match{
			ID:         i + 1000,
			TeamA:      safeString(teams[0]),
			TeamB:      safeString(teams[1]),
			Venue:      safeString(item["venue"]),
			AvgScore:   150,
			SpinAssist: 50,
			PaceAssist: 50,
			StartTime:  matchTimeStr,
			Status:     "secondary",
		})
	}

	return matches, nil
}

/* =========================
   HELPERS
========================= */

func safeString(v interface{}) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return "Unknown"
}

func safeMap(v interface{}) map[string]interface{} {
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}
