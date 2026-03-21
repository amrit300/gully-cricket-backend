package providers

import (
	"io/ioutil"
	"net/http"
	"strings"
)

func GetMatchesFromScraper() ([]Match, error) {

	resp, err := http.Get("https://www.cricbuzz.com/cricket-match/live-scores")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	html := string(body)

	var matches []Match

	// VERY BASIC extraction (safe fallback)
	if strings.Contains(html, "India") && strings.Contains(html, "Australia") {

		matches = append(matches, Match{
			ID:        "scrape_1",
			TeamA:     "India",
			TeamB:     "Australia",
			StartTime: "Live",
			Status:    "Live",
		})
	}

	return matches, nil
}
