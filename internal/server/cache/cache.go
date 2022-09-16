package cache

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	rate "github.com/go-redis/redis_rate/v9"

	"github.com/infrahq/infra/internal/logging"
)

type Cache struct {
	client *redis.Client
}

type Options struct {
	Host string
	Port int
	Username string
	Password string
	Options string
}

func NewCache(options Options) *Cache {
	var client *redis.Client

	if len(options.Host) > 0 {
		redisOptions, err := redis.ParseURL(fmt.Sprintf("redis://%s:%d?%s", options.Host, options.Port, options.Options))
		if err != nil {
			logging.Warnf("invalid cache options: %#v", options)
			return nil
		}

		redisOptions.Username = options.Username
		// TODO: read password as a secret
		redisOptions.Password = options.Password

		client = redis.NewClient(redisOptions)
	}

	return &Cache{
		client: client,
	}
}

// RateOK checks if the rate per minute is acceptable for the specified key
func (c *Cache) RateOK(key string, limit int) bool {
	if c.client != nil {
		ctx := context.TODO()
		limiter := rate.NewLimiter(c.client)
		result, err := limiter.Allow(ctx, key, rate.PerMinute(limit))
		if err != nil {
			panic(err)
		}

		logging.L.Debug().
			Int("allowed", result.Allowed).
			Int("remaining", result.Remaining).
			Dur("retry_after", result.RetryAfter).
			Msg("")
		// TODO: also return result.Remaining and result.RetryAfter so the response headers can be set
		return result.Allowed > 0
	}

	return true
}
