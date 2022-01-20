package uuid

import (
	"math/rand"
	"time"

	"github.com/bwmarrin/snowflake"
)

type UUID snowflake.ID

var idGen *snowflake.Node

func init() {
	snowflake.Epoch = time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	var err error
	idGen, err = snowflake.NewNode(rand.Int63n(1024))
	if err != nil {
		panic(err)
	}

}

func New() UUID {
	return UUID(idGen.Generate())
}

func (u UUID) String() string {
	return snowflake.ID(u).Base58()
}

func (u *UUID) UnmarshalText(b []byte) error {
	id, err := snowflake.ParseBase58(b)
	if err != nil {
		return err
	}
	*u = UUID(id)
	return nil
}
