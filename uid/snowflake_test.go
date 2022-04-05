package uid_test

import (
	"encoding/json"
	"testing"

	"github.com/infrahq/infra/uid"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestJSONCanUnmarshal(t *testing.T) {
	obj := struct {
		ID uid.ID
	}{}

	newID := uid.New()

	source := []byte(`{"id": "` + newID.String() + `"}`)

	err := json.Unmarshal(source, &obj)
	assert.NilError(t, err)

	assert.Equal(t, newID, obj.ID)
}

func TestBadIDs(t *testing.T) {
	ok := "npL6MjP8Qfc"   // 0x7fffffffffffffff
	bad1 := "npL6MjP8Qfd" // 0x7fffffffffffffff + 1
	// bad2 := "JPwcyDCgEuq" //0xffffffffffffffff + 1
	bad3 := "JPwcyDCgEuqJPwcyDCgEuq"

	id, err := uid.Parse([]byte(ok))
	assert.NilError(t, err)
	assert.Equal(t, 0x7fffffffffffffff, id)

	id, err = uid.Parse([]byte(bad1))
	assert.Assert(t, is.ErrorContains(err, ""))
	assert.Equal(t, 0, id)

	// I think I need to fork snowflake to fix this.
	// id, err = uid.Parse([]byte(bad2))
	// require.Error(t, err)
	// require.EqualValues(t, 0, id)

	id, err = uid.Parse([]byte(bad3))
	t.Log(id)
	assert.Assert(t, is.ErrorContains(err, ""))
	assert.Equal(t, 0, id)
}
