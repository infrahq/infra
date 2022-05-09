package uid

import (
	"math/rand"
)

var node *Node

func init() {
	var err error

	//nolint:gosec // do not need cryptographic random value here
	node, err = NewNode(rand.Int63n(1024))
	if err != nil {
		panic(err)
	}
}

// New returns an ID using a random NodeID. The NodeID is selected when the
// process starts, and won't change until the process is restarted.
func New() ID {
	return node.Generate()
}
