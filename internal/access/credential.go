package access

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"unicode"

	"golang.org/x/crypto/bcrypt"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/validate"
)

func CreateCredential(rCtx RequestContext, user *models.Identity) (string, error) {
	if err := IsAuthorized(rCtx, models.InfraAdminRole); err != nil {
		return "", HandleAuthErr(err, "user", "create", models.InfraAdminRole)
	}

	password, err := generate.CryptoRandom(12, generate.CharsetPassword)
	if err != nil {
		return "", fmt.Errorf("crypto random: %w", err)
	}

	credential := &models.Credential{
		OneTimePassword: true,
	}

	if err := createCredential(rCtx.DBTxn, user, credential, password); err != nil {
		return "", err
	}

	return password, err
}

func createCredential(tx *data.Transaction, user *models.Identity, credential *models.Credential, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("generate from password: %w", err)
	}

	credential.IdentityID = user.ID
	credential.PasswordHash = hash

	if err := data.CreateCredential(tx, credential); err != nil {
		return fmt.Errorf("create credential: %w", err)
	}

	if _, err = data.CreateProviderUser(tx, data.InfraProvider(tx), user); err != nil {
		return fmt.Errorf("create provider user: %w", err)
	}

	return nil
}

// ResetCredential resets a user's password to a specified value. If the input value is empty, a password
// is randomly generated. No matter the input, the new password is one-time use and must be changed by the user.
func ResetCredential(rCtx RequestContext, user *models.Identity, newPassword string) (string, error) {
	err := IsAuthorized(rCtx, models.InfraAdminRole)
	if err != nil {
		return "", HandleAuthErr(err, "user", "update", models.InfraAdminRole)
	}

	if newPassword == "" {
		password, err := generate.CryptoRandom(12, generate.CharsetPassword)
		if err != nil {
			return "", fmt.Errorf("crypto random: %w", err)
		}

		newPassword = password
	}

	tx := rCtx.DBTxn
	credential, err := data.GetCredentialByUserID(tx, user.ID)
	switch {
	case errors.Is(err, internal.ErrNotFound):
		if err := createCredential(tx, user, &models.Credential{OneTimePassword: true}, newPassword); err != nil {
			return "", err
		}

		return newPassword, nil
	case err != nil:
		return "", fmt.Errorf("get credential: %w", err)
	}

	credential.OneTimePassword = true

	if err := updateCredential(tx, credential, newPassword); err != nil {
		return "", err
	}

	return newPassword, nil
}

// UpdateCredential updates a user's password to a specified valued. In order to update the user's password,
// specified requirements must be met:
//
// 1. The old password hash must match value stored in the database
// 2. The new password must meet the password policy defined for the organization
func UpdateCredential(rCtx RequestContext, user *models.Identity, oldPassword, newPassword string) error {
	tx := rCtx.DBTxn

	errs := make(validate.Error)

	credential, err := data.GetCredentialByUserID(tx, user.ID)
	if err != nil {
		return fmt.Errorf("get credential: %w", err)
	}

	// compare the stored hash of the user's password and the hash of the presented password
	if err := bcrypt.CompareHashAndPassword(credential.PasswordHash, []byte(oldPassword)); err != nil {
		errs["oldPassword"] = append(errs["oldPassword"], "invalid password")
		return errs
	}

	if err := checkPasswordRequirements(tx, newPassword); err != nil {
		return err
	}

	credential.OneTimePassword = false

	if err := updateCredential(tx, credential, newPassword); err != nil {
		return err
	}

	// if we updated our own password, remove the password-reset scope from our access key.
	if accessKey := rCtx.Authenticated.AccessKey; accessKey != nil {
		for i, v := range accessKey.Scopes {
			if v == models.ScopePasswordReset {
				accessKey.Scopes = append(accessKey.Scopes[:i], accessKey.Scopes[i+1:]...)
				break
			}
		}

		if err := data.UpdateAccessKey(tx, accessKey); err != nil {
			return err
		}
	}

	return nil
}

func updateCredential(tx *data.Transaction, credential *models.Credential, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("generate from password: %w", err)
	}

	credential.PasswordHash = hash

	if err := data.UpdateCredential(tx, credential); err != nil {
		return fmt.Errorf("update credential: %w", err)
	}

	return nil
}

// list of special charaters from OWASP:
// https://owasp.org/www-community/password-special-characters
func isSymbol(r rune) bool {
	return (r >= '\u0020' && r <= '\u002F') || (r >= '\u003A' && r <= '\u0040') || (r >= '\u005B' && r <= '\u0060') || (r >= '\u007B' && r <= '\u007E')
}

func hasMinimumCount(password string, min int, minCheck func(rune) bool) bool {
	var count int
	for _, r := range password {
		if minCheck(r) {
			count++
		}
	}
	return count >= min
}

func checkPasswordRequirements(db data.ReadTxn, password string) error {
	settings, err := data.GetSettings(db)
	if err != nil {
		return err
	}

	requirements := []struct {
		minCount      int
		countFunc     func(rune) bool
		singularError string
		pluralError   string
	}{
		{settings.LengthMin, func(r rune) bool { return true }, "%d character", "%d characters"},
		{settings.LowercaseMin, unicode.IsLower, "%d lowercase letter", "%d lowercase letters"},
		{settings.UppercaseMin, unicode.IsUpper, "%d uppercase letter", "%d uppercase letters"},
		{settings.NumberMin, unicode.IsDigit, "%d number", "%d numbers"},
		{settings.SymbolMin, isSymbol, "%d symbol", "%d symbols"},
	}

	requirementError := make([]string, 0)

	valid := true
	for _, r := range requirements {
		if !hasMinimumCount(password, r.minCount, r.countFunc) {
			valid = false
		}

		switch {
		case r.minCount == 1:
			requirementError = append(requirementError, fmt.Sprintf(r.singularError, r.minCount))
		case r.minCount > 1:
			requirementError = append(requirementError, fmt.Sprintf(r.pluralError, r.minCount))
		}
	}

	if !valid {
		return validate.Error{"password": requirementError}
	}

	if err := checkBadPasswords(password); err != nil {
		return err
	}

	return nil
}

// checkBadPasswords checks if the password is a known bad password, i.e. a widely reused password.
func checkBadPasswords(password string) error {
	badPasswordsFile := os.Getenv("INFRA_SERVER_BAD_PASSWORDS_FILE")
	if badPasswordsFile == "" {
		return nil
	}

	file, err := os.Open(badPasswordsFile)
	if err != nil {
		return err
	}

	scan := bufio.NewScanner(file)
	scan.Split(bufio.ScanLines)
	for scan.Scan() {
		if scan.Text() == password {
			return fmt.Errorf("%w: cannot use a common password", internal.ErrBadRequest)
		}
	}

	if err := file.Close(); err != nil {
		return err
	}

	return nil
}
