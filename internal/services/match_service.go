package services

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"gully-cricket/internal/cache"
	dbutil "gully-cricket/internal/db"
)

func GetMatches(db *sql.DB) (map[string]interface{}, error) {

	//////////////////////////////////////////////////////////////
	// 1. REDIS CACHE
	//////////////////////////////////////////////////////////////

	cached, err := cache.Rdb.Get(cache.Ctx, "matches:v1").Result()
	if err == nil {
		var data map[string]interface{}
		json.Unmarshal([]byte(cached), &data)
		return data, nil
	}

	//////////////////////////////////////////////////////////////
	// 2. DB QUERY (SAFE WITH TIMEOUT)
	//////////////////////////////////////////////////////////////

	ctx, cancel := dbutil.Ctx()
	defer cancel()

	rows, err := db.QueryContext(ctx, `
		SELECT team_a, team_b, start_time, status, venue
		FROM matches_master
		ORDER BY start_time DESC
		LIMIT 50
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	live := []map[string]interface{}{}
	upcoming := []map[string]interface{}{}
	recent := []map[string]interface{}{}

	for rows.Next() {

		var teamA, teamB, status, venue string
		var startTime string

		if err := rows.Scan(&teamA, &teamB, &startTime, &status, &venue); err != nil {
			return nil, err
		}

		match := map[string]interface{}{
			"teamA":     teamA,
			"teamB":     teamB,
			"startTime": startTime,
			"status":    status,
			"venue":     venue,
		}

		if strings.Contains(status, "Live") || strings.Contains(status, "Stumps") {
			live = append(live, match)
		} else if strings.Contains(status, "Starts") || strings.Contains(status, "Upcoming") {
			upcoming = append(upcoming, match)
		} else {
			recent = append(recent, match)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// fallback logic
	if len(live) == 0 && len(upcoming) == 0 {
		live = recent
	}

	result := map[string]interface{}{
		"live":     live,
		"upcoming": upcoming,
		"recent":   recent,
	}

	//////////////////////////////////////////////////////////////
	// 3. CACHE RESULT
	//////////////////////////////////////////////////////////////

	bytes, _ := json.Marshal(result)
	cache.Rdb.Set(cache.Ctx, "matches:v1", bytes, 20*time.Second)

	return result, nil
}
