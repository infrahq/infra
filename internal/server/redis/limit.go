package redis

import (
	"context"

	rate "github.com/go-redis/redis_rate/v9"

	"github.com/infrahq/infra/internal/logging"
)

// RateOK checks if the rate per minute is acceptable for the specified key
func RateOK(r *Redis, key string, limit int) bool {
	if r != nil && r.client != nil {
		ctx := context.TODO()
		limiter := rate.NewLimiter(r.client)
		result, err := limiter.Allow(ctx, key, rate.PerMinute(limit))
		if err != nil {
			panic(err)
		}

		logging.L.Debug().
			Str("key", key).
			Int("limit", limit).
			Int("allowed", result.Allowed).
			Int("remaining", result.Remaining).
			Dur("retry_after", result.RetryAfter).
			Msg("")
		// TODO: also return result.Remaining and result.RetryAfter so the response headers can be set
		return result.Allowed > 0
	}

	return true
}
