package registry

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/data"
	"github.com/infrahq/infra/internal/logging"
)

func syncProviders(r *Registry) {
	hub := newSentryHub("sync_providers_timer")
	defer recoverWithSentryHub(hub)

	providers, err := data.ListProviders(r.db, &data.Provider{})
	if err != nil {
		logging.S.Errorf("providers: %w", err)
		return
	}

	var wg sync.WaitGroup

	for _, p := range providers {
		wg.Add(1)

		go func(provider data.Provider) {
			if err := syncProvider(r, r.db, provider); err != nil {
				logging.S.Warnf("sync provider: %w", err)
			}

			wg.Done()
		}(p)
	}

	wg.Wait()
}

func syncProvider(r *Registry, db *gorm.DB, provider data.Provider) error {
	switch provider.Kind {
	case data.ProviderKindOkta:
		okta := NewOkta()

		token, err := r.GetSecret(provider.Okta.APIToken)
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

		if err := provider.SetUsers(db, emails...); err != nil {
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

		if err := provider.SetGroups(db, groupNames...); err != nil {
			return err
		}

		if err := importRoleMappings(db, r.config.Users, r.config.Groups); err != nil {
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
		user, err := data.CreateOrUpdateUser(db, &data.User{Email: email}, &data.User{Email: email})
		if err != nil {
			return err
		}

		toKeep = append(toKeep, user.ID)
	}

	if err := data.DeleteUsers(db, db.Model(&data.User{}).Not(toKeep)); err != nil {
		return err
	}

	return nil
}

func syncGroups(db *gorm.DB, groups map[string][]string) error {
	toKeep := make([]uuid.UUID, 0)

	for name, emails := range groups {
		group, err := data.CreateOrUpdateGroup(db, &data.Group{Name: name}, &data.Group{Name: name})
		if err != nil {
			return err
		}

		users, err := data.ListUsers(db, db.Where("email IN (?)", emails))
		if err != nil {
			return err
		}

		if err := group.BindUsers(db, users...); err != nil {
			return err
		}

		toKeep = append(toKeep, group.ID)
	}

	if err := data.DeleteGroups(db, db.Model(&data.Group{}).Not(toKeep)); err != nil {
		return err
	}

	return nil
}

func syncDestinations(db *gorm.DB, maxAge time.Duration) {
	hub := newSentryHub("sync_destinations_timer")
	defer recoverWithSentryHub(hub)

	now := time.Now()

	destinations, err := data.ListDestinations(db, &data.Destination{})
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

	if err := data.DeleteDestinations(db, db.Model(&data.Destination{}).Not(toKeep)); err != nil {
		logging.S.Errorw("delete destination", "error", err.Error())
		return
	}
}
