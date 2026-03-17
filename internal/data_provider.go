package main

import (
	"log"
	"sync"
	"time"
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

func GetMatchesData() ([]Match, error) {

	matchCache.Mutex.Lock()
	defer matchCache.Mutex.Unlock()

	// CACHE VALIDITY → 60 sec
	if time.Since(matchCache.Timestamp) < 60*time.Second && len(matchCache.Data) > 0 {
		log.Println("Serving matches from cache")
		return matchCache.Data, nil
	}

	// 1️⃣ TRY API
	data, err := GetMatchesFromAPI()

	if err != nil || len(data) == 0 {
		log.Println("API failed → fallback to scraper")

		// 2️⃣ FALLBACK SCRAPER
		data, err = GetMatchesFromScraper()

		if err != nil {
			return nil, err
		}
	}

	// 3️⃣ SAVE CACHE
	matchCache.Data = data
	matchCache.Timestamp = time.Now()

	return data, nil
}
