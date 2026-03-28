package services

import (
	"database/sql"
	"encoding/json"
	"log"
	"strings"
	"time"

	"gully-cricket/internal/cache"
	dbutil "gully-cricket/internal/db"
)

func GetMatches(db *sql.DB) (map[string]interface{}, error) {

	cacheKey := "matches:v1"

	//////////////////////////////////////////////////////////////
	// 1. CACHE HIT (FAST PATH)
	//////////////////////////////////////////////////////////////

	cached, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
	if err == nil {
		var data map[string]interface{}
		if json.Unmarshal([]byte(cached), &data) == nil {
			return data, nil
		}
	}

	//////////////////////////////////////////////////////////////
	// 2. CACHE STAMPEDE PROTECTION (LIGHT LOCK)
	//////////////////////////////////////////////////////////////

	lockKey := cacheKey + ":lock"

	locked, _ := cache.Rdb.SetNX(cache.Ctx, lockKey, "1", 5*time.Second).Result()

	// If another request is building cache → wait briefly & retry cache
	if !locked {
		time.Sleep(100 * time.Millisecond)

		cached, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
		if err == nil {
			var data map[string]interface{}
			if json.Unmarshal([]byte(cached), &data) == nil {
				return data, nil
			}
		}
	}

	//////////////////////////////////////////////////////////////
	// 3. DB QUERY (SAFE WITH TIMEOUT)
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
			log.Println("SCAN ERROR:", err)
			continue
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

	//////////////////////////////////////////////////////////////
	// 4. FALLBACK LOGIC (SAFE)
	//////////////////////////////////////////////////////////////

	if len(live) == 0 && len(upcoming) == 0 {
		live = recent
	}

	result := map[string]interface{}{
		"live":     live,
		"upcoming": upcoming,
		"recent":   recent,
	}

	//////////////////////////////////////////////////////////////
	// 5. CACHE STORE (WITH TTL)
	//////////////////////////////////////////////////////////////

	bytes, err := json.Marshal(result)
	if err == nil {
		cache.Rdb.Set(cache.Ctx, cacheKey, bytes, 20*time.Second)
	}

	// release lock
	cache.Rdb.Del(cache.Ctx, lockKey)

	return result, nil
}
