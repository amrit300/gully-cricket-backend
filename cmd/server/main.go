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
	"gully-cricket/internal/workers"
	"gully-cricket/internal/queue"

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

// 🔥 REDIS FIRST (MANDATORY)
	cache.InitRedis()
	log.Println("✅ Redis connected")

	queue.Init() // 🔥 REQUIRED

// 🔥 THEN WORKERS
	workers.DB = db
	workers.StartWorkerPool(5)
	log.Println("✅ Worker pool initialized")
	workers.StartSubscriptionWorker()   // 🔥 ADD THIS

	//////////////////////////////////////////////////////////////
	// BACKGROUND WORKERS
	//////////////////////////////////////////////////////////////
	go func() {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		if err := ingestion.UpdateVenueStatsWithCtx(ctx, db); err != nil {
			log.Println("Venue update error:", err)
		}

		cancel()

		time.Sleep(6 * time.Hour)
	}
	}()

	go func() {
		for {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	if err = ingestion.SyncMatchesToDBWithCtx(ctx, db); err != nil {
		log.Println("Match sync error:", err)
	}

	cancel() // 🔥 move here

	time.Sleep(10 * time.Minute)
		}
		}()
	//////////////////////////////////////////////////////////////
	// INITIAL SYNC
	//////////////////////////////////////////////////////////////

	go func() {
	log.Println("🚀 Initial match sync...")

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	if err := ingestion.SyncMatchesToDBWithCtx(ctx, db); err != nil {
		log.Println("Initial sync error:", err)
		return
	}

	log.Println("✅ Initial match sync done")
}()

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
	ReadTimeout:  5 * time.Second,
	WriteTimeout: 5 * time.Second,
	IdleTimeout:  30 * time.Second,
	BodyLimit:    2 * 1024 * 1024, // 2MB
})

	app.Use(recover.New())
	app.Use(logger.New())
	
	app.Use(func(c *fiber.Ctx) error {
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("X-Frame-Options", "DENY")
	c.Set("X-XSS-Protection", "1; mode=block")
	c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	return c.Next()
})

	// ✅ GLOBAL RATE LIMIT (FIXED import)
	app.Use(middleware.RateLimit())

	app.Use(cors.New(cors.Config{

	AllowOriginsFunc: func(origin string) bool {
		switch origin {
		case "https://your-frontend-domain.vercel.app":
			return true
		default:
			return false
		}
	},

	AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
	AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",

	// 🔥 safer handling
	AllowCredentials: false,
	
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

		db.Close()
		log.Println("✅ DB connection closed")

		os.Exit(0)
	}()

	//////////////////////////////////////////////////////////////
	// SERVER START
	//////////////////////////////////////////////////////////////

port := os.Getenv("PORT")

if port == "" {
	log.Println("⚠️ PORT not set, defaulting to 8080")
	port = "8080"
}

// 🔥 MUST bind to 0.0.0.0 explicitly
addr := "0.0.0.0:" + port

log.Println("🚀 Server running on", addr)

if err := app.Listen(addr); err != nil {
	log.Fatal("Server failed:", err)
}
}
