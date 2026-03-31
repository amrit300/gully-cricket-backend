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

	opt, err := redis.ParseURL(url)
	if err != nil {
		log.Println("⚠️ Redis parse failed — running without cache:", err)
		return
	}

	Rdb = redis.NewClient(opt)

	_, err = Rdb.Ping(Ctx).Result()
	if err != nil {
		log.Println("⚠️ Redis connection failed — running without cache:", err)
		return
	}

	log.Println("✅ Redis connected")
}
