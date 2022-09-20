package redis

import (
	"fmt"

	"github.com/go-redis/redis/v8"

	"github.com/infrahq/infra/internal/logging"
)

type Redis struct {
	client *redis.Client
}

type Options struct {
	Host     string
	Port     int
	Username string
	Password string
	Options  string
}

func NewRedis(options Options) *Redis {
	var client *redis.Client

	if len(options.Host) > 0 {
		redisOptions, err := redis.ParseURL(fmt.Sprintf("redis://%s:%d?%s", options.Host, options.Port, options.Options))
		if err != nil {
			logging.Warnf("invalid redis options: %#v", options)
			return nil
		}

		redisOptions.Username = options.Username
		// TODO: read password as a secret
		redisOptions.Password = options.Password

		client = redis.NewClient(redisOptions)
	}

	return &Redis{
		client: client,
	}
}
