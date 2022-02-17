package uid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPolyMorphicIDToSnowflakeID(t *testing.T) {
	mID := New()
	mPID := NewMachinePolymorphicID(mID)
	fromMachinePID, err := mPID.ID()
	assert.NoError(t, err)

	assert.Equal(t, mID, fromMachinePID)

	uID := New()
	uPID := NewUserPolymorphicID(uID)
	fromUserPID, err := uPID.ID()
	assert.NoError(t, err)

	assert.Equal(t, uID, fromUserPID)

	gID := New()
	gPID := NewGroupPolymorphicID(gID)
	fromGroupPID, err := gPID.ID()
	assert.NoError(t, err)

	assert.Equal(t, gID, fromGroupPID)
}
