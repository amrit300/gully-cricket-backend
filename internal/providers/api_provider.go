package providers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

//////////////////////////////////////////////////////////////
// 🔥 HTTP CLIENT (HARDENED)
//////////////////////////////////////////////////////////////

var client = &http.Client{
	Timeout: 10 * time.Second,
}

//////////////////////////////////////////////////////////////
// 🔥 SAFE HTTP EXECUTOR (REUSABLE)
//////////////////////////////////////////////////////////////

func doRequest(url string) ([]byte, error) {

	res, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("http error: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}

	return body, nil
}

//////////////////////////////////////////////////////////////
// 🟢 ENTITY API (PRIMARY)
//////////////////////////////////////////////////////////////

func FetchMatchesFromEntityAPI() ([]map[string]interface{}, error) {

	apiKey := os.Getenv("ENTITY_API_KEY")

	url := fmt.Sprintf(
		"https://rest.entitysport.com/v2/matches/?status=3&token=%s",
		apiKey,
	)

	body, err := doRequest(url)
	if err != nil {
		return nil, err
	}

	log.Println("🔍 ENTITY RAW:", string(body))

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("entity decode error: %w", err)
	}

	response, ok := raw["response"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("entity invalid response structure")
	}

	items, ok := response["items"].([]interface{})
	if !ok {
		log.Println("⚠️ ENTITY: empty items")
		return []map[string]interface{}{}, nil
	}

	var matches []map[string]interface{}

	for _, item := range items {
		if m, ok := item.(map[string]interface{}); ok {
			matches = append(matches, m)
		}
	}

	return matches, nil
}

//////////////////////////////////////////////////////////////
// 🔵 CRIC API (FALLBACK - HARDENED)
//////////////////////////////////////////////////////////////

func FetchMatchesFromCricAPI() ([]map[string]interface{}, error) {

	apiKey := os.Getenv("CRIC_API_KEY")

	url := fmt.Sprintf(
		"https://api.cricapi.com/v1/currentMatches?apikey=%s",
		apiKey,
	)

	body, err := doRequest(url)
	if err != nil {
		return nil, err
	}

	log.Println("🔍 CRIC RAW:", string(body))

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("cric decode error: %w", err)
	}

	dataRaw, ok := raw["data"]
	if !ok {
		return nil, fmt.Errorf("cric missing data field")
	}

	dataSlice, ok := dataRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("cric invalid data format")
	}

	var matches []map[string]interface{}

	for _, item := range dataSlice {
		if m, ok := item.(map[string]interface{}); ok {
			matches = append(matches, m)
		}
	}

	return matches, nil
}

//////////////////////////////////////////////////////////////
// 🚀 SMART FETCH (PRIMARY → FALLBACK)
//////////////////////////////////////////////////////////////

func FetchMatches() ([]map[string]interface{}, error) {

	// 1️⃣ ENTITY FIRST
	matches, err := FetchMatchesFromEntityAPI()
	if err == nil && len(matches) > 0 {
		log.Println("✅ Using ENTITY API:", len(matches))
		return matches, nil
	}

	log.Println("⚠️ ENTITY failed or empty — switching to CRIC API")

	// 2️⃣ FALLBACK
	fallbackMatches, err := FetchMatchesFromCricAPI()
	if err != nil {
		return nil, fmt.Errorf("fallback failed: %w", err)
	}

	log.Println("✅ Using CRIC API:", len(fallbackMatches))

	return fallbackMatches, nil
}

//////////////////////////////////////////////////////////////
// 👤 PLAYERS (SAFE + HARDENED)
//////////////////////////////////////////////////////////////

func FetchPlayersFromEntityAPI(matchID string) ([]map[string]interface{}, error) {

	apiKey := os.Getenv("ENTITY_API_KEY")

	url := fmt.Sprintf(
		"https://rest.entitysport.com/v2/matches/%s/squads?token=%s",
		matchID,
		apiKey,
	)

	body, err := doRequest(url)
	if err != nil {
		return nil, err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("players decode error: %w", err)
	}

	response, ok := raw["response"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid players response")
	}

	squads, ok := response["squads"].([]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}

	var players []map[string]interface{}

	for _, team := range squads {

		t, ok := team.(map[string]interface{})
		if !ok {
			continue
		}

		pl, ok := t["players"].([]interface{})
		if !ok {
			continue
		}

		for _, p := range pl {
			if playerMap, ok := p.(map[string]interface{}); ok {
				players = append(players, playerMap)
			}
		}
	}

	return players, nil
}
