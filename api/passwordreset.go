package api

type PasswordResetRequest struct {
	Email string `json:"email"`
}

type VerifiedResetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}
