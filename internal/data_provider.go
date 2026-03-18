package internal

import (
	"log"
	"sync"
	"time"
	"os"
	"fmt"
)

type Match struct {
	ID        string `json:"id"`
	TeamA     string `json:"teamA"`
	TeamB     string `json:"teamB"`
	StartTime string `json:"startTime"`
	Status    string `json:"status"`
}

var matchCache struct {
	Data      []Match
	Timestamp time.Time
	Mutex     sync.Mutex
}
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
		return nil, fmt.Errorf("invalid response format")
	}

	var matches []Match

	for i, m := range response {

		item := m.(map[string]interface{})

		teama := item["teama"].(map[string]interface{})
		teamb := item["teamb"].(map[string]interface{})

		matches = append(matches, Match{
			ID:          i + 1,
			TeamA:       toString(teama["name"]),
			TeamB:       toString(teamb["name"]),
			Venue:       toString(item["venue"]),
			AvgScore:    160,
			SpinAssist:  40,
			PaceAssist:  60,
			StartTime:   toString(item["date_start"]),
		})
	}

	return matches, nil
}
