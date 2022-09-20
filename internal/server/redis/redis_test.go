package redis

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"gotest.tools/v3/assert"
)

func TestRedis(t *testing.T) {
	setup := func(t *testing.T) (*miniredis.Miniredis, string, int) {
		redis := miniredis.RunT(t)
		host := redis.Host()
		port, err := strconv.Atoi(redis.Port())
		assert.NilError(t, err)
		return redis, host, port
	}

	t.Run("no redis", func(t *testing.T) {
		redis := NewRedis(Options{})
		assert.Assert(t, redis.client == nil)
	})

	t.Run("with host port", func(t *testing.T) {
		_, host, port := setup(t)
		redis := NewRedis(Options{Host: host, Port: port})
		assert.Assert(t, redis.client != nil)

		pong, err := redis.client.Ping(context.TODO()).Result()
		assert.NilError(t, err)
		assert.Equal(t, pong, "PONG")
	})

	t.Run("with auth", func(t *testing.T) {
		srv, host, port := setup(t)
		srv.RequireAuth("mypassword")
		t.Cleanup(func() {
			srv.RequireAuth("")
		})

		nopass := NewRedis(Options{Host: host, Port: port})
		assert.Assert(t, nopass.client != nil)

		_, err := nopass.client.Ping(context.TODO()).Result()
		assert.ErrorContains(t, err, "NOAUTH Authentication required.")

		redis := NewRedis(Options{Host: host, Port: port, Password: "mypassword"})
		assert.Assert(t, redis.client != nil)

		pong, err := redis.client.Ping(context.TODO()).Result()
		assert.NilError(t, err)
		assert.Equal(t, pong, "PONG")
	})

	t.Run("with user auth", func(t *testing.T) {
		srv, host, port := setup(t)
		srv.RequireUserAuth("myuser", "mypassword")
		t.Cleanup(func() {
			srv.RequireUserAuth("myuser", "")
		})

		nouserpass := NewRedis(Options{Host: host, Port: port})
		assert.Assert(t, nouserpass.client != nil)

		_, err := nouserpass.client.Ping(context.TODO()).Result()
		assert.ErrorContains(t, err, "NOAUTH Authentication required.")

		nopass := NewRedis(Options{Host: host, Port: port, Username: "myuser"})
		assert.Assert(t, nopass.client != nil)

		_, err = nopass.client.Ping(context.TODO()).Result()
		assert.ErrorContains(t, err, "NOAUTH Authentication required.")

		redis := NewRedis(Options{Host: host, Port: port, Username: "myuser", Password: "mypassword"})
		assert.Assert(t, redis.client != nil)

		pong, err := redis.client.Ping(context.TODO()).Result()
		assert.NilError(t, err)
		assert.Equal(t, pong, "PONG")
	})

	t.Run("with db", func(t *testing.T) {
		srv, host, port := setup(t)
		redis := NewRedis(Options{Host: host, Port: port, Options: "db=1"})
		assert.Assert(t, redis.client != nil)

		set := redis.client.Set(context.TODO(), "foo", "bar", time.Minute)
		assert.NilError(t, set.Err())

		assert.Assert(t, !srv.Exists("foo"))

		db1 := srv.DB(1)
		foo1, err := db1.Get("foo")
		assert.NilError(t, err)
		assert.Equal(t, foo1, "bar")
	})
}
