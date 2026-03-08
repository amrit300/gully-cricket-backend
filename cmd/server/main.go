package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/lib/pq"
	_"github.com/lib/pq"
)

var db *sql.DB

// =========================
// MODELS
// =========================

type User struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Whatsapp string `json:"whatsapp"`
	Telegram string `json:"telegram"`
}

type TeamRequest struct {
	UserID      int    `json:"user_id"`
	MatchID     int    `json:"match_id"`
	TeamName    string `json:"team_name"`
	Captain     int    `json:"captain"`
	ViceCaptain int    `json:"vice_captain"`
	Players     []int  `json:"players"`
}

type Player struct {
	ID     int     `json:"id"`
	Name   string  `json:"name"`
	Team   string  `json:"team"`
	Role   string  `json:"role"`
	Credit float64 `json:"credit"`
}

type Contest struct {
	ID         int     `json:"id"`
	MatchID    int     `json:"match_id"`
	PrizePool  float64 `json:"prize_pool"`
	TotalSpots int     `json:"total_spots"`
	FilledSpots int    `json:"filled_spots"`
	Status     string  `json:"status"`
}
// =========================
// MAIN
// =========================

func main() {

	// ---------------------
	// Database Connection
	// ---------------------

	databaseURL := os.Getenv("DATABASE_URL")

log.Println("DATABASE_URL:", databaseURL)

	if databaseURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	var err error

	db, err = sql.Open("postgres", databaseURL)

	if err != nil {
		log.Fatal("DB open error:", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	err = db.Ping()

	if err != nil {
		log.Fatal("DB connection failed:", err)
	}

	log.Println("Database connected")

	// ---------------------
	// Fiber App
	// ---------------------

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {

			log.Println("ERROR:", err)

			return c.Status(500).JSON(fiber.Map{
				"error": "internal server error",
			})
		},
	})

	// ---------------------
	// Routes
	// ---------------------

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Gully Cricket Backend Running")
	})

	app.Get("/matches", getMatches)

	app.Post("/users", createUser)

	app.Get("/players/:match_id", getPlayers)

	app.Post("/teams", createTeam)
	
	app.Get("/contests/:match_id", getContests)

	// ---------------------
	// Start Server
	// ---------------------

	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	log.Println("Server starting on port", port)

	log.Fatal(app.Listen(":" + port))
}

// =========================
// MATCHES
// =========================

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
func getContests(c *fiber.Ctx) error {

	matchID, err := strconv.Atoi(c.Params("match_id"))

	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid match id",
		})
	}

	rows, err := db.Query(`
		SELECT id, match_id, prize_pool, total_spots, filled_spots, status
		FROM contests
		WHERE match_id=$1
	`, matchID)

	if err != nil {
		log.Println(err)

		return c.Status(500).JSON(fiber.Map{
			"error": "database query failed",
		})
	}

	defer rows.Close()

	var contests []Contest

	for rows.Next() {

		var contest Contest

		err := rows.Scan(
			&contest.ID,
			&contest.MatchID,
			&contest.PrizePool,
			&contest.TotalSpots,
			&contest.FilledSpots,
			&contest.Status,
		)

		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "row scan failed",
			})
		}

		contests = append(contests, contest)
	}

	return c.JSON(contests)
}
// =========================
// CREATE USER
// =========================

