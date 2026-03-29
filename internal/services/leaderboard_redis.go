package services

import (
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"gully-cricket/internal/cache"
)

//////////////////////////////////////////////////////////////
// UPDATE SCORE (WRITE PATH)
//////////////////////////////////////////////////////////////

func UpdateLeaderboardScore(contestID int, teamID int, points float64) error {

	key := "leaderboard:" + strconv.Itoa(contestID)

	err := cache.Rdb.ZAdd(cache.Ctx, key,
		redis.Z{
			Score:  points,
			Member: strconv.Itoa(teamID),
		},
	).Err()

	if err != nil {
		return err
	}

	// auto expiry (cleanup old contests)
	cache.Rdb.Expire(cache.Ctx, key, 5*time.Minute)

	return nil
}

//////////////////////////////////////////////////////////////
// GET TOP LEADERBOARD
//////////////////////////////////////////////////////////////

func GetTopLeaderboard(contestID int, limit int64) ([]map[string]interface{}, error) {

	key := "leaderboard:" + strconv.Itoa(contestID)

	results, err := cache.Rdb.ZRevRangeWithScores(cache.Ctx, key, 0, limit-1).Result()
	if err != nil {
		return nil, err
	}

	var leaderboard []map[string]interface{}

	for i, v := range results {
		leaderboard = append(leaderboard, map[string]interface{}{
			"rank":   i + 1,
			"teamId": v.Member,
			"points": v.Score,
		})
	}

	return leaderboard, nil
}

//////////////////////////////////////////////////////////////
// GET USER RANK
//////////////////////////////////////////////////////////////

func GetUserRank(contestID int, teamID int) (int64, error) {

	key := "leaderboard:" + strconv.Itoa(contestID)

	rank, err := cache.Rdb.ZRevRank(cache.Ctx, key, strconv.Itoa(teamID)).Result()
	if err != nil {
		return -1, err
	}

	return rank + 1, nil
}
