package cache

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var Rdb *redis.Client
var Ctx = context.Background()

func InitRedis() {

	url := os.Getenv("REDIS_URL")

	if url == "" {
		log.Println("⚠️ REDIS_URL not set — cache disabled")
		Rdb = nil
		return
	}

	Rdb = redis.NewClient(&redis.Options{
		Addr:         url,
		Password:     "", // set if needed
		DB:           0,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		PoolSize:     50,
		MinIdleConns: 10,
	})

	// 🔥 SAFE PING (NO PANIC)
	_, err := Rdb.Ping(Ctx).Result()
	if err != nil {
		log.Println("⚠️ Redis connection failed — running without cache:", err)

		// 🔥 CRITICAL: disable Redis instead of crashing
		Rdb = nil
		return
	}

	log.Println("✅ Redis connected")
}
