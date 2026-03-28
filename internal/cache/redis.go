package cache

import (
	"context"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var Rdb *redis.Client
var Ctx = context.Background()

func InitRedis() {

	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = "localhost:6379"
	}

	Rdb = redis.NewClient(&redis.Options{
		Addr:         url,
		Password:     "",
		DB:           0,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		PoolSize:     50,
		MinIdleConns: 10,
	})

	_, err := Rdb.Ping(Ctx).Result()
	if err != nil {
		panic("Redis connection failed")
	}
}
