package data

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/data/querybuilder"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type identitiesTable models.Identity

func (i identitiesTable) Table() string {
	return "identities"
}

func (i identitiesTable) Columns() []string {
	return []string{"created_at", "created_by", "deleted_at", "id", "last_seen_at", "name", "organization_id", "ssh_login_name", "updated_at", "verification_token", "verified"}
}

func (i identitiesTable) Values() []any {
	return []any{i.CreatedAt, i.CreatedBy, i.DeletedAt, i.ID, i.LastSeenAt, i.Name, i.OrganizationID, i.SSHLoginName, i.UpdatedAt, i.VerificationToken, i.Verified}
}

func (i *identitiesTable) ScanFields() []any {
	return []any{&i.CreatedAt, &i.CreatedBy, &i.DeletedAt, &i.ID, &i.LastSeenAt, &i.Name, &i.OrganizationID, &i.SSHLoginName, &i.UpdatedAt, &i.VerificationToken, &i.Verified}
}

// diff compares two string slices (x and y), it returns a slice of strings that exist in x that do not exist in y
func diff(x, y []string) []string {
	// this modifies the order of x and y, but its ok in this case since the order of the groups assigned to a user does not matter
	slices.Sort(x)
	slices.Sort(y)

	difference := []string{}

	for _, val := range x {
		_, contains := slices.BinarySearch(y, val)
		if !contains {
			difference = append(difference, val)
		}
	}

	return difference
}

func deduplicate(groups []string) []string {
	uniqueGroups := make(map[string]bool)
	for _, g := range groups {
		uniqueGroups[g] = true
	}
	return maps.Keys(uniqueGroups)
}

// AssignIdentityToGroups updates the identity's group membership relations based on the provider user's groups
// and returns the identity's current groups after the update has persisted them
func AssignIdentityToGroups(tx WriteTxn, user *models.ProviderUser, newGroups []string) ([]models.Group, error) {
	identity, err := GetIdentity(tx, GetIdentityOptions{ByID: user.IdentityID, LoadGroups: true})
	if err != nil {
		return nil, err
	}

	newGroups = deduplicate(newGroups)

	oldGroups := user.Groups
	groupsToBeRemoved := diff(oldGroups, newGroups)
	groupsToBeAdded := diff(newGroups, oldGroups)

	user.Groups = newGroups
	user.LastUpdate = time.Now().UTC()
	if err := UpdateProviderUser(tx, user); err != nil {
		return nil, fmt.Errorf("save: %w", err)
	}

	// remove user from groups
	if len(groupsToBeRemoved) > 0 {
		query := querybuilder.New(`DELETE FROM identities_groups`)
		query.B(`WHERE identity_id = ?`, user.IdentityID)
		query.B(`AND group_id in (`)
		query.B(`SELECT id FROM groups WHERE organization_id = ?`, tx.OrganizationID())
		query.B(`AND name IN`)
		queryInClause(query, groupsToBeRemoved)
		query.B(`)`)
		if _, err := tx.Exec(query.String(), query.Args...); err != nil {
			return nil, err
		}
		for _, name := range groupsToBeRemoved {
			for i, g := range identity.Groups {
				if g.Name == name {
					// remove from list
					identity.Groups = append(identity.Groups[:i], identity.Groups[i+1:]...)
				}
			}
		}
	}

	type idNamePair struct {
		ID   uid.ID
		Name string
	}

	query := querybuilder.New(`SELECT id, name FROM groups`)
	query.B(`WHERE deleted_at is null`)
	query.B(`AND organization_id = ?`, tx.OrganizationID())
	query.B(`AND name IN`)
	queryInClause(query, groupsToBeAdded)
	rows, err := tx.Query(query.String(), query.Args...)
	if err != nil {
		return nil, err
	}
	addIDs, err := scanRows(rows, func(item *idNamePair) []any {
		return []any{&item.ID, &item.Name}
	})
	if err != nil {
		return nil, err
	}

	for _, name := range groupsToBeAdded {
		// find or create group
		var groupID uid.ID
		found := false
		for _, obj := range addIDs {
			if obj.Name == name {
				found = true
				groupID = obj.ID
				break
			}
		}
		if !found {
			group := &models.Group{
				Name:              name,
				CreatedByProvider: user.ProviderID,
			}

			if err = CreateGroup(tx, group); err != nil {
				return nil, fmt.Errorf("create group: %w", err)
			}
			groupID = group.ID
		}

		rows, err := tx.Query("SELECT identity_id FROM identities_groups WHERE identity_id = ? AND group_id = ?", user.IdentityID, groupID)
		if err != nil {
			return nil, err
		}
		ids, err := scanRows(rows, func(item *uid.ID) []any {
			return []any{item}
		})
		if err != nil {
			return nil, err
		}

		if len(ids) == 0 {
			// add user to group
			_, err = tx.Exec("INSERT INTO identities_groups (identity_id, group_id) VALUES (?, ?)", user.IdentityID, groupID)
			if err != nil {
				return nil, fmt.Errorf("insert: %w", handleError(err))
			}
			identity.Groups = append(identity.Groups,
				models.Group{
					Model: models.Model{ID: groupID},
					OrganizationMember: models.OrganizationMember{
						OrganizationID: identity.OrganizationID,
					},
					Name: name,
				})
		}
	}

	return identity.Groups, nil
}

