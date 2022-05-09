package uid

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"
)

func TestNewNode(t *testing.T) {
	_, err := NewNode(0)
	assert.NilError(t, err)

	_, err = NewNode(5000)
	assert.ErrorContains(t, err, "number must be between")
}

// lazy check if Generate will create duplicate IDs
// would be good to later enhance this with more smarts
func TestGenerateDuplicateID(t *testing.T) {
	node, _ := NewNode(1)

	var x, y ID
	for i := 0; i < 1000000; i++ {
		y = node.Generate()
		if x == y {
			t.Errorf("x(%d) & y(%d) are the same", x, y)
		}
		x = y
	}
}

func TestNode_Generate_ConcurrencyAndUniqueness(t *testing.T) {
	var nodes = 40
	var iterations = 10000
	if testing.Short() {
		nodes = 10
		iterations = 1000
	}

	// Pick a random set of unique node IDs each time
	nodeMax := -1 ^ (-1 << nodeBits) // modeMax=1024
	nodeIDs := make([]int64, nodeMax)
	for n := 0; n < nodeMax; n++ {
		nodeIDs[n] = int64(n)
	}
	seed := time.Now().UnixNano()
	t.Log("seed", seed)
	// nolint: gosec // does not need to be cryptographically secure
	r := rand.New(rand.NewSource(seed))
	r.Shuffle(nodeMax, func(i, j int) {
		nodeIDs[i], nodeIDs[j] = nodeIDs[j], nodeIDs[i]
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	g, ctx := errgroup.WithContext(ctx)

	all := sync.Map{}
	for n := 0; n < nodes; n++ {
		n := n // capture n
		g.Go(func() error {
			node, err := NewNode(nodeIDs[n])
			if err != nil {
				return err
			}

			for i := 0; i < iterations; i++ {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				id := node.Generate()
				if id < 0 {
					return fmt.Errorf("node %d: %w", n, err)
				}
				all.Store(id, true)
			}
			return nil
		})
	}
	assert.NilError(t, g.Wait())

	var total int
	all.Range(func(key, value any) bool {
		total++
		return true
	})
	assert.Equal(t, total, nodes*iterations)
}

func TestBase58(t *testing.T) {
	assert.Equal(t, len(encodeBase58Map), 58)

	node, err := NewNode(0)
	assert.NilError(t, err)

	for i := 0; i < 10; i++ {
		id := node.Generate()
		base58 := id.String()
		parsed, err := Parse([]byte(base58))
		assert.NilError(t, err)
		assert.Equal(t, id, parsed)
	}
}

func TestID_JSONUnmarshal(t *testing.T) {
	obj := struct {
		ID ID
	}{}

	node, err := NewNode(122)
	assert.NilError(t, err)
	newID := node.Generate()

	source := []byte(`{"id": "` + newID.String() + `"}`)

	err = json.Unmarshal(source, &obj)
	assert.NilError(t, err)

	assert.Equal(t, newID, obj.ID)
}

func BenchmarkParse(b *testing.B) {
	node, _ := NewNode(1)
	sf := node.Generate()
	b58 := sf.String()

	b.ReportAllocs()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		id, err := Parse([]byte(b58))
		assert.NilError(b, err)
		assert.Assert(b, id > 0)
	}
}

func BenchmarkBase58(b *testing.B) {
	node, _ := NewNode(1)
	sf := node.Generate()

	b.ReportAllocs()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = sf.String()
	}
}

func BenchmarkGenerate(b *testing.B) {
	node, _ := NewNode(1)

	b.ReportAllocs()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = node.Generate()
	}
}

func BenchmarkGenerateMaxSequence(b *testing.B) {
	nodeBits = 1
	stepBits = 21
	node, _ := NewNode(1)

	b.ReportAllocs()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = node.Generate()
	}
}

