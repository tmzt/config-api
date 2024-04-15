package util

import (
	"context"
	"encoding/json"

	"github.com/go-redis/redis/v8"
)

type CacheService struct {
	logger SetRequestLogger
	rdb    *redis.Client

	cache map[string]Cacheable
}

func NewCacheService(rdb *redis.Client) *CacheService {
	logger := NewLogger("CacheService", 0)

	// Change to LRU cache
	cache := make(map[string]Cacheable)

	return &CacheService{
		logger: logger,
		rdb:    rdb,
		cache:  cache,
	}
}

func (s *CacheService) GetCacheObject(ctx context.Context, cacheKey string, data Cacheable) (bool, error) {
	redisRes := s.rdb.Get(ctx, cacheKey)
	if redisRes.Err() == redis.Nil {
		s.logger.Printf("Cache miss for Redis key %s\n", cacheKey)
		return false, nil
	} else if redisRes.Err() != nil {
		s.logger.Printf("Error getting cache value for Redis key (other than not found) %s\n", cacheKey)
		return false, redisRes.Err()
	}

	b, err := redisRes.Bytes()
	if err != nil {
		s.logger.Printf("Error getting cache value as bytes for Redis key %s\n", cacheKey)
		return false, err
	}

	err = json.Unmarshal(b, data)
	if err != nil {
		s.logger.Printf("Error decoding cache data for key %s\n", cacheKey)
		return false, err
	}

	return true, nil
}

func (s *CacheService) SaveCacheObject(ctx context.Context, data Cacheable) error {
	cacheKey := data.CacheKey()

	encoded, err := json.Marshal(data)
	if err != nil {
		s.logger.Println("Error encoding cache data")
		return err
	}

	err = s.rdb.Set(ctx, cacheKey, encoded, data.Ttl()).Err()
	if err != nil {
		s.logger.Printf("Error setting cache value %s", cacheKey)
		return err
	}

	return nil
}

func (s *CacheService) GetCachedObject(ctx context.Context, query Cacheable) (Cacheable, bool, error) {
	cacheKey := query.CacheKey()

	var res Cacheable

	dirty := false
	inRedis := false

	// Get from in-memory cache
	cached, ok := s.cache[cacheKey]
	if ok {
		res = cached
		dirty = true
	}

	// Get from redis
	ok, err := s.GetCacheObject(ctx, cacheKey, query)
	if err != nil {
		s.logger.Printf("Error getting cached object: %s\n", err)
	} else if ok {
		dirty = true
		inRedis = true
		res = query
	}

	if ok && dirty {
		// Set in-memory cache
		s.cache[cacheKey] = res
	}
	if ok && dirty && !inRedis {
		if err := s.SaveCacheObject(ctx, res); err != nil {
			s.logger.Printf("Error saving cache to Redis for key %s\n", cacheKey)
		}
	}

	return res, true, nil
}