func CreateIdentity(tx WriteTxn, identity *models.Identity) error {
	if identity.VerificationToken == "" {
		identity.VerificationToken = generate.MathRandom(10, generate.CharsetAlphaNumeric)
	}
	if err := insert(tx, (*identitiesTable)(identity)); err != nil {
		return err
	}
	username, err := setSSHLoginName(tx, *identity)
	identity.SSHLoginName = username
	return err
}

func setSSHLoginName(tx WriteTxn, user models.Identity) (string, error) {
	user.SetOrganizationID(tx)
	normalizedUsername := normalizeEmailToSSHLoginName(user.Name)

	stmt := `
		UPDATE identities SET ssh_login_name = ?
		WHERE id = ? AND organization_id = ?
		AND deleted_at is null`

	for i := 0; i < 3; i++ {
		nextUsername := normalizedUsername
		if i != 0 || len(nextUsername) < 4 || isReservedUsername(nextUsername) {
			nextUsername = normalizedUsername + generate.MathRandom(3, generate.CharsetNumbers)
		}

		_, _ = tx.Exec("SAVEPOINT update_username")

		_, err := tx.Exec(stmt, nextUsername, user.ID, user.OrganizationID)
		err = handleError(err)
		var ucErr UniqueConstraintError
		if errors.As(err, &ucErr) {
			_, _ = tx.Exec("ROLLBACK TO SAVEPOINT update_username")
			continue
		}
		return nextUsername, err
	}
	return "", fmt.Errorf("failed to generated a unique ssh username")
}

// See https://man7.org/linux/man-pages/man8/useradd.8.html#CAVEATS
func normalizeEmailToSSHLoginName(emailAddr string) string {
	username, _, _ := strings.Cut(emailAddr, "@")
	username = strings.ToLower(username)

	// first character must be a letter
	if len(username) > 0 && username[0] < 'a' || username[0] > 'z' {
		username = "u" + username
	}

	username = strings.Map(func(r rune) rune {
		switch {
		case r == '_' || r == '-':
			return r
		case r >= 'a' && r <= 'z':
			return r
		case r >= '0' && r <= '9':
			return r
		default:
			// drop all other characters
			return -1
		}
	}, username)

	const maxUsernameLength = 28 // 31 bytes minus 3 reserved for random numbers
	if len(username) > maxUsernameLength {
		username = username[:maxUsernameLength]
	}
	return username
}

