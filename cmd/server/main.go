package main

import (
	"gully-cricket/internal"
)
import "gully-cricket/internal/ingestion"
import "gully-cricket/internal/ai"
import "github.com/gofiber/fiber/v2/middleware/cors"
import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
	"encoding/json"
	"net/url"

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
type Entry struct {
	TeamID int `json:"team_id"`
	Points float64 `json:"points"`
	Rank int `json:"rank"`
}
type RegisterRequest struct {
	Username string `json:"username"`
	InitData string `json:"initData"`
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
	go leaderboardWorker()
	
	go func() {
	for {
		err := ingestion.UpdateVenueStats(db)
		if err != nil {
			log.Println("Venue update error:", err)
		}
		time.Sleep(6 * time.Hour)
	}
}()

	// ---------------------
	// Fiber App
	// ---------------------

	app := fiber.New(fiber.Config{
	ErrorHandler: func(c *fiber.Ctx, err error) error {

		log.Println("SERVER ERROR:", err)

		return c.Status(500).JSON(fiber.Map{
			"error": "internal server error",
		})
	},
})

app.Use(cors.New(cors.Config{
	AllowOrigins: "*",
	AllowHeaders: "Origin, Content-Type, Accept",
	AllowMethods: "GET,POST,OPTIONS",
}))

app.Options("/*", func(c *fiber.Ctx) error {
	return c.SendStatus(200)
})

	// ---------------------
	// Routes
	// ---------------------

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Gully Cricket Backend Running")
	})

	app.Get("/matches", func(c *fiber.Ctx) error {

	matches, err := internal.GetMatches()

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(matches)
})
	app.Get("/ai/team/:match_id", func(c *fiber.Ctx) error {

	matchID, _ := strconv.Atoi(c.Params("match_id"))

	result, err := ai.GenerateAITeam(db, matchID)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(result)
})
	app.Get("/sync-players/:match_id/:external_id", func(c *fiber.Ctx) error {

	matchID, _ := strconv.Atoi(c.Params("match_id"))
	externalID := c.Params("external_id")

	err := syncPlayers(matchID, externalID)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status": "players synced",
	})
})

	app.Post("/user/register", createUser)

	app.Get("/players/:match_id", getPlayers)

	app.Post("/teams", createTeam)
	
	app.Get("/contests/:match_id", getContests)

	app.Post("/join-contest", joinContest)

	app.Get("/leaderboard/:contest_id", getLeaderboard)

	app.Post("/update-leaderboard", updateLeaderboard)

	app.Post("/match-event", processMatchEvent)

	app.Post("/update-team-points", updateTeamPoints)
	

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

	var req RegisterRequest

	if err := c.BodyParser(&req); err != nil {

		log.Println("BODY PARSE ERROR:", err)

		return c.Status(400).JSON(fiber.Map{
			"error": "invalid request",
		})
	}

	log.Println("USERNAME:", req.Username)
	log.Println("INIT DATA:", req.InitData)

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")

	if botToken == "" {
		log.Println("BOT TOKEN MISSING")
		return c.Status(500).JSON(fiber.Map{
			"error": "server configuration error",
		})
	}

	/*
	Verify Telegram HMAC signature
	*/

	if !verifyTelegram(req.InitData, botToken) {

		log.Println("TELEGRAM VERIFICATION FAILED")

		return c.Status(403).JSON(fiber.Map{
			"error": "telegram verification failed",
		})
	}

	/*
	Extract Telegram user id
	*/

	values, err := url.ParseQuery(req.InitData)

	if err != nil {
		log.Println("INIT DATA PARSE ERROR:", err)
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid telegram payload",
		})
	}

	userJSON := values.Get("user")

if userJSON == "" {

	log.Println("USER FIELD MISSING IN INIT DATA")

	return c.Status(400).JSON(fiber.Map{
		"error": "telegram user missing",
	})
}

var telegramUser struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
}

if err := json.Unmarshal([]byte(userJSON), &telegramUser); err != nil {

	log.Println("USER JSON PARSE ERROR:", err)

	return c.Status(400).JSON(fiber.Map{
		"error": "invalid telegram user",
	})
}

	log.Println("TELEGRAM USER ID:", telegramUser.ID)

	query := `
	INSERT INTO users (username, telegram)
	VALUES ($1,$2)
	ON CONFLICT (telegram)
	DO UPDATE SET username=EXCLUDED.username
	RETURNING id
	`

	var id int

err = db.QueryRow(
	query,
	req.Username,
	telegramUser.ID,
).Scan(&id)

