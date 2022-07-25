package access

import (
	"errors"
	"fmt"
	"regexp"
	"unicode"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/validate"
)

func CreateCredential(c *gin.Context, user models.Identity) (string, error) {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return "", HandleAuthErr(err, "user", "create", models.InfraAdminRole)
	}

	tmpPassword, err := generate.CryptoRandom(12, generate.CharsetPassword)
	if err != nil {
		return "", fmt.Errorf("generate: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(tmpPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash: %w", err)
	}

	userCredential := &models.Credential{
		IdentityID:      user.ID,
		PasswordHash:    hash,
		OneTimePassword: true,
	}

	if err := data.CreateCredential(db, userCredential); err != nil {
		return "", err
	}

	return tmpPassword, nil
}

func UpdateCredential(c *gin.Context, user *models.Identity, newPassword string) error {
	db, err := hasAuthorization(c, user.ID, isIdentitySelf, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "user", "update", models.InfraAdminRole)
	}

	isSelf, err := isIdentitySelf(c, user.ID)
	if err != nil {
		return err
	}

	err = checkPasswordRequirements(db, newPassword)
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash: %w", err)
	}

	userCredential, err := data.GetCredential(db, data.ByIdentityID(user.ID))
	if err != nil {
		if errors.Is(err, internal.ErrNotFound) && !isSelf {
			if err := data.CreateCredential(db, &models.Credential{
				IdentityID:      user.ID,
				PasswordHash:    hash,
				OneTimePassword: true,
			}); err != nil {
				return fmt.Errorf("creating credentials: %w", err)
			}
			return nil
		}
		return fmt.Errorf("existing credential: %w", err)
	}

	userCredential.PasswordHash = hash
	userCredential.OneTimePassword = !isSelf

	if err := data.SaveCredential(db, userCredential); err != nil {
		return fmt.Errorf("saving credentials: %w", err)
	}

	if isSelf {
		// if we updated our own password, remove the password-reset scope from our access key.
		if k, ok := c.Get("key"); ok {
			if accessKey, ok := k.(*models.AccessKey); ok {
				accessKey.Scopes = models.CommaSeparatedStrings{}
				if err = data.SaveAccessKey(db, accessKey); err != nil {
					return fmt.Errorf("updating access key: %w", err)
				}
			}
		}
	}

	return nil
}

// list of valid special chars is from OWASP, wikipedia
func isValidSymbol(letter rune) bool {
	match, _ := regexp.MatchString(fmt.Sprintf(`(.*[ !"#$%%&'()*+,-./\:;<=>?@^_{}|~%s%s]){1,}`, regexp.QuoteMeta(`/\[]`), "`"), string(letter))
	return match
}

func hasMinimumCount(min int, password string, check func(rune) bool) bool {
	var count int
	for _, r := range password {
		if check(r) {
			count++
		}
	}
	return count >= min
}

func checkPasswordRequirements(db *gorm.DB, password string) error {
	settings, err := data.GetSettings(db)
	if err != nil {
		return err
	}
	errs := make(validate.Error)

	if !hasMinimumCount(settings.LowercaseMin, password, unicode.IsLower) {
		errs["password"] = append(errs["password"], fmt.Sprintf("needs minimum %d lower case letters", settings.LowercaseMin))
	}

	if !hasMinimumCount(settings.UppercaseMin, password, unicode.IsUpper) {
		errs["password"] = append(errs["password"], fmt.Sprintf("needs minimum %d upper case letters", settings.UppercaseMin))
	}

	if !hasMinimumCount(settings.NumberMin, password, unicode.IsDigit) {
		errs["password"] = append(errs["password"], fmt.Sprintf("needs minimum %d numbers", settings.NumberMin))
	}

	if !hasMinimumCount(settings.SymbolMin, password, isValidSymbol) {
		errs["password"] = append(errs["password"], fmt.Sprintf("needs minimum %d symbols", settings.SymbolMin))
	}

	if len(password) < settings.LengthMin {
		errs["password"] = append(errs["password"], fmt.Sprintf("needs minimum length of %d", settings.LengthMin))
	}

	if len(errs["password"]) > 0 {
		return errs
	}

	return nil
}
