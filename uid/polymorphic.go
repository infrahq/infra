package uid

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
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

func (p PolymorphicID) DescribeSchema(schema *openapi3.Schema) {
	schema.Type = "string"
	schema.Format = "poly-uid"
	schema.Pattern = `\w:[1-9a-km-zA-HJ-NP-Z]{1,11}`
	schema.Example = "i:4yJ3n3D8E3"
}

func (p PolymorphicID) IsIdentity() bool {
	return strings.HasPrefix(string(p), "i:")
}

func (p PolymorphicID) IsGroup() bool {
	return strings.HasPrefix(string(p), "g:")
}

func NewPolymorphicID(prefix string, id ID) PolymorphicID {
	return PolymorphicID(fmt.Sprintf("%s:%s", prefix, id))
}
