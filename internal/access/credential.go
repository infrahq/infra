package access

import (
	"bufio"
	"errors"
	"fmt"
	"os"

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

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("generate from password: %w", err)
	}

	credential.OneTimePassword = true
	credential.PasswordHash = hash

	if err := data.UpdateCredential(tx, credential); err != nil {
		return "", fmt.Errorf("update credential: %w", err)
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

	hash, err := GenerateFromPassword(newPassword)
	if err != nil {
		return err
	}

	credential.OneTimePassword = false
	credential.PasswordHash = hash

	if err := data.UpdateCredential(tx, credential); err != nil {
		return fmt.Errorf("update credential: %w", err)
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

func GenerateFromPassword(password string) ([]byte, error) {
	if len(password) < 8 {
		return nil, validate.Error{"password": []string{"8 characters"}}
	}

	if err := checkBadPasswords(password); err != nil {
		return nil, err
	}

	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
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
