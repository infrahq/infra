package redis

import (
	"context"
	"fmt"
	"time"

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
