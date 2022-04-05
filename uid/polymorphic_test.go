package uid

import (
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestPolyMorphicIDToSnowflakeID(t *testing.T) {
	id := New()
	iPID := NewIdentityPolymorphicID(id)
	fromIdentityPID, err := iPID.ID()
	assert.Check(t, err)

	assert.Check(t, is.Equal(id, fromIdentityPID))

	uID := New()
	uPID := NewIdentityPolymorphicID(uID)
	fromUserPID, err := uPID.ID()
	assert.Check(t, err)

	assert.Check(t, is.Equal(uID, fromUserPID))

	gID := New()
	gPID := NewGroupPolymorphicID(gID)
	fromGroupPID, err := gPID.ID()
	assert.Check(t, err)

	assert.Check(t, is.Equal(gID, fromGroupPID))
}
