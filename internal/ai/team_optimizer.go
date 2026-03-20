package ai

import "sort"

func BuildOptimalTeam(players []PlayerFeatures) []PlayerFeatures {

	sort.Slice(players, func(i, j int) bool {
		return players[i].Score > players[j].Score
	})

	var team []PlayerFeatures

	roleCount := map[string]int{}
	teamCount := map[string]int{}
	totalCredits := 0.0

	for _, p := range players {

		if len(team) == 11 {
			break
		}

		if totalCredits+p.Credit > 100 {
			continue
		}

		if teamCount[p.Team] >= 7 {
			continue
		}

		if !validRole(roleCount, p.Role) {
			continue
		}

		team = append(team, p)

		roleCount[p.Role]++
		teamCount[p.Team]++
		totalCredits += p.Credit
	}

	return team
}

func validRole(rc map[string]int, role string) bool {

	switch role {
	case "WK":
		return rc["WK"] < 2
	case "BAT":
		return rc["BAT"] < 5
	case "ALL":
		return rc["ALL"] < 3
	case "BOWL":
		return rc["BOWL"] < 5
	}
	return true
}
