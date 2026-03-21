package handlers

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
)

//////////////////////////////////////////////////////////////
// JOIN CONTEST (CRITICAL FLOW)
//////////////////////////////////////////////////////////////

func JoinContest(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		type Request struct {
			UserID    int `json:"user_id"`
			TeamID    int `json:"team_id"`
			ContestID int `json:"contest_id"`
		}

		var req Request

		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid request",
			})
		}

		// 🔥 START TRANSACTION
		tx, err := db.Begin()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "db error"})
		}
		defer tx.Rollback()

		//////////////////////////////////////////////////////////////
		// 1. PREVENT DUPLICATE ENTRY
		//////////////////////////////////////////////////////////////

		var exists int

		err = tx.QueryRow(`
			SELECT 1 FROM contest_entries
			WHERE contest_id=$1 AND team_id=$2
		`, req.ContestID, req.TeamID).Scan(&exists)

		if err == nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "already joined",
			})
		}

		//////////////////////////////////////////////////////////////
		// 2. LOCK CONTEST (CRITICAL)
		//////////////////////////////////////////////////////////////

		var filled, total int
		var entryFee float64

		err = tx.QueryRow(`
			SELECT filled_spots, total_spots, entry_fee
			FROM contests
			WHERE id=$1
			FOR UPDATE
		`, req.ContestID).Scan(&filled, &total, &entryFee)

		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "contest not found",
			})
		}

		if filled >= total {
			return c.Status(400).JSON(fiber.Map{
				"error": "contest full",
			})
		}

		//////////////////////////////////////////////////////////////
		// 3. TEAM OWNERSHIP CHECK
		//////////////////////////////////////////////////////////////

		var matchID, ownerID int

		err = tx.QueryRow(`
			SELECT match_id, user_id FROM teams WHERE id=$1
		`, req.TeamID).Scan(&matchID, &ownerID)

		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "team not found",
			})
		}

		if ownerID != req.UserID {
			return c.Status(403).JSON(fiber.Map{
				"error": "unauthorized team access",
			})
		}

		//////////////////////////////////////////////////////////////
		// 4. (FUTURE) WALLET DEDUCTION HOOK
		//////////////////////////////////////////////////////////////

		/*
		// 🔥 ENABLE THIS LATER

		err = deductWallet(tx, req.UserID, entryFee)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "insufficient balance",
			})
		}
		*/

		//////////////////////////////////////////////////////////////
		// 5. INSERT ENTRY
		//////////////////////////////////////////////////////////////

		_, err = tx.Exec(`
			INSERT INTO contest_entries (contest_id,team_id,user_id)
			VALUES ($1,$2,$3)
		`, req.ContestID, req.TeamID, req.UserID)

		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "entry failed",
			})
		}

		//////////////////////////////////////////////////////////////
		// 6. CREATE LEADERBOARD ENTRY
		//////////////////////////////////////////////////////////////

		_, err = tx.Exec(`
			INSERT INTO leaderboard (contest_id, match_id, team_id, points, rank)
			VALUES ($1,$2,$3,0,0)
		`, req.ContestID, matchID, req.TeamID)

		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "leaderboard insert failed",
			})
		}

		//////////////////////////////////////////////////////////////
		// 7. UPDATE FILLED SPOTS
		//////////////////////////////////////////////////////////////

		_, err = tx.Exec(`
			UPDATE contests
			SET filled_spots = filled_spots + 1
			WHERE id=$1
		`, req.ContestID)

		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "update failed",
			})
		}

		//////////////////////////////////////////////////////////////
		// ✅ COMMIT TRANSACTION
		//////////////////////////////////////////////////////////////

		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "commit failed",
			})
		}

		return c.JSON(fiber.Map{
			"status": "contest joined",
		})
	}
}
