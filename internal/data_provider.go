package internal

import (
	"database/sql"
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

func GetMatches(db *sql.DB) ([]Match, error) {

	// PRIMARY → EntitySports
	matches, err := fetchFromEntityAPI(db)
	if err == nil && len(matches) > 0 {
		return matches, nil
	}

	fmt.Println("⚠️ Entity API failed:", err)

	// SECONDARY → CricAPI
	matches, err = fetchFromCricAPI(db)
	if err == nil && len(matches) > 0 {
		return matches, nil
	}

	fmt.Println("❌ All sources failed")

	return []Match{}, nil
}

/* =========================
   ENTITY API
========================= */

func fetchFromEntityAPI(db *sql.DB) ([]Match, error) {

	apiKey := os.Getenv("ENTITY_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("missing ENTITY_API_KEY")
	}

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

	response, ok := raw["response"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	items, ok := response["items"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("no items found")
	}

	var matches []Match

	now := time.Now().UTC()
	pastLimit := now.Add(-3 * time.Hour)
	futureLimit := now.Add(7 * 24 * time.Hour)

	for i, m := range items {

		item, ok := m.(map[string]interface{})
		if !ok {
			continue
		}

		// ✅ STATUS FILTER (LIVE + UPCOMING)
		status := safeString(item["status"])
		if status == "" {
			continue
		}

		matchTimeStr := safeString(item["date_start"])

		// ✅ SAFE TIME PARSE
		matchTime, err := time.Parse("2006-01-02 15:04:05", matchTimeStr)
		if err != nil {
			matchTime, err = time.Parse(time.RFC3339, matchTimeStr)
			if err != nil {
				continue
			}
		}

		matchTime = matchTime.UTC()

		// ✅ TIME FILTER
		if matchTime.Before(pastLimit) || matchTime.After(futureLimit) {
			continue
		}

		teama := safeMap(item["teama"])
		teamb := safeMap(item["teamb"])

		venue := safeString(item["venue"])
		if venue == "Unknown" {
			venue = "Default Stadium"
		}

		// ✅ VENUE INTELLIGENCE
		avg, spin, pace := getVenueStats(db, venue)

		matches = append(matches, Match{
			ID:         i + 1,
			TeamA:      safeString(teama["name"]),
			TeamB:      safeString(teamb["name"]),
			Venue:      venue,
			AvgScore:   avg,
			SpinAssist: spin,
			PaceAssist: pace,
			StartTime:  matchTimeStr,
			Status:     status,
		})
	}

	return matches, nil
}

/* =========================
   CRICAPI (SECONDARY)
========================= */

func fetchFromCricAPI(db *sql.DB) ([]Match, error) {

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

	now := time.Now().UTC()
	pastLimit := now.Add(-3 * time.Hour)
	futureLimit := now.Add(7 * 24 * time.Hour)

	for i, m := range data {

		item, ok := m.(map[string]interface{})
		if !ok {
			continue
		}

		teams, ok := item["teams"].([]interface{})
		if !ok || len(teams) < 2 {
			continue
		}

		matchTimeStr := safeString(item["dateTimeGMT"])

		matchTime, err := time.Parse(time.RFC3339, matchTimeStr)
		if err != nil {
			continue
		}

		matchTime = matchTime.UTC()

		// ✅ TIME FILTER
		if matchTime.Before(pastLimit) || matchTime.After(futureLimit) {
			continue
		}

		venue := safeString(item["venue"])
		if venue == "Unknown" {
			venue = "Default Stadium"
		}

		avg, spin, pace := getVenueStats(db, venue)

		matches = append(matches, Match{
			ID:         i + 1000,
			TeamA:      safeString(teams[0]),
			TeamB:      safeString(teams[1]),
			Venue:      venue,
			AvgScore:   avg,
			SpinAssist: spin,
			PaceAssist: pace,
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

func FetchPlayersFromCricAPI(matchID string) ([]map[string]interface{}, error) {

	apiKey := os.Getenv("CRIC_API_KEY")

	url := fmt.Sprintf(
		"https://api.cricapi.com/v1/match_squad?apikey=%s&id=%s",
		apiKey,
		matchID,
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

	data, ok := raw["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid squad data")
	}

	players, ok := data["players"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("no players found")
	}

	var result []map[string]interface{}

	for _, p := range players {
		player := p.(map[string]interface{})
		result = append(result, player)
	}

	return result, nil
}

/* =========================
   VENUE STATS
========================= */

func getVenueStats(db *sql.DB, venue string) (int, int, int) {

	row := db.QueryRow(`
	SELECT 
		COALESCE(avg_score,150),
		COALESCE((spin_wickets*100/NULLIF(spin_wickets+pace_wickets,0)),50),
		COALESCE((pace_wickets*100/NULLIF(spin_wickets+pace_wickets,0)),50)
	FROM venue_stats
	WHERE venue=$1
	`, venue)

	var avg, spin, pace int

	err := row.Scan(&avg, &spin, &pace)
	if err != nil {
		return 150, 50, 50
	}

	return avg, spin, pace
}
