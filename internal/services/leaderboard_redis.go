package services

import (
	"strconv"

	"gully-cricket/internal/cache"
)

//////////////////////////////////////////////////////////////
// UPDATE SCORE (REAL-TIME)
//////////////////////////////////////////////////////////////

func UpdateLeaderboardScore(contestID int, teamID int, points float64) error {

	key := "leaderboard:" + strconv.Itoa(contestID)

	return cache.Rdb.ZAdd(cache.Ctx, key, 
		cache.Z{
			Score:  points,
			Member: teamID,
		},
	).Err()
}
