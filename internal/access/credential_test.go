package access

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func TestCreateCredential(t *testing.T) {
	c, db, _ := setupAccessTestContext(t)

	username := "bruce@example.com"
	user := &models.Identity{Name: username}
	err := data.CreateIdentity(db, user)
	assert.NilError(t, err)

	oneTimePassword, err := CreateCredential(c, *user)
	assert.NilError(t, err)
	assert.Assert(t, oneTimePassword != "")

	creds, err := data.GetCredential(db, data.ByIdentityID(user.ID))
	assert.NilError(t, err)

	assert.Equal(t, creds.OneTimePasswordUsed, false)
}

func TestUpdateCredentials(t *testing.T) {
	c, db, _ := setupAccessTestContext(t)

	username := "bruce@example.com"
	user := &models.Identity{Name: username}
	err := data.CreateIdentity(db, user)
	assert.NilError(t, err)

	_, err = CreateCredential(c, *user)
	assert.NilError(t, err)

	t.Run("Update user credentials IS single use password", func(t *testing.T) {
		err := UpdateCredential(c, user, "newPassword")
		assert.NilError(t, err)

		creds, err := data.GetCredential(db, data.ByIdentityID(user.ID))
		assert.NilError(t, err)
		assert.Equal(t, creds.OneTimePassword, true)
	})

	t.Run("Update own credentials is NOT single use password", func(t *testing.T) {
		c.Set("identity", user)

		err := UpdateCredential(c, user, "newPassword")
		assert.NilError(t, err)

		creds, err := data.GetCredential(db, data.ByIdentityID(user.ID))
		assert.NilError(t, err)
		assert.Equal(t, creds.OneTimePassword, false)
	})
}

func TestLowercaseRequirements(t *testing.T) {
	result := lowercaseCheck(2, "a")
	assert.Equal(t, result, false)

	result = lowercaseCheck(2, "A")
	assert.Equal(t, result, false)

	result = lowercaseCheck(2, "ab")
	assert.Equal(t, result, true)

	result = lowercaseCheck(2, "AB")
	assert.Equal(t, result, false)

	result = lowercaseCheck(2, "Ab")
	assert.Equal(t, result, false)

	result = lowercaseCheck(2, "abc")
	assert.Equal(t, result, true)

	result = lowercaseCheck(2, "abC")
	assert.Equal(t, result, true)

	result = lowercaseCheck(2, "AbC")
	assert.Equal(t, result, false)

	result = lowercaseCheck(2, "aBc")
	assert.Equal(t, result, true)

	result = lowercaseCheck(2, "")
	assert.Equal(t, result, false)

	result = lowercaseCheck(2, "!$!@#23")
	assert.Equal(t, result, false)
}

func TestUppercaseRequirements(t *testing.T) {
	result := uppercaseCheck(2, "a")
	assert.Equal(t, result, false)

	result = uppercaseCheck(2, "A")
	assert.Equal(t, result, false)

	result = uppercaseCheck(2, "ab")
	assert.Equal(t, result, false)

	result = uppercaseCheck(2, "AB")
	assert.Equal(t, result, true)

	result = uppercaseCheck(2, "Ab")
	assert.Equal(t, result, false)

	result = uppercaseCheck(2, "abc")
	assert.Equal(t, result, false)

	result = uppercaseCheck(2, "abC")
	assert.Equal(t, result, false)

	result = uppercaseCheck(2, "AbC")
	assert.Equal(t, result, true)

	result = uppercaseCheck(2, "aBc")
	assert.Equal(t, result, false)

	result = uppercaseCheck(2, "")
	assert.Equal(t, result, false)

	result = uppercaseCheck(2, "!$!@#23")
	assert.Equal(t, result, false)
}

func TestNumberRequirements(t *testing.T) {
	result := numberCheck(2, "abc")
	assert.Equal(t, result, false)

	result = numberCheck(2, "aBc")
	assert.Equal(t, result, false)

	result = numberCheck(2, "")
	assert.Equal(t, result, false)

	result = numberCheck(2, "!$!@#")
	assert.Equal(t, result, false)

	result = numberCheck(2, "!$!@#23")
	assert.Equal(t, result, true)

	result = numberCheck(2, "!$!@#23123")
	assert.Equal(t, result, true)
}

func TestSymbolRequirements(t *testing.T) {
	result := symbolCheck(2, "")
	assert.Equal(t, result, false)

	result = symbolCheck(2, "abAB")
	assert.Equal(t, result, false)

	result = symbolCheck(2, "abc!")
	assert.Equal(t, result, false)

	result = symbolCheck(2, "  ")
	assert.Equal(t, result, true)

	result = symbolCheck(2, "!!")
	assert.Equal(t, result, true)

	result = symbolCheck(2, `""`)
	assert.Equal(t, result, true)

	result = symbolCheck(2, `##`)
	assert.Equal(t, result, true)

	result = symbolCheck(2, `$$`)
	assert.Equal(t, result, true)

	result = symbolCheck(2, `%%`)
	assert.Equal(t, result, true)

	result = symbolCheck(2, "&&")
	assert.Equal(t, result, true)

	result = symbolCheck(2, "''")
	assert.Equal(t, result, true)

	result = symbolCheck(2, "((")
	assert.Equal(t, result, true)

	result = symbolCheck(2, "))")
	assert.Equal(t, result, true)

	result = symbolCheck(2, "**")
	assert.Equal(t, result, true)

	result = symbolCheck(2, "++")
	assert.Equal(t, result, true)

	result = symbolCheck(2, ",,")
	assert.Equal(t, result, true)

	result = symbolCheck(2, "--")
	assert.Equal(t, result, true)

	result = symbolCheck(2, "..")
	assert.Equal(t, result, true)

	result = symbolCheck(2, "))")
	assert.Equal(t, result, true)

	result = symbolCheck(2, "//")
	assert.Equal(t, result, true)

	result = symbolCheck(2, "::")
	assert.Equal(t, result, true)

	result = symbolCheck(2, ";;")
	assert.Equal(t, result, true)

	result = symbolCheck(2, "<<")
	assert.Equal(t, result, true)

	result = symbolCheck(2, "==")
	assert.Equal(t, result, true)

	result = symbolCheck(2, ">>")
	assert.Equal(t, result, true)

	result = symbolCheck(2, "??")
	assert.Equal(t, result, true)

	result = symbolCheck(2, "@@")
	assert.Equal(t, result, true)

	result = symbolCheck(2, "^^")
	assert.Equal(t, result, true)

	result = symbolCheck(2, "__")
	assert.Equal(t, result, true)

	result = symbolCheck(2, "{{")
	assert.Equal(t, result, true)

	result = symbolCheck(2, "}}")
	assert.Equal(t, result, true)

	result = symbolCheck(2, "||")
	assert.Equal(t, result, true)

	result = symbolCheck(2, "~~")
	assert.Equal(t, result, true)

	result = symbolCheck(2, "~~")
	assert.Equal(t, result, true)

	result = symbolCheck(2, `//`)
	assert.Equal(t, result, true)

	result = symbolCheck(2, `\\`)
	assert.Equal(t, result, true)

	result = symbolCheck(2, `[[`)
	assert.Equal(t, result, true)

	result = symbolCheck(2, `]]`)
	assert.Equal(t, result, true)

	result = symbolCheck(2, `@$%@#ss`)
	assert.Equal(t, result, true)
}
