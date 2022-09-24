package redis

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/go-redis/redis/v8"
	rate "github.com/go-redis/redis_rate/v9"

	"github.com/infrahq/infra/internal/logging"
)

type OverLimitError struct {
	RetryAfter time.Duration
}

func (e OverLimitError) Error() string {
	return fmt.Sprintf("over limit; retry after %v", e.RetryAfter)
}

type Limiter struct {
	redis *Redis
}

func NewLimiter(redis *Redis) *Limiter {
	return &Limiter{
		redis: redis,
	}
}

// RateOK checks if the rate per minute is acceptable for the specified key
func (lim *Limiter) RateOK(key string, limit int) error {
	if lim.redis == nil {
		return nil
	}

	ctx := context.TODO()
	limiter := rate.NewLimiter(lim.redis.client)
	result, err := limiter.Allow(ctx, key, rate.PerMinute(limit))
	if err != nil {
		return err
	}

	logging.L.Debug().
		Str("key", key).
		Int("limit", limit).
		Int("allowed", result.Allowed).
		Int("remaining", result.Remaining).
		Dur("retry_after", result.RetryAfter).
		Msg("rate limit check")

	if result.Allowed <= 0 {
		return OverLimitError{
			RetryAfter: result.RetryAfter,
		}
	}

	return nil
}

func loginKey(key string) string {
	return fmt.Sprintf("rate:login:%s", key)
}

func loginKeyLockout(key string) string {
	return fmt.Sprintf("%s:lockout", loginKey(key))
}

func (lim *Limiter) LoginOK(key string) error {
	if lim.redis == nil {
		return nil
	}

	lockout, err := lim.redis.client.Get(context.TODO(), loginKeyLockout(key)).Time()
	switch {
	case errors.Is(err, redis.Nil):
	    // err is redis.Nil when the key does not exist, i.e. no previous failures
	    return nil
	case err != nil:
	    return err
	}

	retryAfter := time.Until(lockout)
	ok := retryAfter > 0

	logging.L.Debug().
		Str("key", key).
		Bool("allowed", ok).
		Dur("retry_after", retryAfter).
		Msg("login limit check")

	if ok {
		return OverLimitError{
			RetryAfter: retryAfter,
		}
	}

	return nil
}

func (lim *Limiter) LoginGood(key string) {
	if lim.redis != nil {
		ctx := context.TODO()
		_, err := lim.redis.client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Del(ctx, loginKey(key))
			pipe.Del(ctx, loginKeyLockout(key))
			return nil
		})
		if err != nil {
			logging.L.Error().Err(err).Msg("could not reset lockout timer")
		}
	}
}

func (lim *Limiter) LoginBad(key string, limit int) {
	if lim.redis != nil {
		ctx := context.TODO()
		rate, err := lim.redis.client.Incr(ctx, loginKey(key)).Result()
		if err != nil {
			logging.L.Error().Err(err).Msg("could not increment lockout timer")
		}

		if rate >= int64(limit) {
			retryAfter := time.Duration(math.Pow(1.5, float64(rate)) * float64(time.Second))
			lockout := time.Now().Add(retryAfter)

			logging.L.Debug().
				Str("key", key).
				Int("limit", limit).
				Dur("retry_after", retryAfter).
				Msg("login failed")

			err := lim.redis.client.SetArgs(ctx, loginKeyLockout(key), lockout, redis.SetArgs{
				Mode:     "NX",
				ExpireAt: lockout,
			}).Err()
			if err != nil {
				logging.L.Error().Err(err).Msg("could not set lockout timer")
			}
		}
	}
}
