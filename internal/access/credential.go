package access

import (
	"fmt"
	"regexp"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func CreateCredential(c *gin.Context, user models.Identity) (string, error) {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return "", err
	}

	oneTimePassword, err := generate.CryptoRandom(10)
	if err != nil {
		return "", fmt.Errorf("generate: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(oneTimePassword), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash: %w", err)
	}

	userCredential := &models.Credential{
		IdentityID:          user.ID,
		PasswordHash:        hash,
		OneTimePassword:     true,
		OneTimePasswordUsed: false,
	}

	if err := data.CreateCredential(db, userCredential); err != nil {
		return "", err
	}

	return oneTimePassword, nil
}

func UpdateCredential(c *gin.Context, user *models.Identity, newPassword string) error {
	db, err := hasAuthorization(c, user.ID, isIdentitySelf, models.InfraAdminRole)
	if err != nil {
		return err
	}

	err = checkPasswordRequirements(newPassword)
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash: %w", err)
	}

	userCredential, err := data.GetCredential(db, data.ByIdentityID(user.ID))
	if err != nil {
		return fmt.Errorf("existing credential: %w", err)
	}

	userCredential.PasswordHash = hash
	userCredential.OneTimePassword = false
	userCredential.OneTimePasswordUsed = false

	if user.ID != AuthenticatedIdentity(c).ID {
		// an admin can only set a one time password for a user
		userCredential.OneTimePassword = true
		userCredential.OneTimePasswordUsed = false
	}

	if err := data.SaveCredential(db, userCredential); err != nil {
		return err
	}

	return nil
}

func lowercaseCheck(n int, password string) bool {
	match, _ := regexp.MatchString(fmt.Sprintf(`(.*\p{Ll}){%d,}`, n), password)
	fmt.Println(match)
	return match
}

func uppercaseCheck(n int, password string) bool {
	match, _ := regexp.MatchString(fmt.Sprintf(`(.*\p{Lu}){%d,}`, n), password)
	return match
}

func numberCheck(n int, password string) bool {
	match, _ := regexp.MatchString(fmt.Sprintf(`(.*\p{N}){%d,}`, n), password)
	return match
}

// list is from OWASP, wikipedia
func symbolCheck(n int, password string) bool {
	match, _ := regexp.MatchString(fmt.Sprintf(`(.*[ !"#$%%&'()*+,-./\:;<=>?@^_{}|~%s%s]){%d,}`, regexp.QuoteMeta(`/\[]`), "`", n), password)
	return match
	// /\[]`
}

func checkPasswordRequirements(password string) error {
	var errs []string

	kind := "custom"
	minLowercase := 1
	minUppercase := 1
	minNumber := 1
	minSymbol := 1
	lowercase := true
	uppercase := true
	number := true
	symbol := true
	minLength := 8

	if kind == "custom" {
		if lowercase && !lowercaseCheck(minLowercase, password) {
			errs = append(errs, fmt.Sprintf("needs minimum %d lower case letters", minLowercase))
		}

		if uppercase && !uppercaseCheck(minUppercase, password) {
			errs = append(errs, fmt.Sprintf("needs minimum %d upper case letters", minUppercase))
		}

		if number && !numberCheck(minNumber, password) {
			errs = append(errs, fmt.Sprintf("needs minimum %d numbers", minNumber))
		}

		if symbol && !symbolCheck(minSymbol, password) {
			errs = append(errs, fmt.Sprintf("needs minimum %d symbols", minSymbol))
		}

		if len(password) > minLength {
			errs = append(errs, fmt.Sprintf("needs min length of %d", minLength))
		}
	}

	if len(errs) > 0 {
		err := fmt.Errorf(fmt.Sprintf("%v", errs))
		// Wrap so it is easier to parse the list of errors
		return fmt.Errorf("cannot update password: new password does not pass requirements: %w", err)
	}

	return nil
}
