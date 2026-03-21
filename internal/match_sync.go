package internal

import (
	"database/sql"
	"log"
)

/*
SyncMatches:
Fetch matches from APIs → store in DB
*/

func SyncMatches(db *sql.DB) error {

	matches, err := GetMatches(db)
	if err != nil {
		return err
	}

	for _, m := range matches {

		_, err := db.Exec(`
		INSERT INTO matches_master 
		(team_a, team_b, venue, start_time, status)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT DO NOTHING
		`,
			m.TeamA,
			m.TeamB,
			m.Venue,
			m.StartTime,
			m.Status,
		)

		if err != nil {
			log.Println("match insert error:", err)
		}
	}

	return nil
}
