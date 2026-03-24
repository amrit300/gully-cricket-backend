package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	"gully-cricket/internal/handlers"
	"gully-cricket/internal/ingestion"
	"gully-cricket/internal/middleware"
	"gully-cricket/internal/services"
	"gully-cricket/internal/workers"
	"gully-cricket/internal/routes"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	_ "github.com/lib/pq"
)

var db *sql.DB

func main() {

	//////////////////////////////////////////////////////////////
	// 🔐 ENV VALIDATION (CRITICAL)
	//////////////////////////////////////////////////////////////

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET not set")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	//////////////////////////////////////////////////////////////
	// DATABASE INIT
	//////////////////////////////////////////////////////////////

	var err error

	db, err = sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatal("DB open error:", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err = db.Ping(); err != nil {
		log.Fatal("DB connection failed:", err)
	}

	log.Println("✅ Database connected")

	//////////////////////////////////////////////////////////////
	// BACKGROUND WORKERS
	//////////////////////////////////////////////////////////////

	go services.StartLeaderboardWorker(db)

	go func() {
		for {
			err := ingestion.UpdateVenueStats(db)
			if err != nil {
				log.Println("Venue update error:", err)
			}
			time.Sleep(6 * time.Hour)
		}
	}()

	go func() {
		for {
			err := ingestion.SyncMatchesToDB(db)
			if err != nil {
				log.Println("Match sync error:", err)
			}
			time.Sleep(10 * time.Minute)
		}
	}()

	go func() {
		for {
			workers.ProcessCompletedMatches(db)
			time.Sleep(30 * time.Second)
		}
	}()

	//////////////////////////////////////////////////////////////
	// INITIAL SYNC
	//////////////////////////////////////////////////////////////

	log.Println("🚀 Initial match sync...")
	err = ingestion.SyncMatchesToDB(db)
	if err != nil {
		log.Println("Initial sync error:", err)
	}
	log.Println("✅ Initial match sync done")

	//////////////////////////////////////////////////////////////
	// FIBER APP
	//////////////////////////////////////////////////////////////

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			log.Println("SERVER ERROR:", err)
			return c.Status(500).JSON(fiber.Map{
				"error": "internal server error",
			})
		},
	})

	// ✅ RECOVER (crash safety)
	app.Use(recover.New())

	// ✅ REQUEST LOGGER
	app.Use(logger.New())

	// 🔐 SECURE CORS (NO WILDCARD)
	app.Use(cors.New(cors.Config{
		AllowOrigins: "https://your-frontend-domain.vercel.app",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowCredentials: true,
	}))

	//////////////////////////////////////////////////////////////
	// HEALTH CHECK (CI/CD)
	//////////////////////////////////////////////////////////////

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
		})
	})

	/////////////
// 🌐  ROUTES //
	///////////

	
	routes.RegisterRoutes(app, db)
	//////////////////////////////////////////////////////////////
	// SERVER START
	//////////////////////////////////////////////////////////////

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("🚀 Server running on port", port)
	log.Fatal(app.Listen(":" + port))
}
