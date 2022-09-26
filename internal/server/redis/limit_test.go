package redis

import (
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/opt"
)

func TestRateOK(t *testing.T) {
	setup := func(t *testing.T) (*miniredis.Miniredis, *Limiter) {
		srv := miniredis.RunT(t)
		port, err := strconv.Atoi(srv.Port())
		assert.NilError(t, err)

		redis, err := NewRedis(Options{Host: srv.Host(), Port: port})
		assert.NilError(t, err)

		return srv, NewLimiter(redis)
	}

	t.Run("under limit", func(t *testing.T) {
		_, lim := setup(t)
		err := lim.RateOK("key1", 1)
		assert.NilError(t, err)
	})

	t.Run("over limit", func(t *testing.T) {
		_, lim := setup(t)

		err := lim.RateOK("key1", 1)
		assert.NilError(t, err)

		err = lim.RateOK("key1", 1)
		assert.ErrorContains(t, err, "over limit")
	})

	t.Run("limit reset after 1 minute", func(t *testing.T) {
		srv, lim := setup(t)

		err := lim.RateOK("key1", 1)
		assert.NilError(t, err)

		err = lim.RateOK("key1", 1)
		assert.ErrorContains(t, err, "over limit")

		srv.FastForward(time.Minute)
		err = lim.RateOK("key1", 1)
		assert.NilError(t, err)
	})

	t.Run("consistently under limit", func(t *testing.T) {
		srv, lim := setup(t)

		for i := 0; i < 20; i++ {
			err := lim.RateOK("key1", 10)
			assert.NilError(t, err)
			srv.FastForward(6 * time.Second)
		}
	})

	t.Run("keys are counted separately", func(t *testing.T) {
		_, lim := setup(t)

		keys := []string{"key1", "key2", "key3"}
		for _, key := range keys {
			err := lim.RateOK(key, 1)
			assert.NilError(t, err)

			err = lim.RateOK(key, 1)
			assert.ErrorContains(t, err, "over limit")
		}
	})
}

func TestLoginOK(t *testing.T) {
	setup := func(t *testing.T) (*miniredis.Miniredis, *Limiter) {
		srv := miniredis.RunT(t)
		port, err := strconv.Atoi(srv.Port())
		assert.NilError(t, err)

		redis, err := NewRedis(Options{Host: srv.Host(), Port: port})
		assert.NilError(t, err)

		return srv, NewLimiter(redis)
	}

	t.Run("under limit", func(t *testing.T) {
		_, lim := setup(t)
		err := lim.LoginOK("admin@example.com")
		assert.NilError(t, err)
	})

	t.Run("over limit", func(t *testing.T) {
		_, lim := setup(t)

		lim.LoginBad("admin@example.com", 1)

		expected, _ := time.ParseDuration("1.5s")
		err := lim.LoginOK("admin@example.com")
		assert.DeepEqual(t, err, OverLimitError{
			RetryAfter: expected,
		}, opt.DurationWithThreshold(100*time.Millisecond))
	})

	t.Run("way over limit", func(t *testing.T) {
		_, lim := setup(t)

		for i := 0; i < 10; i++ {
			lim.LoginBad("admin@example.com", 10)
		}

		expected, _ := time.ParseDuration("57s")
		err := lim.LoginOK("admin@example.com")
		assert.DeepEqual(t, err, OverLimitError{
			RetryAfter: expected,
		}, opt.DurationWithThreshold(time.Second))
	})

	t.Run("reset limit", func(t *testing.T) {
		_, lim := setup(t)

		for i := 0; i < 10; i++ {
			lim.LoginBad("admin@example.com", 10)
		}

		expected, _ := time.ParseDuration("57s")
		err := lim.LoginOK("admin@example.com")
		assert.DeepEqual(t, err, OverLimitError{
			RetryAfter: expected,
		}, opt.DurationWithThreshold(time.Second))

		lim.LoginGood("admin@example.com")
		err = lim.LoginOK("admin@example.com")
		assert.NilError(t, err)
	})

	t.Run("over limit reset after lockout period", func(t *testing.T) {
		srv, lim := setup(t)

		for i := 0; i < 10; i++ {
			lim.LoginBad("admin@example.com", 10)
		}

		expected, _ := time.ParseDuration("57s")
		err := lim.LoginOK("admin@example.com")
		assert.DeepEqual(t, err, OverLimitError{
			RetryAfter: expected,
		}, opt.DurationWithThreshold(time.Second))

		srv.FastForward(time.Minute)

		err = lim.LoginOK("admin@example.com")
		assert.NilError(t, err)
	})

	t.Run("failed after lockout period", func(t *testing.T) {
		srv, lim := setup(t)

		for i := 0; i < 10; i++ {
			lim.LoginBad("admin@example.com", 10)
		}

		expected, _ := time.ParseDuration("57s")
		err := lim.LoginOK("admin@example.com")
		assert.DeepEqual(t, err, OverLimitError{
			RetryAfter: expected,
		}, opt.DurationWithThreshold(time.Second))

		srv.FastForward(time.Minute)

		err = lim.LoginOK("admin@example.com")
		assert.NilError(t, err)

		lim.LoginBad("admin@example.com", 10)

		expected, _ = time.ParseDuration("1m26s")
		err = lim.LoginOK("admin@example.com")
		assert.DeepEqual(t, err, OverLimitError{
			RetryAfter: expected,
		}, opt.DurationWithThreshold(time.Second))
	})
}
