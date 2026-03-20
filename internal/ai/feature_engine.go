package ai

type PlayerFeatures struct {
	PlayerID int
	Name     string
	Team     string
	Role     string
	Credit   float64

	RecentForm     float64
	VenueScore     float64
	OpponentScore  float64
	AvgPoints      float64
	Consistency    float64
	RoleWeight     float64
	ContextScore   float64

	Score float64
}

func CalculateFeatures(p *PlayerFeatures) {

	// basic derived feature (expand later)
	p.RoleWeight = getRoleWeight(p.Role)

	// consistency (inverse variance assumption for now)
	if p.Consistency == 0 {
		p.Consistency = 0.5
	}
}

func getRoleWeight(role string) float64 {
	switch role {
	case "ALL":
		return 1.3
	case "BOWL":
		return 1.1
	case "BAT":
		return 1.0
	case "WK":
		return 0.9
	default:
		return 1.0
	}
}
