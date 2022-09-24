package redis

import (
	"crypto/tls"
	"fmt"

	"github.com/go-redis/redis/v8"
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

	TLS TLSOptions
}

type TLSOptions struct {
	Enabled bool

	// TODO: add other TLS options
}

func NewRedis(options Options) (*Redis, error) {
	if options.Host == "" {
		return nil, nil
	}

	redisOptions, err := redis.ParseURL(fmt.Sprintf("redis://%s:%d?%s", options.Host, options.Port, options.Options))
	if err != nil {
		return nil, fmt.Errorf("invalid redis options: %v", options)
	}

	redisOptions.Username = options.Username
	redisOptions.Password = options.Password
	if options.TLS.Enabled {
		redisOptions.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	return &Redis{client: redis.NewClient(redisOptions)}, nil
}
