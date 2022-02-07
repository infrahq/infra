package uid

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/bwmarrin/snowflake"
)

type ID snowflake.ID

var idGen *snowflake.Node

func init() {
	snowflake.Epoch = time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	var err error
	idGen, err = snowflake.NewNode(rand.Int63n(1024))
	if err != nil {
		panic(err)
	}

}

func New() ID {
	return ID(idGen.Generate())
}

func (u ID) String() string {
	return snowflake.ID(u).Base58()
}

func Parse(b []byte) (ID, error) {
	if len(b) > 11 {
		return ID(0), fmt.Errorf("invalid id %q", string(b))
	}

	id, err := snowflake.ParseBase58(b)
	if err != nil {
		return ID(0), err
	}

	if id < 0 {
		return ID(0), fmt.Errorf("invalid id %q", string(b))
	}

	return ID(id), nil
}

func ParseString(s string) (ID, error) {
	return Parse([]byte(s))
}

func (u *ID) UnmarshalText(b []byte) error {
	id, err := Parse(b)
	if err != nil {
		return err
	}

	*u = id
	return nil
}

func (u *ID) MarshalText() ([]byte, error) {
	return []byte(u.String()), nil
}
