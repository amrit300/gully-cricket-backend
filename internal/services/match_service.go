package services

import (
	"database/sql"
	"encoding/json"
	"time"

	"gully-cricket/internal/cache"
)

func GetMatches(db *sql.DB) ([]map[string]interface{}, error) {

	cached, err := cache.Rdb.Get(cache.Ctx, "matches").Result()
	if err == nil {
		var data []map[string]interface{}
		json.Unmarshal([]byte(cached), &data)
		return data, nil
	}

	rows, err := db.Query(`
		SELECT id, team1, team2, status, start_time
		FROM matches_master
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []map[string]interface{}

	for rows.Next() {
		var id int
		var t1, t2, status string
		var start string

		rows.Scan(&id, &t1, &t2, &status, &start)

		matches = append(matches, map[string]interface{}{
			"id": id,
			"team1": t1,
			"team2": t2,
			"status": status,
			"start_time": start,
		})
	}

	bytes, _ := json.Marshal(matches)
	cache.Rdb.Set(cache.Ctx, "matches", bytes, 30*time.Second)

	return matches, nil
}
