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
