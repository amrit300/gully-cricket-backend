package ai

import "database/sql"

func GenerateAITeam(db *sql.DB, matchID int) (map[string]interface{}, error) {

	rows, err := db.Query(`
	SELECT 
		p.id, p.name, p.team, p.role, p.credit,
		ps.recent_form,
		ps.venue_points,
		ps.opponent_points,
		ps.avg_points,
		ps.consistency
	FROM players p
	JOIN player_stats ps ON p.id = ps.player_id
	WHERE p.match_id=$1
	`, matchID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []PlayerFeatures

	for rows.Next() {

		var p PlayerFeatures

		rows.Scan(
			&p.PlayerID,
			&p.Name,
			&p.Team,
			&p.Role,
			&p.Credit,
			&p.RecentForm,
			&p.VenueScore,
			&p.OpponentScore,
			&p.AvgPoints,
			&p.Consistency,
		)

		p.ContextScore = 0.5 // placeholder (can upgrade later)

		CalculateFeatures(&p)
		CalculateScore(&p)

		players = append(players, p)
	}

	team := BuildOptimalTeam(players)
	captain, vc := SelectCaptainVC(team)

	return map[string]interface{}{
		"team": team,
		"captain": captain,
		"vice_captain": vc,
	}, nil
}
