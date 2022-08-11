-- SQL generated by TestMigrations DO NOT EDIT.
-- Instead of editing this file, add a migration to ./migrations.go and run:
--
--     go test -run TestMigrations ./internal/server/data -update
--

CREATE TABLE access_keys (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text,
    issued_for bigint,
    provider_id bigint,
    expires_at timestamp with time zone,
    extension bigint,
    extension_deadline timestamp with time zone,
    key_id text,
    secret_checksum bytea,
    organization_id bigint,
    scopes text
);

CREATE TABLE credentials (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    identity_id bigint,
    password_hash bytea,
    one_time_password boolean,
    organization_id bigint
);

CREATE TABLE destinations (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text,
    unique_id text,
    connection_url text,
    connection_ca text,
    last_seen_at timestamp with time zone,
    organization_id bigint,
    version text,
    resources text,
    roles text
);

CREATE TABLE encryption_keys (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    key_id integer,
    name text,
    encrypted bytea,
    algorithm text,
    root_key_id text
);

CREATE TABLE grants (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    subject text,
    privilege text,
    resource text,
    created_by bigint,
    organization_id bigint
);

CREATE TABLE groups (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text,
    created_by bigint,
    organization_id bigint,
    created_by_provider bigint
);

CREATE TABLE identities (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text,
    last_seen_at timestamp with time zone,
    created_by bigint,
    organization_id bigint
);

CREATE TABLE identities_groups (
    identity_id bigint NOT NULL,
    group_id bigint NOT NULL
);

CREATE TABLE organizations (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text,
    created_by bigint,
    domain text
);

CREATE TABLE password_reset_tokens (
    id bigint NOT NULL,
    token text,
    identity_id bigint,
    expires_at timestamp with time zone,
    organization_id bigint
);

CREATE TABLE provider_users (
    identity_id bigint NOT NULL,
    provider_id bigint NOT NULL,
    email text,
    groups text,
    last_update timestamp with time zone,
    redirect_url text,
    access_token text,
    refresh_token text,
    expires_at timestamp with time zone
);

CREATE TABLE providers (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text,
    url text,
    client_id text,
    client_secret text,
    created_by bigint,
    kind text,
    auth_url text,
    scopes text,
    organization_id bigint,
    private_key text,
    client_email text,
    domain_admin_email text
);

CREATE TABLE settings (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    private_jwk bytea,
    public_jwk bytea,
    organization_id bigint,
    lowercase_min bigint DEFAULT 0,
    uppercase_min bigint DEFAULT 0,
    number_min bigint DEFAULT 0,
    symbol_min bigint DEFAULT 0,
    length_min bigint DEFAULT 8
);

ALTER TABLE ONLY access_keys
    ADD CONSTRAINT access_keys_pkey PRIMARY KEY (id);

ALTER TABLE ONLY credentials
    ADD CONSTRAINT credentials_pkey PRIMARY KEY (id);

ALTER TABLE ONLY destinations
    ADD CONSTRAINT destinations_pkey PRIMARY KEY (id);

ALTER TABLE ONLY encryption_keys
    ADD CONSTRAINT encryption_keys_pkey PRIMARY KEY (id);

ALTER TABLE ONLY grants
    ADD CONSTRAINT grants_pkey PRIMARY KEY (id);

ALTER TABLE ONLY groups
    ADD CONSTRAINT groups_pkey PRIMARY KEY (id);

ALTER TABLE ONLY identities_groups
    ADD CONSTRAINT identities_groups_pkey PRIMARY KEY (identity_id, group_id);

ALTER TABLE ONLY identities
    ADD CONSTRAINT identities_pkey PRIMARY KEY (id);

ALTER TABLE ONLY organizations
    ADD CONSTRAINT organizations_pkey PRIMARY KEY (id);

ALTER TABLE ONLY password_reset_tokens
    ADD CONSTRAINT password_reset_tokens_pkey PRIMARY KEY (id);

ALTER TABLE ONLY provider_users
    ADD CONSTRAINT provider_users_pkey PRIMARY KEY (identity_id, provider_id);

ALTER TABLE ONLY providers
    ADD CONSTRAINT providers_pkey PRIMARY KEY (id);

ALTER TABLE ONLY settings
    ADD CONSTRAINT settings_pkey PRIMARY KEY (id);

CREATE UNIQUE INDEX idx_access_keys_key_id ON access_keys USING btree (key_id) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX idx_access_keys_name ON access_keys USING btree (organization_id, name) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX idx_credentials_identity_id ON credentials USING btree (organization_id, identity_id) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX idx_destinations_unique_id ON destinations USING btree (organization_id, unique_id) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX idx_encryption_keys_key_id ON encryption_keys USING btree (key_id);

CREATE UNIQUE INDEX idx_grant_srp ON grants USING btree (organization_id, subject, privilege, resource) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX idx_groups_name ON groups USING btree (organization_id, name) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX idx_identities_name ON identities USING btree (organization_id, name) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX idx_organizations_domain ON organizations USING btree (domain) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX idx_organizations_name ON organizations USING btree (name) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX idx_password_reset_tokens_token ON password_reset_tokens USING btree (token);

CREATE UNIQUE INDEX idx_providers_name ON providers USING btree (organization_id, name) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX settings_org_id ON settings USING btree (organization_id) WHERE (deleted_at IS NULL);
