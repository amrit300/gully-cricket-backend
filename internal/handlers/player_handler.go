package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"gully-cricket/internal/cache"
	"gully-cricket/internal/providers"

	"github.com/gofiber/fiber/v2"
)

//////////////////////////////////////////////////////////////
// SYNC PLAYERS (FROM EXTERNAL API)
//////////////////////////////////////////////////////////////

func SyncPlayers(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		matchID := c.Params("match_id")
		externalID := c.Params("external_id")

		if matchID == "" || externalID == "" {
			return c.Status(400).JSON(fiber.Map{
				"error": "match_id and external_id required",
			})
		}

		players, err := providers.FetchPlayersFromEntityAPI(externalID)
		if err != nil {
			log.Println("PLAYER FETCH ERROR:", err)
			return c.Status(500).JSON(fiber.Map{
				"error": "failed to fetch players",
			})
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		for _, p := range players {

			name := fmt.Sprintf("%v", p["name"])
			role := fmt.Sprintf("%v", p["role"])
			team := fmt.Sprintf("%v", p["team"])

			_, err := db.ExecContext(ctx, `
				INSERT INTO players (name, team, role, credit, match_id)
				VALUES ($1,$2,$3,8.5,$4)
				ON CONFLICT DO NOTHING
			`, name, team, role, matchID)

			if err != nil {
				log.Println("PLAYER INSERT ERROR:", err)
				continue
			}
		}

		//////////////////////////////////////////////////////////////
		// 🔥 CACHE INVALIDATION
		//////////////////////////////////////////////////////////////

		cache.Rdb.Del(cache.Ctx, "players:"+matchID)

		return c.JSON(fiber.Map{
			"status": "players synced",
		})
	}
}

//////////////////////////////////////////////////////////////
// GET PLAYERS (WITH REDIS CACHE)
//////////////////////////////////////////////////////////////

func GetPlayers(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		matchIDStr := c.Params("match_id")

		matchID, err := strconv.Atoi(matchIDStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid match id",
			})
		}

		cacheKey := "players:" + matchIDStr

		//////////////////////////////////////////////////////////////
		// 🔥 CACHE HIT
		//////////////////////////////////////////////////////////////

		cached, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
		if err == nil {
			return c.Type("json").SendString(cached)
		}

		//////////////////////////////////////////////////////////////
		// 🔥 DB QUERY
		//////////////////////////////////////////////////////////////

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		rows, err := db.QueryContext(ctx, `
			SELECT id, name, team, role, credit, fantasy_points
			FROM players
			WHERE match_id = $1
		`, matchID)

		if err != nil {
			log.Println("GET PLAYERS ERROR:", err)
			return c.Status(500).JSON(fiber.Map{
				"error": "failed to fetch players",
			})
		}
		defer rows.Close()

		var players []map[string]interface{}

		for rows.Next() {
			var id int
			var name, team, role string
			var credit, fantasyPoints float64

			if err := rows.Scan(&id, &name, &team, &role, &credit, &fantasyPoints); err != nil {
				continue
			}

			players = append(players, fiber.Map{
				"id":              id,
				"name":            name,
				"team":            team,
				"role":            role,
				"credit":          credit,
				"fantasy_points":  fantasyPoints,
			})
		}

		if err := rows.Err(); err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "failed to read players",
			})
		}

		if players == nil {
			players = []map[string]interface{}{}
		}

		//////////////////////////////////////////////////////////////
		// 🔥 CACHE STORE
		//////////////////////////////////////////////////////////////

		bytes, _ := json.Marshal(players)
		cache.Rdb.Set(cache.Ctx, cacheKey, bytes, 30*time.Second)

		return c.JSON(players)
	}
}
