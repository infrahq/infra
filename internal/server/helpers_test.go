package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	gocmp "github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type organizationData struct {
	Organization   *models.Organization
	Admin          *models.Identity
	AdminAccessKey string
}

// createOtherOrg creates an organization with domain other.example.org, with
// a user, and a grant that makes them an infra admin. It can be used by tests
// to ensure that an API endpoint honors the organization of the user making
// the request.
func createOtherOrg(t *testing.T, db *data.DB) organizationData {
	t.Helper()
	otherOrg := &models.Organization{Name: "Other", Domain: "other.example.org"}
	assert.NilError(t, data.CreateOrganization(db, otherOrg))

	tx := txnForTestCase(t, db, otherOrg.ID)
	admin := createAdmin(t, tx)

	token := &models.AccessKey{
		IssuedFor:  admin.ID,
		ProviderID: data.InfraProvider(tx).ID,
		ExpiresAt:  time.Now().Add(1000 * time.Second),
	}

	accessKey, err := data.CreateAccessKey(tx, token)
	assert.NilError(t, err)

	assert.NilError(t, tx.Commit())
	return organizationData{
		Organization:   otherOrg,
		Admin:          admin,
		AdminAccessKey: accessKey,
	}
}

func adminAccessKey(s *Server) string {
	for _, id := range s.options.Users {
		if id.Name == "admin@example.com" {
			return id.AccessKey
		}
	}

	return ""
}

// withAdminUser may be used with setupServer to setup the server
// with an admin identity and access key
func withAdminUser(_ *testing.T, opts *Options) {
	opts.Users = append(opts.Users, User{
		Name:      "admin@example.com",
		AccessKey: "BlgpvURSGF.NdcemBdzxLTGIcjPXwPoZNrb",
		InfraRole: "admin",
	})
}

func withSupportAdminGrant(_ *testing.T, opts *Options) {
	opts.Users = append(opts.Users, User{
		Name:      "admin@example.com",
		AccessKey: "BlgpvURSGF.NdcemBdzxLTGIcjPXwPoZNrb",
		InfraRole: "support-admin",
	})
}

func withMultiOrgEnabled(_ *testing.T, opts *Options) {
	opts.DefaultOrganizationDomain = "example.com"
	opts.EnableSignup = true
}

func createAdmin(t *testing.T, db data.WriteTxn) *models.Identity {
	user := &models.Identity{
		Name: "admin+" + generate.MathRandom(10, generate.CharsetAlphaNumeric),
	}
	err := data.CreateIdentity(db, user)
	assert.NilError(t, err)

	err = data.CreateGrant(db, &models.Grant{
		Subject:   uid.NewIdentityPolymorphicID(user.ID),
		Resource:  "infra",
		Privilege: models.InfraAdminRole,
	})
	assert.NilError(t, err)

	return user
}

func createAccessKey(t *testing.T, db data.WriteTxn, email string) (string, *models.Identity) {
	t.Helper()
	user := &models.Identity{Name: email}
	err := data.CreateIdentity(db, user)
	assert.NilError(t, err)

	provider := data.InfraProvider(db)

	token := &models.AccessKey{
		IssuedFor:  user.ID,
		ProviderID: provider.ID,
		ExpiresAt:  time.Now().Add(10 * time.Second),
	}

	body, err := data.CreateAccessKey(db, token)
	assert.NilError(t, err)

	return body, user
}

func createIdentities(t *testing.T, db data.WriteTxn, identities ...*models.Identity) {
	t.Helper()
	for i := range identities {
		err := data.CreateIdentity(db, identities[i])
		assert.NilError(t, err, identities[i].Name)
		for _, g := range identities[i].Groups {
			err := data.AddUsersToGroup(db, g.ID, []uid.ID{identities[i].ID})
			assert.NilError(t, err)
		}
		assert.NilError(t, err, identities[i].Name)
	}
}

func createGroups(t *testing.T, db data.WriteTxn, groups ...*models.Group) {
	t.Helper()
	for i := range groups {
		err := data.CreateGroup(db, groups[i])
		assert.NilError(t, err, groups[i].Name)
	}
}

func createUser(t *testing.T, srv *Server, routes Routes, email string) *api.CreateUserResponse {
	r := &api.CreateUserRequest{
		Name: email,
	}
	body, err := json.Marshal(r)
	assert.NilError(t, err)

	// nolint:noctx
	req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewReader(body))
	req.Header.Add("Authorization", "Bearer "+adminAccessKey(srv))
	req.Header.Add("Infra-Version", "0.14")

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, 201, resp.Code)

	result := &api.CreateUserResponse{}
	err = json.Unmarshal(resp.Body.Bytes(), result)
	assert.NilError(t, err)

	return result
}

