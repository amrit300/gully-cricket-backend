package workers

import (
	"database/sql"
	"log"

	"gully-cricket/internal/services"
)

func ProcessCompletedMatches(db *sql.DB) {

	rows, err := db.Query(`
		SELECT c.id, m.status
		FROM contests c
		JOIN matches_master m ON c.match_id = m.id
		WHERE c.status != 'completed'
	`)

	if err != nil {
		log.Println(err)
		return
	}
	defer rows.Close()

	for rows.Next() {

		var contestID int
		var status string

		rows.Scan(&contestID, &status)

		if status == "Completed" {
			services.ProcessContestPayout(db, contestID)
		}
	}
}