if err != nil {

	log.Printf("USER INSERT ERROR: %+v\n", err)

	// 🔥 RETURN REAL ERROR TO FRONTEND
	return c.Status(500).JSON(fiber.Map{
		"error": err.Error(),
	})

}
	log.Println("USER CREATED:", id)

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
	var err error

	if err = c.BodyParser(&req); err != nil {
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

	err = checkDailyTeamLimit(req.UserID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	err = validateTeam(req.Players)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

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
func joinContest(c *fiber.Ctx) error {

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

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	/* =========================
	   DUPLICATE CHECK
	========================= */

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

	/* =========================
	   LOCK CONTEST
	========================= */

	var filled, total int

	err = tx.QueryRow(`
	SELECT filled_spots, total_spots
	FROM contests
	WHERE id=$1
	FOR UPDATE
	`, req.ContestID).Scan(&filled, &total)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "contest query failed",
		})
	}

	if filled >= total {
		return c.Status(400).JSON(fiber.Map{
			"error": "contest full",
		})
	}

	/* =========================
	   TEAM OWNERSHIP CHECK
	========================= */

	var matchID, ownerID int

	err = tx.QueryRow(`
	SELECT match_id, user_id FROM teams WHERE id=$1
	`, req.TeamID).Scan(&matchID, &ownerID)

	if err != nil {
		return err
	}

	if ownerID != req.UserID {
		return c.Status(403).JSON(fiber.Map{
			"error": "unauthorized team access",
		})
	}

	/* =========================
	   INSERT ENTRY
	========================= */

	_, err = tx.Exec(`
	INSERT INTO contest_entries (contest_id,team_id,user_id)
	VALUES ($1,$2,$3)
	`, req.ContestID, req.TeamID, req.UserID)

	if err != nil {
		return err
	}

	/* =========================
	   INSERT LEADERBOARD
	========================= */

	_, err = tx.Exec(`
	INSERT INTO leaderboard (contest_id, match_id, team_id, points, rank)
	VALUES ($1,$2,$3,0,0)
	`, req.ContestID, matchID, req.TeamID)

	if err != nil {
		return err
	}

	/* =========================
	   UPDATE SPOTS
	========================= */

	_, err = tx.Exec(`
	UPDATE contests
	SET filled_spots = filled_spots + 1
	WHERE id=$1
	`, req.ContestID)

	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"status": "contest joined",
	})
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
/* THEN VALIDATION STARTS */

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

func getLeaderboard(c *fiber.Ctx) error {

contestID, err := strconv.Atoi(c.Params("contest_id"))

if err != nil {
	return c.Status(400).JSON(fiber.Map{
		"error":"invalid contest id",
	})
}

rows, err := db.Query(`
SELECT team_id, points, rank
FROM leaderboard
WHERE match_id=$1
ORDER BY rank ASC
`, contestID)

if err != nil {
	return c.Status(500).JSON(fiber.Map{
		"error":"database query failed",
	})
}

defer rows.Close()

var leaderboard []Entry

for rows.Next() {

	var e Entry

	err := rows.Scan(
		&e.TeamID,
		&e.Points,
		&e.Rank,
	)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":"row scan failed",
		})
	}

	leaderboard = append(leaderboard,e)
}

return c.JSON(leaderboard)
}

func updateLeaderboard(c *fiber.Ctx) error {

	type Request struct {
		MatchID int `json:"match_id"`
	}

	var req Request

	if err := c.BodyParser(&req); err != nil {
		return err
	}

	_, err := db.Exec(`
	UPDATE leaderboard l
	SET rank = r.rank
	FROM (
		SELECT team_id,
		RANK() OVER (ORDER BY points DESC) as rank
		FROM leaderboard
		WHERE match_id=$1
	) r
	WHERE l.team_id=r.team_id
	`, req.MatchID)

	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"status":"leaderboard updated",
	})
}

func processMatchEvent(c *fiber.Ctx) error {

	type Event struct {
		PlayerID int `json:"player_id"`
		Event    string `json:"event"`
		Value    int `json:"value"`
	}

	var e Event

	if err := c.BodyParser(&e); err != nil {
		return err
	}

	points := 0

	switch e.Event {

	case "run":
		points = e.Value

	case "four":
		points = 1

	case "six":
		points = 2

	case "wicket":
		points = 25

	case "catch":
		points = 8

	case "stumping":
		points = 12
	}

	_, err := db.Exec(`
	UPDATE players
	SET fantasy_points = fantasy_points + $1
	WHERE id=$2
	`, points, e.PlayerID)

	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"status": "event processed",
	})
}