func txnForTestCase(t *testing.T, db *data.DB, orgID uid.ID) *data.Transaction {
	t.Helper()
	tx, err := db.Begin(context.Background(), nil)
	assert.NilError(t, err)
	t.Cleanup(func() {
		assert.NilError(t, tx.Rollback())
	})
	return tx.WithOrgID(orgID)
}

func jsonBody(t *testing.T, body interface{}) *bytes.Buffer {
	t.Helper()
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(body)
	assert.NilError(t, err)
	return buf
}

func jsonUnmarshal(t *testing.T, raw string) interface{} {
	t.Helper()
	var out interface{}
	err := json.Unmarshal([]byte(raw), &out)
	assert.NilError(t, err, "failed to decode JSON")
	return out
}

// cmpApproximateTime is a gocmp.Option that compares a time formatted as an
// RFC3339 string. The times may be up to 2 seconds different from each other,
// to account for the runtime of a test.
// cmpApproximateTime accepts interface{} instead of time.Time because it is
// intended to be used to compare times in API responses that were decoded
// into an interface{}.
var cmpApproximateTime = gocmp.Comparer(func(x, y interface{}) bool {
	xs, _ := x.(string)
	xd, _ := time.Parse(time.RFC3339, xs)

	ys, _ := y.(string)
	yd, _ := time.Parse(time.RFC3339, ys)

	if xd.After(yd) {
		xd, yd = yd, xd
	}
	return yd.Sub(xd) < 30*time.Second
})

// cmpAnyValidUID is a gocmp.Option that allows a field to match any valid uid.ID,
// as long as the expected value is the literal string "<any-valid-uid>".
// cmpAnyValidUID accepts interface{} instead of string because it is intended
// to be used to compare a UID.ID in API responses that were decoded
// into an interface{}.
var cmpAnyValidUID = gocmp.Comparer(func(x, y interface{}) bool {
	xs, _ := x.(string)
	ys, _ := y.(string)

	if xs == "<any-valid-uid>" {
		_, err := uid.Parse([]byte(ys))
		return err == nil
	}
	if ys == "<any-valid-uid>" {
		_, err := uid.Parse([]byte(xs))
		return err == nil
	}
	return xs == ys
})

// pathMapKey is a gocmp.FilerPath filter that matches map entries with any
// of the keys.
// TODO: allow dotted identifier for keys in nested maps.
func pathMapKey(keys ...string) func(path gocmp.Path) bool {
	return func(path gocmp.Path) bool {
		mapIndex, ok := path.Last().(gocmp.MapIndex)
		if !ok {
			return false
		}

		for _, key := range keys {
			if mapIndex.Key().Interface() == key {
				return true
			}
		}
		return false
	}
}

// cmpAnyString is a gocmp.Option that allows a field to match any non-zero string.
var cmpAnyString = gocmp.Comparer(func(x, y interface{}) bool {
	xs, _ := x.(string)
	ys, _ := y.(string)

	if xs == "" || ys == "" {
		return false
	}
	if xs == "<any-string>" || ys == "<any-string>" {
		return true
	}
	return xs == ys
})

// cmpAnyString is a gocmp.Option that allows a field to match a string with any suffix.
var cmpAnyStringSuffix = gocmp.Comparer(func(x, y interface{}) bool {
	xs, _ := x.(string)
	ys, _ := y.(string)

	switch {
	case strings.HasSuffix(xs, "<any-string>"):
		return strings.HasPrefix(ys, strings.TrimSuffix(xs, "<any-string>"))
	case strings.HasSuffix(ys, "<any-string>"):
		return strings.HasPrefix(xs, strings.TrimSuffix(ys, "<any-string>"))
	}

	return xs == ys
})

// cmpEquateEmptySlice is a gocmp.Option that evalutes any empty slice as equal regardless of their types, then falls back to deep comparison
var cmpEquateEmptySlice = gocmp.Comparer(func(x, y interface{}) bool {
	xs, _ := x.([]any)
	ys, _ := y.([]any)

	if len(xs) == len(ys) {
		return true
	}

	return reflect.DeepEqual(xs, ys)
})

// cmpAnyValidAccessKey is a gocmp.Option that allows a field to match any valid access key
var cmpAnyValidAccessKey = gocmp.Comparer(func(x, y interface{}) bool {
	xs, _ := x.(string)
	ys, _ := y.(string)

	validAccessKey := func(s string) bool {
		key, secret, ok := strings.Cut(s, ".")
		if !ok {
			return false
		}

		return len(key) == 10 && len(secret) == 24
	}

	switch {
	case xs == "<any-valid-access-key>":
		return validAccessKey(ys)
	case ys == "<any-valid-access-key>":
		return validAccessKey(xs)
	}

	return xs == ys
})