// linuxSystemUsernames is a list of common linux system usernames. It was
// initially generated by taking the usernames from /etc/password from the
// docker images for ubuntu, debian, fedora, centos, redhat/ubi9, alpine, and
// archlinux.
var linuxSystemUsernames = map[string]struct{}{
	"root":                   {},
	"adm":                    {},
	"_apt":                   {},
	"at":                     {},
	"backup":                 {},
	"bin":                    {},
	"cron":                   {},
	"cyrus":                  {},
	"daemon":                 {},
	"dbus":                   {},
	"ftp":                    {},
	"games":                  {},
	"gnats":                  {},
	"guest":                  {},
	"halt":                   {},
	"http":                   {},
	"irc":                    {},
	"list":                   {},
	"lp":                     {},
	"mail":                   {},
	"man":                    {},
	"news":                   {},
	"nobody":                 {},
	"ntp":                    {},
	"operator":               {},
	"postmaster":             {},
	"proxy":                  {},
	"shutdown":               {},
	"smmsp":                  {},
	"squid":                  {},
	"sshd":                   {},
	"sync":                   {},
	"sys":                    {},
	"systemd-coredump":       {},
	"systemd-journal-remote": {},
	"systemd-network":        {},
	"systemd-oom":            {},
	"systemd-resolve":        {},
	"systemd-timesync":       {},
	"tss":                    {},
	"uucp":                   {},
	"uuidd":                  {},
	"vpopmail":               {},
	"www-data":               {},
	"xfs":                    {},

	"git":    {},
	"apache": {},
	"nginx":  {},
	"docker": {},
	// TODO: what other users are created by commonly installed packages?
}

func isReservedUsername(username string) bool {
	_, match := linuxSystemUsernames[username]
	return match
}

type GetIdentityOptions struct {
	ByID           uid.ID
	ByName         string
	LoadGroups     bool
	LoadProviders  bool
	LoadPublicKeys bool

	// FromOrganization is the organization ID of the provider. When set to a
	// non-zero value the organization ID from the transaction is ignored.
	FromOrganization uid.ID
}

func GetIdentity(tx ReadTxn, opts GetIdentityOptions) (*models.Identity, error) {
	if opts.ByID == 0 && opts.ByName == "" {
		return nil, fmt.Errorf("GetIdentity must specify id or name")
	}
	identity := &identitiesTable{}
	query := querybuilder.New("SELECT")
	query.B(columnsForSelect(identity))
	query.B("FROM")
	query.B(identity.Table())
	query.B("WHERE deleted_at IS NULL")

	orgID := opts.FromOrganization
	if orgID == 0 {
		orgID = tx.OrganizationID()
	}
	query.B("AND organization_id = ?", orgID)

	switch {
	case opts.ByID != 0:
		query.B("AND identities.id = ?", opts.ByID)
	case opts.ByName != "":
		query.B("AND identities.name = ?", opts.ByName)
	default:
		return nil, fmt.Errorf("GetIdentity must specify id or name")
	}

	err := tx.QueryRow(query.String(), query.Args...).Scan(identity.ScanFields()...)
	if err != nil {
		return nil, handleError(err)
	}

	if opts.LoadGroups {
		groups, err := ListGroups(tx, ListGroupsOptions{ByGroupMember: identity.ID})
		if err != nil {
			return nil, fmt.Errorf("load identity groups: %w", err)
		}
		identity.Groups = groups
	}
	if opts.LoadProviders {
		// find the providers that this identity is active in
		opts := ListProviderUsersOptions{
			ByIdentityID: identity.ID,
			HideInactive: true,
		}
		existsInProviders, err := ListProviderUsers(tx, opts)
		if err != nil {
			return nil, err
		}

		if len(existsInProviders) > 0 {
			var providerIDs []uid.ID
			for _, relation := range existsInProviders {
				if relation.ProviderID == models.InternalGoogleProviderID {
					// add the google social login which is not stored in the database, only in memory
					identity.Providers = []models.Provider{googleProvider()}
				} else {
					providerIDs = append(providerIDs, relation.ProviderID)
				}
			}

			if len(providerIDs) > 0 {
				providers, err := ListProviders(tx, ListProvidersOptions{ByIDs: providerIDs})
				if err != nil {
					return nil, fmt.Errorf("list providers for identity: %w", err)
				}
				identity.Providers = append(identity.Providers, providers...)
			}
		}
	}

	// TODO: use a join?
	if opts.LoadPublicKeys {
		identity.PublicKeys, err = listUserPublicKeys(tx, identity.ID)
		if err != nil {
			return nil, err
		}
	}

	return (*models.Identity)(identity), nil
}