func updateTeamPoints(c *fiber.Ctx) error {

	type Request struct {
		MatchID int `json:"match_id"`
	}

	var req Request

	if err := c.BodyParser(&req); err != nil {
		return err
	}

	rows, err := db.Query(`
	SELECT 
	  t.id,
	  COALESCE(SUM(
	    CASE 
	      WHEN tp.player_id = t.captain_player_id THEN p.fantasy_points * 2
	      WHEN tp.player_id = t.vice_captain_player_id THEN p.fantasy_points * 1.5
	      ELSE p.fantasy_points
	    END
	  ),0) as total
	FROM teams t
	JOIN team_players tp ON tp.team_id = t.id
	JOIN players p ON p.id = tp.player_id
	WHERE t.match_id = $1
	GROUP BY t.id
	`, req.MatchID)

	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {

		var teamID int
		var points float64

		err := rows.Scan(&teamID, &points)
		if err != nil {
			return err
		}

		_, err = db.Exec(`
		UPDATE teams
		SET total_points = $1
		WHERE id = $2
		`, points, teamID)

		if err != nil {
			return err
		}
	}

	return c.JSON(fiber.Map{
		"status": "team points updated",
	})
}

func leaderboardWorker() {

	for {

		time.Sleep(10 * time.Second)

		rows, err := db.Query(`
		SELECT DISTINCT contest_id, match_id
		FROM leaderboard
		`)

		if err != nil {
			log.Println("worker query error:", err)
			continue
		}

		for rows.Next() {

			var contestID int
			var matchID int

			if err := rows.Scan(&contestID, &matchID); err != nil {
				log.Println("scan error:", err)
				continue
			}

			/* =========================
			   STEP 1 → UPDATE TEAM POINTS (MATCH LEVEL)
			========================= */

			_, err = db.Exec(`
			UPDATE teams t
			SET total_points = sub.points
			FROM (
				SELECT 
				  t2.id as team_id,
				  COALESCE(SUM(
				    CASE 
				      WHEN tp.player_id = t2.captain_player_id THEN p.fantasy_points * 2
				      WHEN tp.player_id = t2.vice_captain_player_id THEN p.fantasy_points * 1.5
				      ELSE p.fantasy_points
				    END
				  ),0) as points
				FROM teams t2
				JOIN team_players tp ON tp.team_id = t2.id
				JOIN players p ON p.id = tp.player_id
				WHERE t2.match_id = $1
				GROUP BY t2.id
			) sub
			WHERE t.id = sub.team_id
			AND t.match_id = $1
			`, matchID)

			if err != nil {
				log.Println("team update error:", err)
				continue
			}

			/* =========================
			   STEP 2 → SYNC LEADERBOARD (CONTEST LEVEL)
			========================= */

			_, err = db.Exec(`
			UPDATE leaderboard l
			SET points = t.total_points
			FROM teams t
			WHERE l.team_id = t.id
			AND l.contest_id = $1
			`, contestID)

			if err != nil {
				log.Println("leaderboard sync error:", err)
				continue
			}

			/* =========================
			   STEP 3 → RANK PER CONTEST
			========================= */

			_, err = db.Exec(`
			UPDATE leaderboard l
			SET rank = r.rank
			FROM (
				SELECT 
				  team_id,
				  RANK() OVER (ORDER BY points DESC) as rank
				FROM leaderboard
				WHERE contest_id = $1
			) r
			WHERE l.team_id = r.team_id
			AND l.contest_id = $1
			`, contestID)

			if err != nil {
				log.Println("leaderboard rank error:", err)
				continue
			}
		}

		rows.Close()
	}
}
func syncPlayers(matchID int, externalMatchID string) error {

	players, err := internal.FetchPlayersFromCricAPI(externalMatchID)
	if err != nil {
		return err
	}

	for _, p := range players {

		name := fmt.Sprintf("%v", p["name"])

		role := "BAT" // fallback

		if r, ok := p["role"]; ok {
			role = fmt.Sprintf("%v", r)
		}

		team := "Unknown"

		if t, ok := p["team"]; ok {
			team = fmt.Sprintf("%v", t)
		}

		_, err := db.Exec(`
		INSERT INTO players (name, team, role, credit, match_id)
		VALUES ($1,$2,$3,8.5,$4)
		ON CONFLICT DO NOTHING
		`,
			name,
			team,
			role,
			matchID,
		)

		if err != nil {
			log.Println("PLAYER INSERT ERROR:", err)
		}
	}

	return nil
}
