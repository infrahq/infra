package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"

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

func (c *Cache) RateOK(key string, limit int64) bool {
	if c.client != nil {
		ctx := context.TODO()
		bucket := time.Now().Round(time.Minute)
		key := fmt.Sprintf("%s-%v", key, bucket.Unix())

		incr := c.client.Incr(ctx, key)
		c.client.ExpireAt(ctx, key, bucket.Add(2*time.Minute))
		if incr.Val() >= limit {
			c.client.Decr(ctx, key)

			return false
		}
	}

	return true
}
