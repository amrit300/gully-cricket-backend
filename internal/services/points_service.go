package services

import (
	"context"
	"database/sql"
)

func CalculateTeamPointsWithCtx(ctx context.Context, db *sql.DB, teamID int) (float64, error) {

	var points float64

	err := db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(p.fantasy_points),0)
		FROM team_players tp
		JOIN players p ON p.id = tp.player_id
		WHERE tp.team_id=$1
	`, teamID).Scan(&points)

	if err != nil {
		return 0, err
	}

	return points, nil
}
