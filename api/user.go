package api

import (
	"github.com/infrahq/infra/internal/validate"
	"github.com/infrahq/infra/uid"
)

type GetUserRequest struct {
	ID IDOrSelf `uri:"id"`
}

func (r GetUserRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("id", r.ID),
	}
}

type User struct {
	ID            uid.ID          `json:"id" note:"User ID" example:"4ACFkc434M"`
	Created       Time            `json:"created" note:"Date the user was created"`
	Updated       Time            `json:"updated" note:"Date the user was updated"`
	LastSeenAt    Time            `json:"lastSeenAt" note:"Date the user was last seen"`
	Name          string          `json:"name" note:"Name of the user" example:"bob@example.com"`
	ProviderNames []string        `json:"providerNames,omitempty" note:"List of providers this user belongs to" example:"['okta']"`
	PublicKeys    []UserPublicKey `json:"publicKeys,omitempty" note:"List of the users public keys"`
	SSHLoginName  string          `json:"sshLoginName" note:"Username for SSH destinations" example:"bob"`
}

type ListUsersRequest struct {
	Name                 string   `form:"name" note:"Name of the user" example:"bob@example.com"`
	Group                uid.ID   `form:"group" note:"Group the user belongs to" example:"admins"`
	IDs                  []uid.ID `form:"ids" note:"List of User IDs"`
	ShowSystem           bool     `form:"showSystem" note:"if true, this shows the connector and other internal users" example:"false"`
	PublicKeyFingerprint string   `form:"publicKeyFingerprint" note:"Find the user with a public key that matches this SHA256 fingerprint."`
	PaginationRequest
}

func (r ListUsersRequest) ValidationRules() []validate.ValidationRule {
	// no-op ValidationRules implementation so that the rules from the
	// embedded PaginationRequest struct are not applied twice.
	return nil
}

// CreateUserRequest is only for creating users with the Infra provider
type CreateUserRequest struct {
	Name string `json:"name" note:"Email address of the new user" example:"bob@example.com"`
}

func (r CreateUserRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("name", r.Name),
		validate.Email("name", r.Name),
	}
}

type CreateUserResponse struct {
	ID              uid.ID `json:"id" note:"User ID"`
	Name            string `json:"name" note:"Email address of the user" example:"bob@example.com"`
	OneTimePassword string `json:"oneTimePassword,omitempty" note:"One-time password (only returned when self-hosted)" example:"password"`
}

type UpdateUserRequest struct {
	ID          uid.ID `uri:"id" json:"-"`
	OldPassword string `json:"oldPassword" note:"Old password for the user. Only required when the access key making this request is not owned by an Infra admin" example:"oldpassword"`
	Password    string `json:"password" note:"New one-time password for the user" example:"newpassword"`
}

func (r UpdateUserRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("id", r.ID),
	}
}

type UpdateUserResponse struct {
	User
	OneTimePassword string `json:"oneTimePassword,omitempty"`
}

func (req ListUsersRequest) SetPage(page int) Paginatable {
	req.PaginationRequest.Page = page

	return req
}

type AddUserPublicKeyRequest struct {
	Name string `json:"name" note:"Name of the public key, often the name of the device used to create it"`
	// PublicKey is the key type and base64 encoded public key as it would appear
	// in an authorized keys file.
	PublicKey string `json:"publicKey"`
}

func (r AddUserPublicKeyRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("publicKey", r.PublicKey),
		ValidateName(r.Name),
	}
}

type UserPublicKey struct {
	ID uid.ID `json:"id"`
	// Name identifies the key, generally set to the hostname of the host
	// that created the key pair.
	Name    string `json:"name"`
	Created Time   `json:"created"`
	// PublicKey is the base64 encoded public key of the key pair.
	PublicKey string `json:"publicKey"`
	// KeyType is the string that identifies the type of the key, in a format
	// appropriate for an SSH authorized keys file.
	KeyType string `json:"keyType"`
	// Fingerprint is the SHA256 fingerprint of the public key.
	Fingerprint string `json:"fingerprint" note:"SHA256 fingerprint of the key"`
	Expires     Time   `json:"expires"`
}
