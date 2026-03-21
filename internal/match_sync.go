package ingestion

import (
	"database/sql"
	"gully-cricket/internal"
)

func SyncMatchesToDB(db *sql.DB) error {
	return internal.SyncMatches(db)
}
