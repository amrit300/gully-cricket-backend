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
// HTTP CLIENT (REUSE)
//////////////////////////////////////////////////////////////

var client = &http.Client{
	Timeout: 10 * time.Second,
}

//////////////////////////////////////////////////////////////
// ENTITY API (PRIMARY)
//////////////////////////////////////////////////////////////

func FetchMatchesFromEntityAPI() ([]map[string]interface{}, error) {

	apiKey := os.Getenv("ENTITY_API_KEY")

	url := fmt.Sprintf(
		"https://rest.entitysport.com/v2/matches/?status=3&token=%s",
		apiKey,
	)

	res, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	log.Println("🔍 ENTITY RAW:", string(body))

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	response, ok := raw["response"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response structure")
	}

	items, ok := response["items"].([]interface{})
	if !ok {
		log.Println("⚠️ ENTITY: no items found")
		return []map[string]interface{}{}, nil
	}

	var matches []map[string]interface{}

	for _, m := range items {
		if matchMap, ok := m.(map[string]interface{}); ok {
			matches = append(matches, matchMap)
		}
	}

	return matches, nil
}

//////////////////////////////////////////////////////////////
// FALLBACK API (CRIC API)
//////////////////////////////////////////////////////////////

func FetchMatchesFromCricAPI() ([]map[string]interface{}, error) {

	apiKey := os.Getenv("CRIC_API_KEY")

	url := fmt.Sprintf(
		"https://api.cricapi.com/v1/currentMatches?apikey=%s",
		apiKey,
	)

	res, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var raw map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	// 🔥 THIS IS THE FIX
	data, ok := raw["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid API structure: data missing")
	}

	var matches []map[string]interface{}

	for _, item := range data {
		if m, ok := item.(map[string]interface{}); ok {
			matches = append(matches, m)
		}
	}

	return matches, nil
}
//////////////////////////////////////////////////////////////
// SMART FETCH (PRIMARY + FALLBACK)
//////////////////////////////////////////////////////////////

func FetchMatches() ([]map[string]interface{}, error) {

	//////////////////////////////////////////////////////////////
	// 1. TRY ENTITY API
	//////////////////////////////////////////////////////////////

	matches, err := FetchMatchesFromEntityAPI()
	if err == nil && len(matches) > 0 {
		log.Println("✅ Using ENTITY API:", len(matches))
		return matches, nil
	}

	log.Println("⚠️ ENTITY failed or empty — switching to CRIC API")

	//////////////////////////////////////////////////////////////
	// 2. FALLBACK → CRIC API
	//////////////////////////////////////////////////////////////

	fallbackMatches, err := FetchMatchesFromCricAPI()
	if err != nil {
		return nil, err
	}

	log.Println("✅ Using CRIC API:", len(fallbackMatches))

	return fallbackMatches, nil
}

//////////////////////////////////////////////////////////////
// PLAYERS (SAFE VERSION)
//////////////////////////////////////////////////////////////

func FetchPlayersFromEntityAPI(matchID string) ([]map[string]interface{}, error) {

	apiKey := os.Getenv("ENTITY_API_KEY")

	url := fmt.Sprintf(
		"https://rest.entitysport.com/v2/matches/%s/squads?token=%s",
		matchID,
		apiKey,
	)

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
		return nil, fmt.Errorf("invalid response")
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