func TestParse(t *testing.T) {
	type testCase struct {
		base58      string
		expected    ID
		expectedErr string
	}

	run := func(t *testing.T, tc testCase) {
		actual, err := Parse([]byte(tc.base58))
		if tc.expectedErr != "" {
			assert.ErrorContains(t, err, tc.expectedErr, "int64=%x", int64(actual))
			return
		}

		assert.NilError(t, err)
		assert.Equal(t, actual, tc.expected, "int64=%x", int64(actual))
	}

	testCases := []testCase{
		{
			base58:   "",
			expected: ID(0),
		},
		{
			base58:   "TX",
			expected: ID(0xbc5),
		},
		{
			base58:   "npL6MjP8Qfc", // 0x7fffffffffffffff
			expected: ID(0x7fffffffffffffff),
		},
		{
			base58:      "npL6MjP8Qfd", // 0x7fffffffffffffff + 1
			expectedErr: `invalid base58: value too large`,
		},
		{
			base58:      "JPwcyDCgEuqJPwcyDCgEuq",
			expectedErr: `invalid base58: too long`,
		},
		{
			base58:      "JPwcyDCgEuq", // 0xffffffffffffffff + 1
			expectedErr: `invalid base58: value too large`,
		},
		{
			base58:      "self",
			expectedErr: `invalid base58: byte 2 is out of range`,
		},
		{
			base58:   "4jgmnx8Js8A",
			expected: 1428076403798048768,
		},
		{
			base58:      "0jgmnx8Js8A",
			expectedErr: `invalid base58: byte 0 is out of range`,
		},
		{
			base58:      "jgmnxI8Js8A",
			expectedErr: `invalid base58: byte 5 is out of range`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.base58, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func FuzzID_Parse_NoPanic(f *testing.F) {
	testCases := []string{
		"self",
		"abcdefghi",
		"123456789",
		"1",
		"gbtNrmnJkvA",
		"11111111111",
	}
	for _, tc := range testCases {
		f.Add(tc)
	}
	f.Fuzz(func(t *testing.T, input string) {
		id, err := Parse([]byte(input))
		if id < 0 {
			assert.ErrorContains(t, err, "invalid base58")
			return
		}
		assert.NilError(t, err)
	})
}

func FuzzID_MarshalText_RoundTrip_FromInt64(f *testing.F) {
	testCases := []int64{
		-1, 0, 1, 2, 10,
		math.MaxInt8, math.MinInt8,
		math.MaxInt16, math.MinInt16,
		math.MaxInt32, math.MinInt32,
		math.MaxInt64, math.MinInt64,
	}
	for _, tc := range testCases {
		f.Add(tc)
	}
	f.Fuzz(func(t *testing.T, original int64) {
		id := ID(original)
		raw, err := id.MarshalText()
		if original < 0 {
			assert.ErrorContains(t, err, "negative value")
			return
		}
		assert.NilError(t, err)

		target := new(ID)
		err = target.UnmarshalText(raw)
		assert.NilError(t, err)

		assert.Equal(t, id, *target)
	})
}

func FuzzID_Parse_RoundTrip_FromString(f *testing.F) {
	testCases := []string{
		"self",
		"abcdefghi",
		"123456789",
		"1",
		"gbtNrmnJkvA",
		"dbtNrmnJkvA",
		"btNrmnJkvA",
		"211111111111",
		"A1111111111",
		"X1111111111",
		"JR111111111",
		"JPwcyDCgEuq",
	}
	for _, tc := range testCases {
		f.Add(tc)
	}
	f.Fuzz(func(t *testing.T, original string) {
		id, err := Parse([]byte(original))
		if shouldError(original) {
			assert.ErrorContains(t, err, "invalid base58", "input=%v", original)
			return
		}
		if id < 0 {
			assert.ErrorContains(t, err, "invalid base58: value too large", "input=%v", original)
			return
		}

		assert.NilError(t, err, "input=%v", original)
		assert.Equal(t, id.String(), original, "int64=%d", id)
	})
}

func shouldError(input string) bool {
	switch {
	case strings.HasPrefix(input, "1"):
		return true
	case len(input) > 11:
		return true
	}
	for i := range input {
		if !strings.Contains(encodeBase58Map, string(input[i])) {
			return true
		}
	}
	return false
}
