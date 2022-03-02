package uid

import (
	"fmt"
	"strings"
)

// PolymorphicID is a reference of the format "u:<idstr>" for users, "m:<idstr>" for machines, and "g:<idstr>" for groups
type PolymorphicID string

func (p PolymorphicID) String() string {
	return string(p)
}

func (p PolymorphicID) ID() (ID, error) {
	id := string(p)[2:]
	return ParseString(id)
}

func (p PolymorphicID) IsMachine() bool {
	return strings.HasPrefix(string(p), "m:")
}

func (p PolymorphicID) IsUser() bool {
	return strings.HasPrefix(string(p), "u:")
}

func (p PolymorphicID) IsGroup() bool {
	return strings.HasPrefix(string(p), "g:")
}

func NewMachinePolymorphicID(id ID) PolymorphicID {
	return newPolymorphicID("m", id)
}

func NewUserPolymorphicID(id ID) PolymorphicID {
	return newPolymorphicID("u", id)
}

func NewGroupPolymorphicID(id ID) PolymorphicID {
	return newPolymorphicID("g", id)
}

func newPolymorphicID(prefix string, id ID) PolymorphicID {
	return PolymorphicID(fmt.Sprintf("%s:%s", prefix, id))
}
