package handlers

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"time"

	"gully-cricket/internal/cache"
	"gully-cricket/internal/services"
	dbutil "gully-cricket/internal/db"

	"github.com/gofiber/fiber/v2"
)

//////////////////////////////////////////////////////////////
// JOIN CONTEST
//////////////////////////////////////////////////////////////

type JoinRequest struct {
	ContestID int `json:"contest_id"`
	TeamID    int `json:"team_id"`
}

func JoinContest(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		var req JoinRequest

		// Parse request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid request",
			})
		}

		// Validate input
		if req.ContestID <= 0 || req.TeamID <= 0 {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid contest_id or team_id",
			})
		}

		// Safe user extraction
		userID, ok := c.Locals("user_id").(int)
		if !ok || userID <= 0 {
			return c.Status(401).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}

		// Call service (WITH RETRY SAFETY)
		err := services.JoinContestWithRetry(
			db,
			userID,
			req.TeamID,
			req.ContestID,
		)

		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		//////////////////////////////////////////////////////////////
		// 🔥 CACHE INVALIDATION (CRITICAL)
		//////////////////////////////////////////////////////////////

		// invalidate contests cache (match-wise unknown → safe wipe pattern)
		cache.Rdb.Del(cache.Ctx, "contests:*")

		// invalidate leaderboard
		cache.Rdb.Del(cache.Ctx, "leaderboard:"+strconv.Itoa(req.ContestID))

		return c.JSON(fiber.Map{
			"status": "joined",
		})
	}
}

//////////////////////////////////////////////////////////////
// GET CONTESTS (WITH REDIS CACHE)
//////////////////////////////////////////////////////////////

func GetContests(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		//////////////////////////////////////////////////////////////
		// 1. SAFE PARAM PARSE
		//////////////////////////////////////////////////////////////

		matchIDStr := c.Params("match_id")

		matchID, err := strconv.Atoi(matchIDStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid match id",
			})
		}

		cacheKey := "contests:" + matchIDStr

		//////////////////////////////////////////////////////////////
		// 2. CACHE HIT
		//////////////////////////////////////////////////////////////

		cached, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
		if err == nil {
			return c.Type("json").SendString(cached)
		}

		//////////////////////////////////////////////////////////////
		// 3. DB QUERY (FALLBACK)
		//////////////////////////////////////////////////////////////

		ctx, cancel := dbutil.Ctx()
		defer cancel()

		rows, err := db.QueryContext(ctx, `
			SELECT id, contest_name, prize_pool, total_spots, filled_spots, status
			FROM contests
			WHERE match_id = $1
		`, matchID)

		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "internal server error",
			})
		}
		defer rows.Close()

		var contests []fiber.Map

		for rows.Next() {
			var id int
			var name string
			var prize float64
			var total int
			var filled int
			var status string

			if err := rows.Scan(&id, &name, &prize, &total, &filled, &status); err != nil {
				return c.Status(500).JSON(fiber.Map{
					"error": "failed to read contests",
				})
			}

			contests = append(contests, fiber.Map{
				"id":           id,
				"contest_name": name,
				"prize_pool":   prize,
				"total_spots":  total,
				"filled_spots": filled,
				"status":       status,
			})
		}

		// rows error check
		if err := rows.Err(); err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "failed to process contests",
			})
		}

		if contests == nil {
			contests = []fiber.Map{}
		}

		//////////////////////////////////////////////////////////////
		// 4. CACHE STORE
		//////////////////////////////////////////////////////////////

		bytes, _ := json.Marshal(contests)
		cache.Rdb.Set(cache.Ctx, cacheKey, bytes, 20*time.Second)

		//////////////////////////////////////////////////////////////
		// 5. RESPONSE
		//////////////////////////////////////////////////////////////

		return c.JSON(contests)
	}
}
