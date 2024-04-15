package connections

import (
	"log"
	"os"

	"github.com/go-redis/redis/v8"
)

func MustNewRedis() *redis.Client {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		log.Fatal("REDIS_URL is not set")
	}

	client := redis.NewClient(&redis.Options{
		Addr:     url,
		Password: "",
		DB:       0,
	})

	return client
}
