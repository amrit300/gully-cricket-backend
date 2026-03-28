package routes

import (
	"database/sql"

	"gully-cricket/internal/handlers"
	"gully-cricket/internal/ingestion"
	"gully-cricket/internal/middleware"
	"gully-cricket/internal/queue"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(app *fiber.App, db *sql.DB) {

	//////////////////////////////////////////////////////////////
	// 🌐 PUBLIC ROUTES
	//////////////////////////////////////////////////////////////

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Gully Cricket Backend Running")
	})

	// HEALTH
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// MATCHES
	app.Get("/matches", handlers.GetMatches(db))

	// MANUAL SYNC
	app.Get("/sync-matches", func(c *fiber.Ctx) error {
		err := ingestion.SyncMatchesToDB(db)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.JSON(fiber.Map{
			"status": "matches synced",
		})
	})

	// VENUE
	app.Get("/venue-stats/:matchId", handlers.GetVenueStatsHandler(db))

	// PLAYERS
	app.Get("/players/:match_id", handlers.GetPlayers(db))
	app.Get("/sync-players/:match_id/:external_id", handlers.SyncPlayers(db))

	// USER
	app.Post("/user/register", handlers.CreateUser(db))

	// CONTEST VIEW
	app.Get("/contests/:match_id", handlers.GetContests(db))

	// LEADERBOARD
	app.Get("/leaderboard/:contest_id", handlers.GetLeaderboard(db))

	// WALLET WEBHOOK
	app.Post("/webhook/nowpayments", handlers.NowPaymentsWebhook(db))

})

	//////////////////////////////////////////////////////////////
	// 🔐 PROTECTED ROUTES
	//////////////////////////////////////////////////////////////

	api := app.Group("/api",
		middleware.JWTProtected(),
		middleware.RateLimit(),
	)

	// TEAM
	api.Post("/teams", handlers.CreateTeam(db))

	// JOIN
	api.Post("/contest/join", handlers.JoinContest(db))

	// WALLET
	api.Get("/wallet", handlers.GetBalance(db))
	api.Post("/wallet/add", handlers.AddFundsHandler(db))

	// WITHDRAW
	api.Post("/withdraw", handlers.RequestWithdrawal(db)

	api.Get("/queue/stats", func(c *fiber.Ctx) error {
	return c.JSON(queue.Stats())
}
