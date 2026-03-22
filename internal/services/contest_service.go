func JoinContest(db *sql.DB, userID, contestID, teamID int) error {

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// ❗ Prevent duplicate entry
	var exists int
	err = tx.QueryRow(`
		SELECT 1 FROM contest_entries
		WHERE contest_id=$1 AND user_id=$2 AND team_id=$3
	`, contestID, userID, teamID).Scan(&exists)

	if err == nil {
		tx.Rollback()
		return fmt.Errorf("already joined with this team")
	}

	// Lock contest
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
		tx.Rollback()
		return err
	}

	if status != "upcoming" {
		tx.Rollback()
		return fmt.Errorf("contest locked")
	}

	if filled >= total {
		tx.Rollback()
		return fmt.Errorf("contest full")
	}

	// Deduct wallet safely
	err = DeductBalance(tx, userID, entryFee)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Insert entry
	_, err = tx.Exec(`
		INSERT INTO contest_entries (contest_id, user_id, team_id)
		VALUES ($1,$2,$3)
	`, contestID, userID, teamID)

	if err != nil {
		tx.Rollback()
		return err
	}

	// Update spots
	_, err = tx.Exec(`
		UPDATE contests SET filled_spots = filled_spots + 1 WHERE id=$1
	`, contestID)

	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