func createUser(c *fiber.Ctx) error {

	var user User

	if err := c.BodyParser(&user); err != nil {

		return c.Status(400).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if user.Username == "" {

		return c.Status(400).JSON(fiber.Map{
			"error": "username required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	query := `
	INSERT INTO users (username,email,whatsapp,telegram)
	VALUES ($1,$2,$3,$4)
	RETURNING id
	`

	var id int

	err := db.QueryRowContext(
		ctx,
		query,
		user.Username,
		user.Email,
		user.Whatsapp,
		user.Telegram,
	).Scan(&id)

	if err != nil {

		if pqErr, ok := err.(*pq.Error); ok {

			if pqErr.Code == "23505" {

				return c.Status(400).JSON(fiber.Map{
					"error": "username already exists",
				})
			}
		}

		log.Println(err)

		return c.Status(500).JSON(fiber.Map{
			"error": "user creation failed",
		})
	}

	return c.JSON(fiber.Map{
		"user_id": id,
	})
}

// =========================
// GET PLAYERS
// =========================

func getPlayers(c *fiber.Ctx) error {

	param := c.Params("match_id")

	if param == "" {

		return c.Status(400).JSON(fiber.Map{
			"error": "match_id required",
		})
	}

	matchID, err := strconv.Atoi(param)

	if err != nil {

		return c.Status(400).JSON(fiber.Map{
			"error": "invalid match id",
		})
	}

	rows, err := db.Query(`
	SELECT id,name,team,role,credit
	FROM players
	WHERE match_id=$1
	ORDER BY role,credit DESC
	`, matchID)

	if err != nil {

	log.Println("DB QUERY ERROR:", err)

	return c.Status(500).JSON(fiber.Map{
		"error": err.Error(),
	})
}

	defer rows.Close()

	var players []Player

	for rows.Next() {

		var p Player

		err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Team,
			&p.Role,
			&p.Credit,
		)

		if err != nil {

			log.Println(err)

			return c.Status(500).JSON(fiber.Map{
				"error": "row scan failed",
			})
		}

		players = append(players, p)
	}

	return c.JSON(players)
}

// =========================
// CREATE TEAM
// =========================

func createTeam(c *fiber.Ctx) error {

	var req TeamRequest

	if err := c.BodyParser(&req); err != nil {

		return c.Status(400).JSON(fiber.Map{
			"error": "invalid request",
		})
	}

	if req.UserID == 0 || req.MatchID == 0 {

		return c.Status(400).JSON(fiber.Map{
			"error": "user_id and match_id required",
		})
	}

	if req.Captain == req.ViceCaptain {

		return c.Status(400).JSON(fiber.Map{
			"error": "captain and vice captain cannot be same",
		})
	}
	err := checkDailyTeamLimit(req.UserID)

if err != nil {
	return c.Status(400).JSON(fiber.Map{
		"error": err.Error(),
	})
}

	// TEAM VALIDATION
	err := validateTeam(req.Players)

	if err != nil {

		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
		})
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

		_, err = tx.Exec(`
		INSERT INTO team_players (team_id,player_id)
		VALUES ($1,$2)
		`, teamID, playerID)

		if err != nil {

			tx.Rollback()

			return err
		}
	}

	err = tx.Commit()

	if err != nil {

		return err
	}

	return c.JSON(fiber.Map{
		"team_id": teamID,
	})
}

// =========================
// TEAM VALIDATION
// =========================
func checkDailyTeamLimit(userID int) error {

	var teamLimit int

	err := db.QueryRow(`
	SELECT max_teams_per_match
	FROM users
	WHERE id=$1
	`, userID).Scan(&teamLimit)

	if err != nil {
		return fmt.Errorf("failed to fetch user plan")
	}

	var teamsToday int

	err = db.QueryRow(`
	SELECT COUNT(*)
	FROM teams
	WHERE user_id=$1
	AND created_at >= CURRENT_DATE
	`, userID).Scan(&teamsToday)

	if err != nil {
		return fmt.Errorf("failed to count teams today")
	}

	if teamsToday >= teamLimit {
		return fmt.Errorf("daily team limit reached")
	}

	return nil
}
func validateTeam(playerIDs []int) error {

	if len(playerIDs) != 11 {

		return fmt.Errorf("team must contain 11 players")
	}

	rows, err := db.Query(`
	SELECT team,role,credit
	FROM players
	WHERE id = ANY($1)
	`, pq.Array(playerIDs))

if err != nil {
	return err
}

defer rows.Close()

teamCount := map[string]int{}
roleCount := map[string]int{}

totalCredit := 0.0
playerCount := 0

for rows.Next() {

	var team string
	var role string
	var credit float64

	err := rows.Scan(&team, &role, &credit)

	if err != nil {
		return err
	}

	playerCount++

	teamCount[team]++
	roleCount[role]++
	totalCredit += credit
}

if playerCount != 11 {
	return fmt.Errorf("invalid player selection")
}

	if totalCredit > 100 {

		return fmt.Errorf("credit limit exceeded")
	}

	for _, count := range teamCount {

		if count > 7 {

			return fmt.Errorf("max 7 players allowed from one team")
		}
	}

	if roleCount["WK"] < 1 || roleCount["WK"] > 4 {

		return fmt.Errorf("invalid wicketkeeper count")
	}

	if roleCount["BAT"] < 3 || roleCount["BAT"] > 6 {

		return fmt.Errorf("invalid batsman count")
	}

	if roleCount["ALL"] < 1 || roleCount["ALL"] > 4 {

		return fmt.Errorf("invalid allrounder count")
	}

	if roleCount["BOWL"] < 3 || roleCount["BOWL"] > 6 {

		return fmt.Errorf("invalid bowler count")
	}

	return nil
}
