package workers

import (
	"database/sql"
	"log"

	dbutil "gully-cricket/internal/db"
	"gully-cricket/internal/services"
)

func ProcessCompletedMatches(db *sql.DB) {

	ctx, cancel := dbutil.Ctx()
defer cancel()

rows, err := db.QueryContext(ctx, `
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

	if err := rows.Scan(&contestID, &status); err != nil {
		log.Println(err)
		continue
	}

	if status == "Completed" {
		if err := services.ProcessContestPayout(db, contestID); err != nil {
			log.Println("payout error:", err)
		}
	}
}

if err := rows.Err(); err != nil {
	log.Println("rows error:", err)
}
}
