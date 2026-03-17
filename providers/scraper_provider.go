package main

import (
	"github.com/gocolly/colly"
)

func GetMatchesFromScraper() ([]Match, error) {

	var matches []Match

	c := colly.NewCollector()

	c.OnHTML(".match-card", func(e *colly.HTMLElement) {

		match := Match{
			ID:        e.ChildText(".match-id"),
			TeamA:     e.ChildText(".team-a"),
			TeamB:     e.ChildText(".team-b"),
			StartTime: e.ChildText(".time"),
			Status:    e.ChildText(".status"),
		}

		matches = append(matches, match)
	})

	c.Visit("https://www.cricbuzz.com/cricket-match/live-scores")

	return matches, nil
}
