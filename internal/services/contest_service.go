package services

import (
	"database/sql"
	"errors"
)

func JoinContest(db *sql.DB, userID, teamID, contestID int) error {

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	//////////////////////////////////////////////////////////////
	// 1. CHECK USER PLAN
	//////////////////////////////////////////////////////////////

	var maxTeams int

	err = tx.QueryRow(`
		SELECT max_teams_per_match
		FROM users
		WHERE id=$1
	`, userID).Scan(&maxTeams)

	if err != nil {
		return err
	}

	//////////////////////////////////////////////////////////////
	// 2. COUNT CURRENT TEAMS IN THIS MATCH
	//////////////////////////////////////////////////////////////

	var currentTeams int

	err = tx.QueryRow(`
		SELECT COUNT(*)
		FROM contest_entries ce
		JOIN teams t ON t.id = ce.team_id
		WHERE ce.user_id=$1 AND t.match_id = (
			SELECT match_id FROM contests WHERE id=$2
		)
	`, userID, contestID).Scan(&currentTeams)

	if err != nil {
		return err
	}

	if currentTeams >= maxTeams {
		return errors.New("team limit reached for your plan")
	}

	//////////////////////////////////////////////////////////////
	// 3. LOCK CONTEST
	//////////////////////////////////////////////////////////////

	var filled, total int
	var status string

	err = tx.QueryRow(`
		SELECT filled_spots, total_spots, status
		FROM contests
		WHERE id=$1
		FOR UPDATE
	`, contestID).Scan(&filled, &total, &status)

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
	// 4. PREVENT DUPLICATE
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
		return errors.New("already joined with this team")
	}

	//////////////////////////////////////////////////////////////
	// 5. INSERT ENTRY (NO MONEY INVOLVED)
	//////////////////////////////////////////////////////////////

	_, err = tx.Exec(`
		INSERT INTO contest_entries (contest_id, user_id, team_id)
		VALUES ($1,$2,$3)
	`, contestID, userID, teamID)

	if err != nil {
		return err
	}

	//////////////////////////////////////////////////////////////
	// 6. UPDATE SPOTS
	//////////////////////////////////////////////////////////////

	_, err = tx.Exec(`
		UPDATE contests
		SET filled_spots = filled_spots + 1
		WHERE id=$1
	`, contestID)

	if err != nil {
		return err
	}

	//////////////////////////////////////////////////////////////
	// 7. LEADERBOARD
	//////////////////////////////////////////////////////////////

	_, err = tx.Exec(`
		INSERT INTO leaderboard (contest_id, team_id, user_id, points, rank)
		VALUES ($1,$2,$3,0,0)
	`, contestID, teamID, userID)

	if err != nil {
		return err
	}

	return tx.Commit()
}
