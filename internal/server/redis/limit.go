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

// RateOK checks if the rate per minute is acceptable for the specified key
func RateOK(r *Redis, key string, limit int) error {
	if r != nil && r.client != nil {
		ctx := context.TODO()
		limiter := rate.NewLimiter(r.client)
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
			Msg("")

		if result.Allowed <= 0 {
			return OverLimitError{
				RetryAfter: result.RetryAfter,
			}
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

func LoginOK(r *Redis, key string) error {
	if r != nil && r.client != nil {
		lockout, err := r.client.Get(context.TODO(), loginKeyLockout(key)).Time()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				// err is redis.Nil when the key does not exist, i.e. no previous failures
				return nil
			}

			return err
		}

		retryAfter := time.Until(lockout)

		logging.L.Debug().
			Str("key", key).
			Dur("retry_after", retryAfter).
			Msg("")

		if retryAfter > 0 {
			return OverLimitError{
				RetryAfter: retryAfter,
			}
		}
	}

	return nil
}

func LoginGood(r *Redis, key string) {
	if r != nil && r.client != nil {
		ctx := context.TODO()
		_, _ = r.client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Del(ctx, loginKey(key))
			pipe.Del(ctx, loginKeyLockout(key))
			return nil
		})
	}
}

func LoginBad(r *Redis, key string, limit int) {
	if r != nil && r.client != nil {
		ctx := context.TODO()
		rate := r.client.Incr(ctx, loginKey(key)).Val()

		if rate >= int64(limit) {
			retryAfter := time.Duration(math.Pow(1.5, float64(rate)) * float64(time.Second))
			lockout := time.Now().Add(retryAfter)

			logging.L.Debug().
				Str("key", key).
				Int("limit", limit).
				Dur("retry_after", retryAfter).
				Msg("")

			r.client.SetArgs(ctx, loginKeyLockout(key), lockout, redis.SetArgs{
				Mode:     "NX",
				ExpireAt: lockout,
			})
		}
	}
}
