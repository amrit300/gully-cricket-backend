package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	_ "github.com/lib/pq"
)

type Match struct {
	ID        int    `json:"id"`
	Team1     string `json:"team1"`
	Team2     string `json:"team2"`
	MatchTime string `json:"match_time"`
	Status    string `json:"status"`
}

func main() {

	databaseURL := os.Getenv("DATABASE_URL")

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatal(err)
	}

	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Gully Cricket Backend Running")
	})

	app.Get("/matches", func(c *fiber.Ctx) error {

		rows, err := db.Query("SELECT id, team1, team2, match_time, status FROM matches")
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		var matches []Match

		for rows.Next() {
			var m Match
			rows.Scan(&m.ID, &m.Team1, &m.Team2, &m.MatchTime, &m.Status)
			matches = append(matches, m)
		}

		return c.JSON(matches)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Fatal(app.Listen(":" + port))
}
