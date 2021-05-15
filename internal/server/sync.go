package server

import (
	"sort"
	"time"

	"github.com/infrahq/infra/internal/data"
)

type Sync struct {
	stop chan bool
}

const SYNC_INTERVAL_SECONDS = 10

func (s *Sync) Start(sync func()) {
	ticker := time.NewTicker(SYNC_INTERVAL_SECONDS * time.Second)
	sync()

	go func() {
		for {
			select {
			case <-ticker.C:
				sync()
			case <-s.stop:
				ticker.Stop()
				return
			}
		}
	}()
}

func (s *Sync) Stop() {
	s.stop <- true
}

func syncUsers(d *data.Data, provider string, emails []string) error {
	// Create new users
	for _, email := range emails {
		user, err := d.FindUser(email)
		if err != nil {
			return err
		}

		if user == nil {
			user = &data.User{}
		}

		user.Email = email
		providers := user.Providers
		if len(providers) == 0 {
			user.Providers = []string{provider}
		} else {
			hasOkta := false
			for _, p := range user.Providers {
				if p == "okta" {
					hasOkta = true
				}
			}
			if !hasOkta {
				user.Providers = append(user.Providers, provider)
				sort.Strings(user.Providers)
			}
		}

		d.PutUser(user)
	}

	users, err := d.ListUsers()
	if err != nil {
		return err
	}

	emailsMap := make(map[string]bool)
	for _, email := range emails {
		emailsMap[email] = true
	}

	// delete users who aren't in okta
	for _, user := range users {
		if !emailsMap[user.Email] {
			providers := []string{}
			for _, p := range user.Providers {
				if p != provider {
					providers = append(providers, p)
				}
			}
			user.Providers = providers
			if len(user.Providers) == 0 {
				d.DeleteUser(user.ID)
			}
		}
	}

	return nil
}
