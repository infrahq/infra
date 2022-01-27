package uid_test

import (
	"encoding/json"
	"testing"

	"github.com/infrahq/infra/uid"
	"github.com/stretchr/testify/require"
)

func TestJSONCanUnmarshal(t *testing.T) {
	obj := struct {
		ID uid.ID
	}{}

	newID := uid.New()

	source := []byte(`{"id": "` + newID.String() + `"}`)

	err := json.Unmarshal(source, &obj)
	require.NoError(t, err)

	require.Equal(t, newID, obj.ID)
}
