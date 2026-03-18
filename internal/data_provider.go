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

	// PRIMARY → EntitySports
	matches, err := fetchFromEntityAPI()
	if err == nil && len(matches) > 0 {
		return matches, nil
	}

	fmt.Println("⚠️ Entity API failed:", err)

	// SECONDARY → CricAPI
	matches, err = fetchFromCricAPI()
	if err == nil && len(matches) > 0 {
		return matches, nil
	}

	fmt.Println("❌ All sources failed")

	return []Match{}, nil
}

/* =========================
   ENTITY API
========================= */

func fetchFromEntityAPI() ([]Match, error) {

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

	for i, m := range items {

		item, ok := m.(map[string]interface{})
		if !ok {
			continue
		}

		status := safeString(item["status"])

		// only upcoming or live
		if status != "1" && status != "2" {
			continue
		}

		matchTimeStr := safeString(item["date_start"])

		// try multiple formats
		matchTime, err := time.Parse("2006-01-02 15:04:05", matchTimeStr)
		if err != nil {
			matchTime, err = time.Parse(time.RFC3339, matchTimeStr)
			if err != nil {
				continue
			}
		}

		now := time.Now().UTC()
		maxTime := now.Add(7 * 24 * time.Hour)

		// DEBUG (safe usage)
		fmt.Println("MATCH:", matchTime, "NOW:", now)

		// TEMP: disable strict filtering for now
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
   CRICAPI (SECONDARY)
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

	for i, m := range data {

		item, ok := m.(map[string]interface{})
		if !ok {
			continue
		}

		teams, ok := item["teams"].([]interface{})
		if !ok || len(teams) < 2 {
			continue
		}

		matches = append(matches, Match{
			ID:         i + 1000,
			TeamA:      safeString(teams[0]),
			TeamB:      safeString(teams[1]),
			Venue:      safeString(item["venue"]),
			AvgScore:   150,
			SpinAssist: 50,
			PaceAssist: 50,
			StartTime:  safeString(item["dateTimeGMT"]),
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
