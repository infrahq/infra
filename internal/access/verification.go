package access

import "github.com/infrahq/infra/internal/server/data"

func VerifyUserByToken(c RequestContext, verificationToken string) error {
	return data.SetIdentityVerified(c.DBTxn, verificationToken)
}
