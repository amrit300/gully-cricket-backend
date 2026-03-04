package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
)

func main() {

	app := fiber.New()

	// Root route
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Gully Cricket Backend Running")
	})

	// Matches route (temporary mock)
	app.Get("/matches", func(c *fiber.Ctx) error {

		type Match struct {
			ID        int    `json:"id"`
			Team1     string `json:"team1"`
			Team2     string `json:"team2"`
			MatchTime string `json:"match_time"`
			Status    string `json:"status"`
		}

		matches := []Match{
			{
				ID:        1,
				Team1:     "India",
				Team2:     "Australia",
				MatchTime: "2026-03-05T19:30:00Z",
				Status:    "upcoming",
			},
		}

		return c.JSON(matches)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Server starting on port", port)
	log.Fatal(app.Listen(":" + port))
}
