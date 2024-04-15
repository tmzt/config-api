package util

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

type Cacheable interface {
	CacheKey() string
	Ttl() time.Duration
}

func SetCache(rdb *redis.Client, data Cacheable) error {
	cacheKey := data.CacheKey()

	encoded, err := json.Marshal(data)
	if err != nil {
		log.Println("Error encoding cache data")
		return err
	}

	err = rdb.Set(context.Background(), cacheKey, encoded, data.Ttl()).Err()
	if err != nil {
		log.Printf("Error setting cache value %s", cacheKey)
		return err
	}

	return nil
}

func GetCache(ctx context.Context, rdb *redis.Client, cacheKey string, data Cacheable) error {
	encoded, err := rdb.Get(ctx, cacheKey).Bytes()
	if err != nil {
		log.Printf("Error getting cache value as bytes for Redis key %s\n", cacheKey)
		return err
	}

	err = json.Unmarshal(encoded, data)
	if err != nil {
		log.Printf("Error decoding cache data for key %s\n", cacheKey)
		return err
	}

	return nil
}
