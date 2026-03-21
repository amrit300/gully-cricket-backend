package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func FetchMatchesFromEntityAPI() ([]map[string]interface{}, error) {

	apiKey := os.Getenv("ENTITY_API_KEY")

	url := fmt.Sprintf(
	"https://rest.entitysport.com/v2/matches/?status=3&token=%s",
	apiKey,
	)

	res, err := http.Get(url)
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

	var matches []map[string]interface{}

	for _, m := range items {
		matches = append(matches, m.(map[string]interface{}))
	}

	return matches, nil
}

func FetchPlayersFromEntityAPI(matchID string) ([]map[string]interface{}, error) {

	apiKey := os.Getenv("ENTITY_API_KEY")

	url := fmt.Sprintf(
		"https://rest.entitysport.com/v2/matches/%s/squads?token=%s",
		matchID,
		apiKey,
	)

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var raw map[string]interface{}

	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return nil, err
	}

	response := raw["response"].(map[string]interface{})
	squads := response["squads"].([]interface{})

	var players []map[string]interface{}

	for _, team := range squads {
		t := team.(map[string]interface{})
		pl := t["players"].([]interface{})

		for _, p := range pl {
			players = append(players, p.(map[string]interface{}))
		}
	}

	return players, nil
}
