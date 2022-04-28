package uid_test

import (
	"encoding/json"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/uid"
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

func TestParse(t *testing.T) {
	type testCase struct {
		id          string
		expected    uid.ID
		expectedErr string
	}

	run := func(t *testing.T, tc testCase) {
		actual, err := uid.Parse([]byte(tc.id))
		if tc.expectedErr != "" {
			assert.ErrorContains(t, err, tc.expectedErr, "got ID int64=%x", int64(actual))
			return
		}

		assert.NilError(t, err)
		assert.Equal(t, actual, tc.expected, "int64=%x", int64(actual))
	}

	testCases := []testCase{
		{
			id:       "npL6MjP8Qfc", // 0x7fffffffffffffff
			expected: uid.ID(0x7fffffffffffffff),
		},
		{
			id:          "npL6MjP8Qfd", // 0x7fffffffffffffff + 1
			expectedErr: `invalid base58 id "npL6MjP8Qfd"`,
		},
		{
			id:          "JPwcyDCgEuqJPwcyDCgEuq",
			expectedErr: `invalid base58 id "JPwcyDCgEuqJPwcyDCgEuq"`,
		},
		// Does not result in an error, but probably should.
		// Requires: https://github.com/bwmarrin/snowflake/pull/45
		// {
		//	 id:          "JPwcyDCgEuq", //0xffffffffffffffff + 1
		//	 expectedErr: `invalid base58 id "JPwcyDCgEuq"`,
		// },
		{
			id:          "self",
			expectedErr: `invalid base58 id "self"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.id, func(t *testing.T) {
			run(t, tc)
		})
	}
}
