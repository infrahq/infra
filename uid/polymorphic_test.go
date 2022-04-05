package uid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPolyMorphicIDToSnowflakeID(t *testing.T) {
	id := New()
	iPID := NewIdentityPolymorphicID(id)
	fromIdentityPID, err := iPID.ID()
	assert.NoError(t, err)

	assert.Equal(t, id, fromIdentityPID)

	uID := New()
	uPID := NewIdentityPolymorphicID(uID)
	fromUserPID, err := uPID.ID()
	assert.NoError(t, err)

	assert.Equal(t, uID, fromUserPID)

	gID := New()
	gPID := NewGroupPolymorphicID(gID)
	fromGroupPID, err := gPID.ID()
	assert.NoError(t, err)

	assert.Equal(t, gID, fromGroupPID)
}
