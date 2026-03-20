package ai

func CalculateScore(p *PlayerFeatures) {

	p.Score =
		(p.RecentForm * 0.30) +
			(p.VenueScore * 0.15) +
			(p.OpponentScore * 0.15) +
			(p.AvgPoints * 0.10) +
			(p.Consistency * 0.10) +
			(p.RoleWeight * 0.10) +
			(p.ContextScore * 0.10)
}
