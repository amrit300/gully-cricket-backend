package services

import (
	"database/sql"
	"errors"
	"fmt"
)

func JoinContest(db *sql.DB, userID, teamID, contestID int) error {

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	//////////////////////////////////////////////////////////////
	// 1. DUPLICATE CHECK (SAFE EXISTS)
	//////////////////////////////////////////////////////////////

	var exists bool
	err = tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM contest_entries
			WHERE contest_id=$1 AND user_id=$2 AND team_id=$3
		)
	`, contestID, userID, teamID).Scan(&exists)

	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf("already joined with this team")
	}

	//////////////////////////////////////////////////////////////
	// 2. LOCK CONTEST (CRITICAL)
	//////////////////////////////////////////////////////////////

	var entryFee float64
	var filled, total int
	var status string

	err = tx.QueryRow(`
		SELECT entry_fee, filled_spots, total_spots, status
		FROM contests
		WHERE id=$1
		FOR UPDATE
	`, contestID).Scan(&entryFee, &filled, &total, &status)

	if err != nil {
		return err
	}

	if status != "upcoming" {
		return errors.New("contest locked")
	}

	if filled >= total {
		return errors.New("contest full")
	}

	//////////////////////////////////////////////////////////////
	// 3. WALLET DEDUCTION (ATOMIC)
	//////////////////////////////////////////////////////////////

	err = DeductBalance(tx, userID, entryFee)
	if err != nil {
		return err
	}

	//////////////////////////////////////////////////////////////
	// 4. INSERT ENTRY
	//////////////////////////////////////////////////////////////

	_, err = tx.Exec(`
		INSERT INTO contest_entries (contest_id, user_id, team_id)
		VALUES ($1,$2,$3)
	`, contestID, userID, teamID)

	if err != nil {
		return err
	}

	//////////////////////////////////////////////////////////////
	// 5. UPDATE SPOTS (SAFE INCREMENT)
	//////////////////////////////////////////////////////////////

	res, err := tx.Exec(`
		UPDATE contests
		SET filled_spots = filled_spots + 1
		WHERE id=$1
	`, contestID)

	if err != nil {
		return err
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("failed to update contest spots")
	}

	//////////////////////////////////////////////////////////////
	// 6. LEADERBOARD ENTRY (IMPORTANT)
	//////////////////////////////////////////////////////////////

	_, err = tx.Exec(`
		INSERT INTO leaderboard (contest_id, team_id, user_id, points, rank)
		VALUES ($1,$2,$3,0,0)
	`, contestID, teamID, userID)

	if err != nil {
		return err
	}

	//////////////////////////////////////////////////////////////
	// 7. COMMIT
	//////////////////////////////////////////////////////////////

	return tx.Commit()
}
