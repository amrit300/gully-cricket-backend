package handlers

import (
	"database/sql"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func GetVenueStatsHandler(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		matchIDStr := c.Params("matchId")
		matchID, err := strconv.Atoi(matchIDStr)

		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid match id",
			})
		}

		var avgScore, paceWickets, spinWickets int

		ctx, cancel := dbutil.Ctx()
defer cancel()

err = db.QueryRowContext(ctx, `
	SELECT
		vs.avgscore,
		vs.pacewickets,
		vs.spinwickets
	FROM venuestats vs
	JOIN matches_master m ON m.venue = vs.venue
	WHERE m.id = $1
	LIMIT 1
`, matchID).Scan(&avgScore, &paceWickets, &spinWickets)
		// ✅ SAFE FALLBACK
		if err == sql.ErrNoRows {
			return c.JSON(defaultVenue())
		}

		if err != nil {
			log.Println("VENUE FETCH ERROR:", err)
			return c.Status(500).JSON(fiber.Map{
				"error": "internal error",
			})
		}

		// 🔥 SMARTER DERIVATION
		total := paceWickets + spinWickets
		if total == 0 {
			total = 1
		}

		paceAssist := (paceWickets * 100) / total
		spinAssist := (spinWickets * 100) / total

		return c.JSON(fiber.Map{
			"avg_score":        avgScore,
			"pace_assist":      paceAssist,
			"spin_assist":      spinAssist,
			"boundary_size":    65,
			"dew_factor":       50,
			"powerplay_bias":   55,
			"death_over_bias":  65,
		})
	}
}

//////////////////////////////////////////////////////////////
// DEFAULT FALLBACK
//////////////////////////////////////////////////////////////

func defaultVenue() fiber.Map {
	return fiber.Map{
		"avg_score":        165,
		"pace_assist":      55,
		"spin_assist":      45,
		"boundary_size":    65,
		"dew_factor":       50,
		"powerplay_bias":   55,
		"death_over_bias":  65,
	}
}
