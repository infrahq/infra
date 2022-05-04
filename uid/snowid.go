package uid

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
)

var (
	// epoch is the baseline date used for generating IDs.
	epoch = time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	// nodeBits holds the number of bits to use for Node
	// Remember, you have a total 22 bits to share between Node/Step
	nodeBits uint8 = 10

	// stepBits holds the number of bits to use for Step
	// Remember, you have a total 22 bits to share between Node/Step
	stepBits uint8 = 12
)

const encodeBase58Map = "123456789abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ"

var decodeBase58Map [256]byte

// initialize mapping for decoding.
func init() {
	for i := 0; i < len(decodeBase58Map); i++ {
		decodeBase58Map[i] = 0xFF
	}

	for i := 0; i < len(encodeBase58Map); i++ {
		decodeBase58Map[encodeBase58Map[i]] = byte(i)
	}
}

// A Node struct holds the basic information needed for a snowflake generator
// node
type Node struct {
	mu    sync.Mutex
	epoch time.Time
	time  int64
	node  int64
	step  int64

	nodeMax   int64
	nodeMask  int64
	stepMask  int64
	timeShift uint8
	nodeShift uint8
}

// An ID is a custom type used for a snowflake ID.  This is used so we can
// attach methods onto the ID.
type ID int64

// NewNode returns a new snowflake node that can be used to generate snowflake
// IDs
func NewNode(node int64) (*Node, error) {
	n := Node{}
	n.node = node
	n.nodeMax = -1 ^ (-1 << nodeBits)
	n.nodeMask = n.nodeMax << stepBits
	n.stepMask = -1 ^ (-1 << stepBits)
	n.timeShift = nodeBits + stepBits
	n.nodeShift = stepBits

	if n.node < 0 || n.node > n.nodeMax {
		return nil, errors.New("Node number must be between 0 and " + strconv.FormatInt(n.nodeMax, 10))
	}

	var curTime = time.Now()
	// add time.Duration to curTime to make sure we use the monotonic clock if available
	n.epoch = curTime.Add(time.Unix(epoch/1000, (epoch%1000)*1000000).Sub(curTime))

	return &n, nil
}

// Generate creates and returns a unique snowflake ID
// To help guarantee uniqueness
// - Make sure your system is keeping accurate system time
// - Make sure you never have multiple nodes running with the same node ID
func (n *Node) Generate() ID {
	n.mu.Lock()

	now := time.Since(n.epoch).Nanoseconds() / 1000000

	if now == n.time {
		n.step = (n.step + 1) & n.stepMask

		if n.step == 0 {
			for now <= n.time {
				now = time.Since(n.epoch).Nanoseconds() / 1000000
			}
		}
	} else {
		n.step = 0
	}

	n.time = now

	r := ID((now)<<n.timeShift |
		(n.node << n.nodeShift) |
		(n.step),
	)

	n.mu.Unlock()
	return r
}

// Bytes returns a byte array of the base58 encoded value for this ID.
func (id ID) Bytes() []byte {
	switch {
	case id <= 0:
		return nil
	case id < 58:
		return []byte{encodeBase58Map[id]}
	}

	b := make([]byte, 0, 11)
	for id >= 58 {
		b = append(b, encodeBase58Map[id%58])
		id /= 58
	}
	b = append(b, encodeBase58Map[id])

	for x, y := 0, len(b)-1; x < y; x, y = x+1, y-1 {
		b[x], b[y] = b[y], b[x]
	}

	return b
}

// String returns the base58 encoded value for this ID.
func (id ID) String() string {
	return string(id.Bytes())
}

// Parse parses a base58 encoded value into an ID.
func Parse(b []byte) (ID, error) {
	switch {
	case bytes.HasPrefix(b, []byte("1")):
		return -1, fmt.Errorf("invalid base58: not in canonical form")
	case len(b) > 11:
		return -1, fmt.Errorf("invalid base58: too long")
	}

	var id int64
	for i := range b {
		if decodeBase58Map[b[i]] == 0xFF {
			return -1, fmt.Errorf("invalid base58: byte %d is out of range", i)
		}

		shifted, ok := multiplyCheckOverflow(id, 58)
		if !ok {
			return -1, fmt.Errorf("invalid base58: value too large")
		}
		id = shifted + int64(decodeBase58Map[b[i]])
		if id <= 0 {
			return -1, fmt.Errorf("invalid base58: value too large")
		}
	}

	return ID(id), nil
}

// multiplyCheckOverflow returns the result of a*b, and if the operation
// caused an overflow. Modified from
// https://groups.google.com/g/golang-nuts/c/h5oSN5t3Au4/m/KaNQREhZh0QJ
func multiplyCheckOverflow(a, b int64) (int64, bool) {
	if a == 0 || b == 0 || a == 1 || b == 1 {
		return a * b, true
	}
	total := a * b
	return total, total/b == a
}

func (id ID) MarshalText() ([]byte, error) {
	if int64(id) < 0 {
		return nil, fmt.Errorf("invalid base58: negative value")
	}
	return id.Bytes(), nil
}

func (id *ID) UnmarshalText(b []byte) error {
	var err error
	*id, err = Parse(b)
	return err
}
