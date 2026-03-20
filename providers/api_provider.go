package main

import (
	"encoding/json"
	"net/http"
)

func GetMatchesFromAPI() ([]Match, error) {

	req, _ := http.NewRequest("GET", "https://api.cricapi.com/v1/currentMatches?apikey=YOUR_API_KEY", nil)

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw map[string]interface{}

	json.NewDecoder(resp.Body).Decode(&raw)

	data := raw["data"].([]interface{})

	var matches []Match

	for _, m := range data {

		match := m.(map[string]interface{})

		matches = append(matches, Match{
			ID:        match["id"].(string),
			TeamA:     match["teams"].([]interface{})[0].(string),
			TeamB:     match["teams"].([]interface{})[1].(string),
			StartTime: match["dateTimeGMT"].(string),
			Status:    match["status"].(string),
		})
	}

	return matches, nil
	
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

