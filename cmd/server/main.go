package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	_ "github.com/lib/pq"
)

var db *sql.DB

type User struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Whatsapp string `json:"whatsapp"`
	Telegram string `json:"telegram"`
}

type TeamRequest struct {
	UserID      int   `json:"user_id"`
	MatchID     int   `json:"match_id"`
	TeamName    string `json:"team_name"`
	Captain     int   `json:"captain"`
	ViceCaptain int   `json:"vice_captain"`
	Players     []int `json:"players"`
}

func main() {

	// Connect database
	databaseURL := os.Getenv("DATABASE_URL")

	var err error
	db, err = sql.Open("postgres", databaseURL)

	if err != nil {
		log.Fatal(err)
	}

	app := fiber.New()

	// Root route
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Gully Cricket Backend Running")
	})

	// Matches route (temporary mock)
	app.Get("/matches", getMatches)

	// Create user
	app.Post("/users", createUser)

	// Get players
	app.Get("/players", getPlayers)

	// Create fantasy team
	app.Post("/teams", createTeam)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Server starting on port", port)
	log.Fatal(app.Listen(":" + port))
}

func getMatches(c *fiber.Ctx) error {

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
}

func createUser(c *fiber.Ctx) error {

	var user User

	if err := c.BodyParser(&user); err != nil {
		return err
	}

	query := `
	INSERT INTO users (username,email,whatsapp,telegram)
	VALUES ($1,$2,$3,$4)
	RETURNING id
	`

	var id int

	err := db.QueryRow(
		query,
		user.Username,
		user.Email,
		user.Whatsapp,
		user.Telegram,
	).Scan(&id)

	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"user_id": id,
	})
}

func getPlayers(c *fiber.Ctx) error {

	rows, err := db.Query("SELECT id,name,team,role,credit FROM players")

	if err != nil {
		return err
	}

	defer rows.Close()

	type Player struct {
		ID     int     `json:"id"`
		Name   string  `json:"name"`
		Team   string  `json:"team"`
		Role   string  `json:"role"`
		Credit float64 `json:"credit"`
	}

	var players []Player

	for rows.Next() {

		var p Player

		rows.Scan(
			&p.ID,
			&p.Name,
			&p.Team,
			&p.Role,
			&p.Credit,
		)

		players = append(players, p)
	}

	return c.JSON(players)
}

func createTeam(c *fiber.Ctx) error {

	var req TeamRequest

	if err := c.BodyParser(&req); err != nil {
		return err
	}

	tx, err := db.Begin()

	if err != nil {
		return err
	}

	var teamID int

	err = tx.QueryRow(`
	INSERT INTO teams
	(user_id,match_id,team_name,captain_player_id,vice_captain_player_id)
	VALUES ($1,$2,$3,$4,$5)
	RETURNING id
	`,
		req.UserID,
		req.MatchID,
		req.TeamName,
		req.Captain,
		req.ViceCaptain,
	).Scan(&teamID)

	if err != nil {
		tx.Rollback()
		return err
	}

	for _, playerID := range req.Players {

		_, err := tx.Exec(`
		INSERT INTO team_players (team_id,player_id)
		VALUES ($1,$2)
		`, teamID, playerID)

		if err != nil {
			tx.Rollback()
			return err
		}
	}

	tx.Commit()

	return c.JSON(fiber.Map{
		"team_id": teamID,
	})
}
