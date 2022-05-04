package uid

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestPolyMorphicIDToSnowflakeID(t *testing.T) {
	node, err := NewNode(123)
	assert.NilError(t, err)

	id := node.Generate()
	iPID := NewIdentityPolymorphicID(id)
	fromIdentityPID, err := iPID.ID()
	assert.NilError(t, err)

	assert.Equal(t, id, fromIdentityPID)

	uID := node.Generate()
	uPID := NewIdentityPolymorphicID(uID)
	fromUserPID, err := uPID.ID()
	assert.NilError(t, err)

	assert.Equal(t, uID, fromUserPID)

	gID := node.Generate()
	gPID := NewGroupPolymorphicID(gID)
	fromGroupPID, err := gPID.ID()
	assert.NilError(t, err)

	assert.Equal(t, gID, fromGroupPID)
}
