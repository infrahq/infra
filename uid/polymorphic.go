package uid

import (
	"fmt"
	"strings"
)

// PolymorphicID is a reference of the format "i:<idstr>" for identities and "g:<idstr>" for groups
type PolymorphicID string

func (p PolymorphicID) String() string {
	return string(p)
}

func (p PolymorphicID) ID() (ID, error) {
	if len(p) < 2 {
		return ID(0), fmt.Errorf("invalid polymorphic ID encountered: %v", p)
	}
	return Parse([]byte(string(p)[2:]))
}

func (p PolymorphicID) IsIdentity() bool {
	return strings.HasPrefix(string(p), "i:")
}

func (p PolymorphicID) IsGroup() bool {
	return strings.HasPrefix(string(p), "g:")
}

func NewIdentityPolymorphicID(id ID) PolymorphicID {
	return newPolymorphicID("i", id)
}

func NewGroupPolymorphicID(id ID) PolymorphicID {
	return newPolymorphicID("g", id)
}

func newPolymorphicID(prefix string, id ID) PolymorphicID {
	return PolymorphicID(fmt.Sprintf("%s:%s", prefix, id))
}
