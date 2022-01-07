package registry

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
)

// TODO: #691 set user permissions based on their internal infrahq grant (user or admin)
var defaultPermissions = strings.Join([]string{
	string(access.PermissionUserRead),
	string(access.PermissionTokenRevoke),
	string(access.PermissionCredentialCreate),
}, " ")

func syncProviders(r *Registry) {
	hub := newSentryHub("sync_providers_timer")
	defer recoverWithSentryHub(hub)

	providers, err := data.ListProviders(r.db, &models.Provider{})
	if err != nil {
		logging.S.Errorf("providers: %w", err)
		return
	}

	var wg sync.WaitGroup

	for _, p := range providers {
		wg.Add(1)

		go func(provider models.Provider) {
			if err := syncProvider(r, r.db, provider); err != nil {
				logging.S.Warnf("sync provider: %w", err)
			}

			wg.Done()
		}(p)
	}

	wg.Wait()
}

func syncProvider(r *Registry, db *gorm.DB, provider models.Provider) error {
	switch provider.Kind {
	case models.ProviderKindOkta:
		okta := NewOkta()

		token, err := r.GetSecret(string(provider.Okta.APIToken))
		if err != nil {
			return err
		}

		emails, err := okta.Emails(provider.Domain, provider.ClientID, token)
		if err != nil {
			return err
		}

		if err := syncUsers(db, emails); err != nil {
			return err
		}

		if err := data.SetProviderUsers(db, &provider, emails...); err != nil {
			return err
		}

		groups, err := okta.Groups(provider.Domain, provider.ClientID, token)
		if err != nil {
			return err
		}

		if err := syncGroups(db, groups); err != nil {
			return err
		}

		groupNames := make([]string, 0)
		for k := range groups {
			groupNames = append(groupNames, k)
		}

		if err := data.SetProviderGroups(db, &provider, groupNames...); err != nil {
			return err
		}

		if err := importGrantMappings(db, r.config.Users, r.config.Groups); err != nil {
			return err
		}

		return nil
	default:
		return fmt.Errorf("unrecognized provider kind")
	}
}

func syncUsers(db *gorm.DB, emails []string) error {
	toKeep := make([]uuid.UUID, 0)

	for _, email := range emails {
		user, err := data.CreateOrUpdateUser(db, &models.User{Email: email, Permissions: defaultPermissions}, &models.User{Email: email})
		if err != nil {
			return err
		}

		toKeep = append(toKeep, user.ID)
	}

	if err := data.DeleteUsers(db, data.ByIDNotInList(toKeep)); err != nil {
		return err
	}

	return nil
}

func syncGroups(db *gorm.DB, groups map[string][]string) error {
	toKeep := make([]uuid.UUID, 0)

	for name, emails := range groups {
		group, err := data.CreateOrUpdateGroup(db, &models.Group{Name: name}, &models.Group{Name: name})
		if err != nil {
			return err
		}

		users, err := data.ListUsers(db, data.ByEmailInList(emails))
		if err != nil {
			return err
		}

		if err := data.BindGroupUsers(db, group, users...); err != nil {
			return err
		}

		toKeep = append(toKeep, group.ID)
	}

	if err := data.DeleteGroups(db, db.Model(&models.Group{}).Not(toKeep)); err != nil {
		return err
	}

	return nil
}

func syncDestinations(db *gorm.DB, maxAge time.Duration) {
	hub := newSentryHub("sync_destinations_timer")
	defer recoverWithSentryHub(hub)

	now := time.Now()

	destinations, err := data.ListDestinations(db, &models.Destination{})
	if err != nil {
		logging.S.Errorw("sync destination", "error", err.Error())
		return
	}

	toKeep := make([]uuid.UUID, 0)

	for _, d := range destinations {
		expires := d.UpdatedAt.Add(maxAge)

		if now.Before(expires) {
			toKeep = append(toKeep, d.ID)
		}
	}

	if err := data.DeleteDestinations(db, func(db *gorm.DB) *gorm.DB {
		return db.Model(&models.Destination{}).Not(toKeep)
	}); err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			logging.S.Errorw("delete destination", "error", err.Error())
			return
		}
	}
}
