package access

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/infrahq/infra/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupDB(t *testing.T) *gorm.DB {
	driver, err := data.NewSQLiteDriver("file::memory:")
	require.NoError(t, err)

	db, err := data.NewDB(driver)
	require.NoError(t, err)

	return db
}

func TestRequireAuthorization(t *testing.T) {
	cases := []struct {
		Name                string
		RequiredPermissions []Permission
		AuthFunc            func(t *testing.T, db *gorm.DB, c *gin.Context)
		VerifyFunc          func(t *testing.T, err error)
	}{
		{
			Name:                "AuthorizedAll",
			RequiredPermissions: []Permission{PermissionUserRead},
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(PermissionAll))
			},
			VerifyFunc: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			Name:                "AuthorizedAllInfra",
			RequiredPermissions: []Permission{PermissionUserRead},
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(PermissionAllInfra))
			},
			VerifyFunc: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			Name:                "AuthorizedExactMatch",
			RequiredPermissions: []Permission{PermissionUserRead},
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(PermissionUserRead))
			},
			VerifyFunc: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			Name:                "AuthorizedOneOfMany",
			RequiredPermissions: []Permission{PermissionUserRead},
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				permissions := []string{string(PermissionUserRead), string(PermissionUserCreate), string(PermissionUserDelete)}
				c.Set("permissions", strings.Join(permissions, " "))
			},
			VerifyFunc: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			Name:                "AuthorizedWildcardAction",
			RequiredPermissions: []Permission{PermissionUserRead},
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(PermissionUser))
			},
			VerifyFunc: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			Name:                "AuthorizedWildcardResource",
			RequiredPermissions: []Permission{PermissionUserRead},
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(PermissionAllRead))
			},
			VerifyFunc: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			Name:                "APITokenAuthorizedNotFirst",
			RequiredPermissions: []Permission{PermissionUserRead},
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				permissions := []string{string(PermissionGroupRead), string(PermissionProviderRead), string(PermissionUserRead)}
				c.Set("permissions", strings.Join(permissions, " "))
			},
			VerifyFunc: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			Name:                "APITokenAuthorizedNotLast",
			RequiredPermissions: []Permission{PermissionUserRead},
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				permissions := []string{string(PermissionGroupRead), string(PermissionUserRead), string(PermissionProviderRead)}
				c.Set("permissions", strings.Join(permissions, " "))
			},
			VerifyFunc: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			Name:                "APITokenAuthorizedNoMatch",
			RequiredPermissions: []Permission{PermissionUserRead},
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				permissions := []string{string(PermissionUserCreate), string(PermissionGroupRead)}
				c.Set("permissions", strings.Join(permissions, " "))
			},
			VerifyFunc: func(t *testing.T, err error) {
				require.EqualError(t, err, `missing permission "infra.user.read": forbidden`)
			},
		},
		{
			Name:                "NotRequired",
			RequiredPermissions: []Permission{},
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
			},
			VerifyFunc: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}

	for _, test := range cases {
		t.Run(test.Name, func(t *testing.T) {
			db := setupDB(t)

			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Set("db", db)

			test.AuthFunc(t, db, c)

			_, err := requireAuthorization(c, test.RequiredPermissions...)
			test.VerifyFunc(t, err)
		})
	}
}

func TestRequireAuthorizationWithCheck(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		Name             string
		CurrentUser      *models.User
		PermissionWanted Permission
		CustomCheckFn    func(cUser *models.User) bool
		ErrorExpected    error
	}{
		{
			Name:             "wrong user with no permissions",
			CurrentUser:      &models.User{},
			PermissionWanted: PermissionDestinationRead,
			CustomCheckFn:    func(cUser *models.User) bool { return false },
			ErrorExpected:    internal.ErrForbidden,
		},
		{
			Name:             "right user with no permissions",
			CurrentUser:      &models.User{Model: models.Model{ID: userID}},
			PermissionWanted: PermissionDestinationRead,
			CustomCheckFn:    func(cUser *models.User) bool { return cUser.ID == userID },
			ErrorExpected:    nil,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			db := setupDB(t)
			c, _ := gin.CreateTestContext(nil)
			c.Set("db", db)
			c.Set("permissions", test.CurrentUser.Permissions)
			c.Set("user", test.CurrentUser)
			c.Set("user_id", test.CurrentUser.ID)

			_, err := requireAuthorizationWithCheck(c, test.PermissionWanted, test.CustomCheckFn)

			if test.ErrorExpected != nil {
				require.ErrorIs(t, err, test.ErrorExpected)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAllRequired(t *testing.T) {
	cases := map[string]map[string]interface{}{
		"ExactMatch": {
			"permissions": []string{string(PermissionUserRead)},
			"required":    []string{string(PermissionUserRead)},
			"expected":    true,
		},
		"SubsetMatch": {
			"permissions": []string{string(PermissionUserCreate), string(PermissionUserRead)},
			"required":    []string{string(PermissionUserRead)},
			"expected":    true,
		},
		"NoMatch": {
			"permissions": []string{string(PermissionUserCreate), string(PermissionUserRead)},
			"required":    []string{string(PermissionUserDelete)},
			"expected":    false,
		},
		"NoPermissions": {
			"permissions": []string{},
			"required":    []string{string(PermissionUserDelete)},
			"expected":    false,
		},
		"AllPermissions": {
			"permissions": []string{string(PermissionAll)},
			"required":    []string{string(PermissionUserDelete)},
			"expected":    true,
		},
		"AllPermissionsAlternate": {
			"permissions": []string{string(PermissionAllInfra)},
			"required":    []string{string(PermissionUserDelete)},
			"expected":    true,
		},
		"AllPermissionsForResource": {
			"permissions": []string{string(PermissionUser)},
			"required":    []string{string(PermissionUserDelete)},
			"expected":    true,
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			permissions, ok := v["permissions"].([]string)
			require.True(t, ok)

			required, ok := v["required"].([]string)
			require.True(t, ok)

			result := AllRequired(permissions, required)

			expected, ok := v["expected"].(bool)
			require.True(t, ok)

			assert.Equal(t, expected, result)
		})
	}
}
