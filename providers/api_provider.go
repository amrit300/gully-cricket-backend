package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

/*
FetchPlayersFromEntityAPI
→ Fetch players for a match
*/

func FetchPlayersFromEntityAPI(matchID string) ([]map[string]interface{}, error) {

	apiKey := os.Getenv("ENTITY_API_KEY")

	url := fmt.Sprintf(
		"https://rest.entitysport.com/v2/matches/%s/squad?token=%s",
		matchID,
		apiKey,
	)

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)

	var raw map[string]interface{}

	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	response := raw["response"].(map[string]interface{})
	teams := response["squads"].([]interface{})

	var players []map[string]interface{}

	for _, t := range teams {
		team := t.(map[string]interface{})
		pList := team["players"].([]interface{})

		for _, p := range pList {
			players = append(players, p.(map[string]interface{}))
		}
	}

	return players, nil
}
