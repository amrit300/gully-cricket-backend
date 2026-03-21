package handlers

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
)

func JoinContest(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		type Req struct {
			UserID    int `json:"user_id"`
			TeamID    int `json:"team_id"`
			ContestID int `json:"contest_id"`
		}

		var req Req
		c.BodyParser(&req)

		_, err := db.Exec(`
			INSERT INTO contest_entries (contest_id,team_id,user_id)
			VALUES ($1,$2,$3)
		`, req.ContestID, req.TeamID, req.UserID)

		if err != nil {
			return err
		}

		return c.JSON(fiber.Map{"status": "joined"})
	}
}
