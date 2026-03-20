package ai

import "sort"

func SelectCaptainVC(team []PlayerFeatures) (PlayerFeatures, PlayerFeatures) {

	sort.Slice(team, func(i, j int) bool {
		return team[i].Score > team[j].Score
	})

	return team[0], team[1]
}
