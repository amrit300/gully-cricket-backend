package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gully-cricket/internal/ingestion"
	"gully-cricket/internal/middleware"
	"gully-cricket/internal/routes"
	"gully-cricket/internal/services"
	"gully-cricket/internal/workers"

	"gully-cricket/internal/cache"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	_ "github.com/lib/pq"
)

var db *sql.DB

func main() {

	//////////////////////////////////////////////////////////////
	// 🔐 ENV VALIDATION
	//////////////////////////////////////////////////////////////

	requiredEnv := []string{
		"DATABASE_URL",
		"JWT_SECRET",
		"NOWPAYMENTS_API_KEY",
		"NOWPAYMENTS_IPN_SECRET",
	}

	for _, v := range requiredEnv {
		if os.Getenv(v) == "" {
			log.Fatalf("❌ Missing required env: %s", v)
		}
	}

	log.Println("✅ Environment validated")

	//////////////////////////////////////////////////////////////
	// DATABASE INIT
	//////////////////////////////////////////////////////////////

	databaseURL := os.Getenv("DATABASE_URL") // ✅ FIX

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

	// 🔥 REDIS INIT
cache.InitRedis()
log.Println("✅ Redis connected")

	//////////////////////////////////////////////////////////////
	// BACKGROUND WORKERS
	//////////////////////////////////////////////////////////////

	go services.StartLeaderboardWorker(db)

	go func() {
		for {
			if err := ingestion.UpdateVenueStats(db); err != nil {
				log.Println("Venue update error:", err)
			}
			time.Sleep(6 * time.Hour)
		}
	}()

	go func() {
		for {
			if err := ingestion.SyncMatchesToDB(db); err != nil {
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
	if err = ingestion.SyncMatchesToDB(db); err != nil {
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

	app.Use(recover.New())
	app.Use(logger.New())

	// ✅ GLOBAL RATE LIMIT (FIXED import)
	app.Use(middleware.RateLimit())

	app.Use(cors.New(cors.Config{
		AllowOrigins:     "https://your-frontend-domain.vercel.app",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowCredentials: true,
	}))

	//////////////////////////////////////////////////////////////
	// ROUTES
	//////////////////////////////////////////////////////////////

	routes.RegisterRoutes(app, db)

	//////////////////////////////////////////////////////////////
	// GRACEFUL SHUTDOWN
	//////////////////////////////////////////////////////////////

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

		<-sig
		log.Println("🛑 Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := app.ShutdownWithContext(ctx); err != nil {
			log.Println("Shutdown error:", err)
		}

		os.Exit(0)
	}()

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
