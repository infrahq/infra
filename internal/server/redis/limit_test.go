package redis

import (
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"gotest.tools/v3/assert"
)

func TestRateOK(t *testing.T) {
	setup := func(t *testing.T) (*miniredis.Miniredis, *Redis) {
		srv := miniredis.RunT(t)
		port, err := strconv.Atoi(srv.Port())
		assert.NilError(t, err)

		redis, err := NewRedis(Options{Host: srv.Host(), Port: port})
		assert.NilError(t, err)

		return srv, redis
	}

	t.Run("under limit", func(t *testing.T) {
		_, redis := setup(t)
		err := RateOK(redis, "key1", 1)
		assert.NilError(t, err)
	})

	t.Run("over limit", func(t *testing.T) {
		_, redis := setup(t)

		err := RateOK(redis, "key1", 1)
		assert.NilError(t, err)

		err = RateOK(redis, "key1", 1)
		assert.ErrorContains(t, err, "over limit")
	})

	t.Run("limit reset after 1 minute", func(t *testing.T) {
		srv, redis := setup(t)

		err := RateOK(redis, "key1", 1)
		assert.NilError(t, err)

		err = RateOK(redis, "key1", 1)
		assert.ErrorContains(t, err, "over limit")

		srv.FastForward(time.Minute)
		err = RateOK(redis, "key1", 1)
		assert.NilError(t, err)
	})

	t.Run("consistently under limit", func(t *testing.T) {
		srv, redis := setup(t)

		for i := 0; i < 20; i++ {
			err := RateOK(redis, "key1", 10)
			assert.NilError(t, err)
			srv.FastForward(6 * time.Second)
		}
	})

	t.Run("keys are counted separately", func(t *testing.T) {
		_, redis := setup(t)

		keys := []string{"key1", "key2", "key3"}
		for _, key := range keys {
			err := RateOK(redis, key, 1)
			assert.NilError(t, err)

			err = RateOK(redis, key, 1)
			assert.ErrorContains(t, err, "over limit")
		}
	})
}
