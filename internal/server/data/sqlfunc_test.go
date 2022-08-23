package data

import (
	"fmt"
	"testing"

	"gotest.tools/v3/assert"
)

func TestSQLUidStrToIntRoundTrip(t *testing.T) {
	type testCase struct {
		base58 string
		intval int64
		err    string
	}
	db := setupDB(t, postgresDriver(t))

	run := func(t *testing.T, tc testCase) {
		var i int64
		err := db.Raw("select uidStrToInt(?);", tc.base58).Scan(&i).Error
		if err != nil {
			if tc.err != "" {
				assert.ErrorContains(t, err, tc.err)
			} else {
				t.Error(err)
			}
		} else {
			assert.Assert(t, tc.err == "", fmt.Sprintf("expected err %q but there was none", tc.err))
		}

		assert.Equal(t, tc.intval, i, "expected result to be %d, but it was %d", tc.intval, i)

		if tc.intval == 0 && tc.base58 != "" {
			return
		}
		var s string
		err = db.Raw("select uidIntToStr(?);", tc.intval).Scan(&s).Error
		if err != nil {
			if tc.err != "" {
				assert.ErrorContains(t, err, tc.err)
			} else {
				t.Error(err)
			}
		} else {
			assert.Assert(t, tc.err == "", fmt.Sprintf("expected err %q but there was none. result was %q", tc.err, s))
		}

		assert.Equal(t, tc.base58, s, fmt.Sprintf("expected result to be %q, but it was %q", tc.base58, s))
	}

	testCases := []testCase{
		{
			base58: "",
			intval: 0,
		},
		{
			base58: "TX",
			intval: 0xbc5,
		},
		{
			base58: "npL6MjP8Qfc", // 0x7fffffffffffffff
			intval: 0x7fffffffffffffff,
		},
		{
			base58: "npL6MjP8Qfd", // 0x7fffffffffffffff + 1
			err:    `invalid base58: value too large`,
		},
		{
			base58: "JPwcyDCgEuqJPwcyDCgEuq",
			err:    `invalid base58: too long`,
		},
		{
			base58: "JPwcyDCgEuq", // 0xffffffffffffffff + 1
			err:    `invalid base58: value too large`,
		},
		{
			base58: "self",
			err:    `invalid base58: byte 2 is out of range`,
		},
		{
			base58: "4jgmnx8Js8A",
			intval: 1428076403798048768,
		},
		{
			base58: "0jgmnx8Js8A",
			err:    `invalid base58: byte 0 is out of range`,
		},
		{
			base58: "jgmnxI8Js8A",
			err:    `invalid base58: byte 5 is out of range`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.base58, func(t *testing.T) {
			run(t, tc)
		})
	}
}