func SetIdentityVerified(tx WriteTxn, token string) error {
	q := querybuilder.New(`UPDATE identities SET verified = true`)
	q.B("WHERE verified = ? AND verification_token = ? AND organization_id = ?", false, token, tx.OrganizationID())

	_, err := tx.Exec(q.String(), q.Args...)
	return err
}

type ListIdentityOptions struct {
	ByID                   uid.ID
	ByIDs                  []uid.ID
	ByName                 string
	ByPublicKeyFingerprint string
	ByNotName              string
	ByGroupID              uid.ID
	Pagination             *Pagination
	LoadGroups             bool
	LoadProviders          bool
	LoadPublicKeys         bool
}

func ListIdentities(tx ReadTxn, opts ListIdentityOptions) ([]models.Identity, error) {
	identities := &identitiesTable{}
	query := querybuilder.New("SELECT")
	query.B(columnsForSelect(identities))
	if opts.Pagination != nil {
		query.B(", count(*) OVER()")
	}
	query.B("FROM")
	query.B(identities.Table())
	if opts.ByGroupID != 0 {
		query.B("JOIN identities_groups ON identities_groups.identity_id = id")
	}
	if opts.ByPublicKeyFingerprint != "" {
		query.B("INNER JOIN user_public_keys ON identities.id = user_public_keys.user_id")
		query.B("AND user_public_keys.fingerprint = ?", opts.ByPublicKeyFingerprint)
	}
	query.B("WHERE identities.deleted_at IS NULL")
	query.B("AND identities.organization_id = ?", tx.OrganizationID())
	if opts.ByID != 0 {
		query.B("AND identities.id = ?", opts.ByID)
	}
	if len(opts.ByIDs) > 0 {
		query.B("AND identities.id IN")
		queryInClause(query, opts.ByIDs)
	}
	if opts.ByName != "" {
		query.B("AND identities.name = ?", opts.ByName)
	}
	if opts.ByNotName != "" {
		query.B("AND identities.name != ?", opts.ByNotName)
	}
	if opts.ByGroupID != 0 {
		query.B("AND identities_groups.group_id = ?", opts.ByGroupID)
	}
	query.B("ORDER BY identities.name ASC")
	if opts.Pagination != nil {
		opts.Pagination.PaginateQuery(query)
	}

	rows, err := tx.Query(query.String(), query.Args...)
	if err != nil {
		return nil, err
	}
	result, err := scanRows(rows, func(identity *models.Identity) []any {
		fields := (*identitiesTable)(identity).ScanFields()
		if opts.Pagination != nil {
			fields = append(fields, &opts.Pagination.TotalCount)
		}
		return fields
	})
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		// return without attempting pre-loads
		return []models.Identity{}, nil
	}

	if opts.LoadGroups {
		if err := loadIdentitiesGroups(tx, result); err != nil {
			return nil, err
		}
	}

	if opts.LoadProviders {
		if err := loadIdentitiesProviders(tx, result); err != nil {
			return nil, err
		}
	}

	// TODO: use a join?
	if opts.LoadPublicKeys {
		for i, identity := range result {
			result[i].PublicKeys, err = listUserPublicKeys(tx, identity.ID)
			if err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

func loadIdentitiesGroups(tx ReadTxn, identities []models.Identity) error {
	// get the ids of all the identities
	identityIDs := []uid.ID{}
	for _, i := range identities {
		identityIDs = append(identityIDs, i.ID)
	}

	// get the groups that contain these identities
	query := querybuilder.New(`SELECT identity_id, group_id`)
	query.B(`FROM identities_groups`)
	query.B(`WHERE identity_id IN`)
	queryInClause(query, identityIDs)
	rows, err := tx.Query(query.String(), query.Args...)
	if err != nil {
		return err
	}

	type identityGroup struct {
		IdentityID uid.ID
		GroupID    uid.ID
	}
	identGrps, err := scanRows(rows, func(identGrp *identityGroup) []any {
		return []any{&identGrp.IdentityID, &identGrp.GroupID}
	})
	if err != nil {
		return err
	}

	// get the ids of the groups these users exist in to look-up the actual group entities
	identityToGroups := make(map[uid.ID][]uid.ID) // map which groups an identity is in to assign them
	groupIDs := make(map[uid.ID]bool)             // track all the distinct group ID
	for _, identGrp := range identGrps {
		iID := identGrp.IdentityID
		gID := identGrp.GroupID

		identityToGroups[iID] = append(identityToGroups[iID], gID)
		groupIDs[gID] = true
	}

	// look-up the group entities
	groups, err := ListGroups(tx, ListGroupsOptions{ByIDs: maps.Keys(groupIDs)})
	if err != nil {
		return err
	}

	// create a look-up map for reading info about the group associated with the ID
	groupsByID := make(map[uid.ID]models.Group)
	for _, g := range groups {
		groupsByID[g.ID] = g
	}

	// now we have all the info we need, add the groups to each identity
	for i := range identities {
		groups := []models.Group{}
		grpIDs := identityToGroups[identities[i].ID]
		for _, gID := range grpIDs {
			groups = append(groups, groupsByID[gID])
		}
		identities[i].Groups = groups
	}

	return nil
}

func loadIdentitiesProviders(tx ReadTxn, identities []models.Identity) error {
	// get the ids of all the identities
	identityIDs := []uid.ID{}
	for _, i := range identities {
		identityIDs = append(identityIDs, i.ID)
	}

	// look-up the relation of these identities to providers
	opts := ListProviderUsersOptions{
		ByIdentityIDs: identityIDs,
		HideInactive:  true,
	}
	existsInProviders, err := ListProviderUsers(tx, opts)
	switch {
	case err != nil:
		return err
	case len(existsInProviders) == 0:
		return nil
	}

	// get the ids of the providers these users exist in to look-up the actual provider entities
	identityToProviders := make(map[uid.ID][]uid.ID) // map which providers an identity is in to apply to them
	providerIDs := make(map[uid.ID]bool)             // track all the distinct provider ID
	for _, relation := range existsInProviders {
		iID := relation.IdentityID
		pID := relation.ProviderID

		identityToProviders[iID] = append(identityToProviders[iID], pID)
		providerIDs[pID] = true
	}

	providers, err := ListProviders(tx, ListProvidersOptions{ByIDs: maps.Keys(providerIDs)})
	if err != nil {
		return err
	}

	// create a look-up map for reading info about the provider associated with the ID
	providersByID := make(map[uid.ID]models.Provider)
	for _, p := range providers {
		providersByID[p.ID] = p
	}

	// add the google social login which is not stored in the database, only in memory
	providersByID[models.InternalGoogleProviderID] = googleProvider()

	// now we have all the info we need, add the providers to each identity
	for i := range identities {
		providers := []models.Provider{}
		pIDs := identityToProviders[identities[i].ID]
		for _, pID := range pIDs {
			providers = append(providers, providersByID[pID])
		}
		identities[i].Providers = providers
	}

	return nil
}

func UpdateIdentity(tx WriteTxn, identity *models.Identity) error {
	return update(tx, (*identitiesTable)(identity))
}

// UpdateIdentityLastSeenAt sets user.LastSeenAt to now and then updates the
// user row in the database. Updates are throttled to once every 2 seconds.
// If the user was updated recently, or the database row is already locked, the
// update will be skipped.
//
// Unlike most functions in this package, this function uses user.OrganizationID
// not tx.OrganizationID.
func UpdateIdentityLastSeenAt(tx WriteTxn, user *models.Identity) error {
	if time.Since(user.LastSeenAt) < lastSeenUpdateThreshold {
		return nil
	}

	origUpdatedAt := user.UpdatedAt
	user.LastSeenAt = time.Now()
	if err := user.OnUpdate(); err != nil {
		return err
	}

	table := (*identitiesTable)(user)
	query := querybuilder.New("UPDATE identities SET")
	query.B(columnsForUpdate(table), table.Values()...)
	query.B("WHERE deleted_at is null")
	query.B("AND organization_id = ?", user.OrganizationID)
	// only update if the row has not changed since the SELECT
	query.B("AND updated_at = ?", origUpdatedAt)
	query.B("AND id IN (SELECT id from identities WHERE id = ? FOR UPDATE SKIP LOCKED)", table.Primary())

	_, err := tx.Exec(query.String(), query.Args...)
	return handleError(err)
}

type DeleteIdentitiesOptions struct {
	ByID         uid.ID
	ByIDs        []uid.ID
	ByProviderID uid.ID
}

func DeleteIdentities(tx WriteTxn, opts DeleteIdentitiesOptions) error {
	if opts.ByID == 0 && len(opts.ByIDs) == 0 {
		return fmt.Errorf("DeleteIdentities requires an ID")
	}
	ids := opts.ByIDs
	if opts.ByID != 0 {
		ids = append(ids, opts.ByID)
	}

	if err := deleteReferencesToIdentities(tx, ids); err != nil {
		return fmt.Errorf("remove identities: %w", err)
	}

	query := querybuilder.New("UPDATE identities")
	query.B("SET deleted_at = ?", time.Now())
	query.B("WHERE id IN")
	queryInClause(query, ids)
	query.B("AND organization_id = ?", tx.OrganizationID())
	_, err := tx.Exec(query.String(), query.Args...)
	return err
}

// deleteReferencesToIdentities removes all entities (keys, grants, etc.) that reference an identity
func deleteReferencesToIdentities(tx WriteTxn, ids []uid.ID) error {
	for _, id := range ids {
		if err := DeleteAccessKeys(tx, DeleteAccessKeysOptions{ByIssuedForID: id}); err != nil {
			return fmt.Errorf("delete identity access keys: %w", err)
		}
		if err := DeleteUserPublicKeys(tx, id); err != nil {
			return fmt.Errorf("delete identity public keys: %w", err)
		}

		// if an identity does not have credentials in the Infra provider this won't be found, but we can proceed
		credential, err := GetCredentialByUserID(tx, id)
		if err != nil && !errors.Is(err, internal.ErrNotFound) {
			return fmt.Errorf("get delete identity creds: %w", err)
		}
		if credential != nil {
			err := DeleteCredential(tx, credential.ID)
			if err != nil {
				return fmt.Errorf("delete identity creds: %w", err)
			}
		}

		if err := DeleteProviderUsers(tx, DeleteProviderUsersOptions{ByIdentityID: id}); err != nil {
			return fmt.Errorf("remove provider user: %w", err)
		}

		groups, err := ListGroups(tx, ListGroupsOptions{ByGroupMember: id})
		if err != nil {
			return fmt.Errorf("list groups for identity: %w", err)
		}
		for _, group := range groups {
			err = RemoveUsersFromGroup(tx, group.ID, []uid.ID{id})
			if err != nil {
				return fmt.Errorf("delete group membership for identity: %w", err)
			}
		}
		err = DeleteGrants(tx, DeleteGrantsOptions{BySubject: models.NewSubjectForUser(id)})
		if err != nil {
			return fmt.Errorf("delete identity creds: %w", err)
		}
	}
	return nil
}

func CountAllIdentities(tx ReadTxn) (int64, error) {
	return countRows(tx, identitiesTable{})
}

// stub details for google social login provider which is not stored in the database
func googleProvider() models.Provider {
	return models.Provider{
		Model: models.Model{
			ID: models.InternalGoogleProviderID,
		},
		Name: "Google",
		URL:  "accounts.google.com",
		Kind: models.ProviderKindGoogle,
	}
}
