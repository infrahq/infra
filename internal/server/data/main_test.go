package data

import (
	"context"
	"fmt"
	"os"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/testing/database"
)

var dbBufferPool *dbBuffer

func TestMain(m *testing.M) {
	models.SkipSymmetricKey = true
	ctx, cancel := context.WithCancel(context.Background())

	dbBufferPool = &dbBuffer{
		items: make(chan dbBufferItem, 10),
		done:  make(chan struct{}),
	}
	go dbBufferPool.fill(ctx)

	code := m.Run()
	cancel()
	<-dbBufferPool.done // wait on cleanup
	os.Exit(code)
}

func setupDB(t *testing.T) *DB {
	t.Helper()
	return dbBufferPool.Get(t)
}

type dbBuffer struct {
	items chan dbBufferItem
	done  chan struct{}
}

type dbBufferItem struct {
	db    *DB
	mainT *database.MainT
}

func (b *dbBuffer) Get(t *testing.T) *DB {
	item := <-b.items
	t.Cleanup(func() {
		item.mainT.RunCleanup()
	})
	return item.db
}

func (b *dbBuffer) fill(ctx context.Context) {
	var count int
	for {
		mainT := &database.MainT{}
		schema := fmt.Sprintf("_data%d", count)
		count++
		db, err := NewDB(NewDBOptions{DSN: database.PostgresDriver(mainT, schema).DSN})
		assert.NilError(mainT, err)

		select {
		case b.items <- dbBufferItem{db: db, mainT: mainT}:
			fmt.Printf("BUF: adding item %d\n", count)
		case <-ctx.Done():
			close(b.items)

			mainT.RunCleanup()
			for item := range b.items {
				item.mainT.RunCleanup()
			}
			close(b.done)
			return
		}
	}
}
