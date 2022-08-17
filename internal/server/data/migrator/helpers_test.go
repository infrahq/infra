package migrator

import (
	"testing"

	"gorm.io/gorm"
	"gotest.tools/v3/assert"
)

func setupExampleTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	if db.Dialector.Name() == "sqlite" {
		t.Skip("does not work with sqlite")
	}

	db.Exec("DROP TABLE example")

	var exampleTable = `
CREATE TABLE example (
    id bigint,
    value text
);
ALTER TABLE example ADD CONSTRAINT example_pkey PRIMARY KEY (id);
`
	err := db.Exec(exampleTable).Error
	assert.NilError(t, err)
	t.Cleanup(func() {
		db.Exec("DROP TABLE example")
	})
}

func TestHasTable(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		setupExampleTable(t, db)

		assert.Assert(t, HasTable(db, "example"))
		assert.Assert(t, !HasTable(db, "nope"))
	})
}

func TestHasColumn(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		setupExampleTable(t, db)

		assert.Assert(t, HasColumn(db, "example", "id"))
		assert.Assert(t, HasColumn(db, "example", "value"))
		assert.Assert(t, !HasColumn(db, "example", "other"))
		assert.Assert(t, !HasColumn(db, "missing", "id"))
	})
}

func TestHasConstraint(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		setupExampleTable(t, db)

		assert.Assert(t, HasConstraint(db, "example", "example_pkey"))
		assert.Assert(t, !HasConstraint(db, "example", "other_pkey"))
		assert.Assert(t, !HasConstraint(db, "other", "example_pkey"))
	})
}
