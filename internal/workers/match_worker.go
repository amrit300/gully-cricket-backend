package workers

import (
	"database/sql"
	"log"

	dbutil "gully-cricket/internal/db"
	"gully-cricket/internal/services"
)

var DB *sql.DB

func handleMatchComplete(data interface{}) error {

	matchID, ok := data.(string)
	if !ok {
		return nil
	}

	ctx, cancel := dbutil.Ctx()
	defer cancel()

	rows, err := DB.QueryContext(ctx, `
	SELECT id
	FROM contests
	WHERE match_id = (
	SELECT id FROM matches_master WHERE external_id = $1)
	AND status != 'completed'
	`, matchID)
	
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {

		var contestID int
		if err := rows.Scan(&contestID); err != nil {
			continue
		}

		//////////////////////////////////////////////////////////////
		// 🔒 LOCK CONTEST (CRITICAL)
		//////////////////////////////////////////////////////////////

		tx, err := DB.BeginTx(ctx, nil)
		if err != nil {
			log.Println(err)
			continue
		}

		var lock int
		err = tx.QueryRowContext(ctx, `
			SELECT id FROM contests WHERE id=$1 FOR UPDATE
		`, contestID).Scan(&lock)

		if err != nil {
			tx.Rollback()
			continue
		}

		// ✅ mark completed (prevents double payout)
		_, err = tx.ExecContext(ctx, `
			UPDATE contests SET status='completed' WHERE id=$1
		`, contestID)

		if err != nil {
			tx.Rollback()
			continue
		}

		tx.Commit()

		//////////////////////////////////////////////////////////////
		// 💰 PAYOUT (OUTSIDE LOCK)
		//////////////////////////////////////////////////////////////

		err = services.ProcessContestPayout(DB, contestID)
		if err != nil {
			log.Println("payout error:", err)
		}
	}

	if err := rows.Err(); err != nil {
		return err
	}

	return nil
}
