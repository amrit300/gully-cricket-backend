package validators

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

//////////////////////////////////////////////////////////////
// 🔒 TEAM VALIDATION (DREAM11 LEVEL)
//////////////////////////////////////////////////////////////

func ValidateTeam(db *sql.DB, playerIDs []int, captain int, vice int) error {

	if len(playerIDs) != 11 {
		return fmt.Errorf("team must contain exactly 11 players")
	}

	playerSet := make(map[int]bool)
	for _, id := range playerIDs {
		if playerSet[id] {
			return fmt.Errorf("duplicate players not allowed")
		}
		playerSet[id] = true
	}

	rows, err := db.Query(`
		SELECT id, team, role, credit
		FROM players
		WHERE id = ANY($1)
	`, pq.Array(playerIDs))

	if err != nil {
		return err
	}
	defer rows.Close()

	teamCount := map[string]int{}
	roleCount := map[string]int{}
	playerMap := make(map[int]bool)

	totalCredit := 0.0
	playerCount := 0

	for rows.Next() {

		var id int
		var team string
		var role string
		var credit float64

		if err := rows.Scan(&id, &team, &role, &credit); err != nil {
			return err
		}

		playerCount++
		playerMap[id] = true

		teamCount[team]++
		roleCount[role]++
		totalCredit += credit
	}

	if playerCount != 11 {
		return fmt.Errorf("invalid player selection")
	}

	if totalCredit > 100 {
		return fmt.Errorf("credit limit exceeded")
	}

	for team, count := range teamCount {
		if count > 7 {
			return fmt.Errorf("max 7 players allowed from one team (%s)", team)
		}
	}

	if roleCount["WK"] < 1 || roleCount["WK"] > 4 {
		return fmt.Errorf("invalid wicketkeeper count")
	}

	if roleCount["BAT"] < 3 || roleCount["BAT"] > 6 {
		return fmt.Errorf("invalid batsman count")
	}

	if roleCount["ALL"] < 1 || roleCount["ALL"] > 4 {
		return fmt.Errorf("invalid allrounder count")
	}

	if roleCount["BOWL"] < 3 || roleCount["BOWL"] > 6 {
		return fmt.Errorf("invalid bowler count")
	}

	if captain == vice {
		return fmt.Errorf("captain and vice captain cannot be same")
	}

	if !playerMap[captain] || !playerMap[vice] {
		return fmt.Errorf("captain/vice must be part of selected players")
	}

	return nil
}

//////////////////////////////////////////////////////////////
// 🔒 MATCH LOCK VALIDATION
//////////////////////////////////////////////////////////////

func ValidateMatchStatus(db *sql.DB, matchID int) error {

	var status string

	err := db.QueryRow(`
		SELECT status FROM matches_master WHERE id=$1
	`, matchID).Scan(&status)

	if err != nil {
		return fmt.Errorf("match not found")
	}

	if status != "Upcoming" {
		return fmt.Errorf("team creation locked, match already started")
	}

	return nil
}

//////////////////////////////////////////////////////////////
// 🔒 TEAM LIMIT VALIDATION
//////////////////////////////////////////////////////////////

func ValidateTeamLimit(db *sql.DB, userID int, matchID int) error {

	var teamLimit int

	err := db.QueryRow(`
		SELECT max_teams_per_match
		FROM users
		WHERE id=$1
	`, userID).Scan(&teamLimit)

	if err != nil {
		return fmt.Errorf("failed to fetch user plan")
	}

	var teamCount int

	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM teams
		WHERE user_id=$1 AND match_id=$2
	`, userID, matchID).Scan(&teamCount)

	if err != nil {
		return fmt.Errorf("failed to count teams")
	}

	if teamCount >= teamLimit {
		return fmt.Errorf("team limit reached for this match")
	}

	return nil
}
