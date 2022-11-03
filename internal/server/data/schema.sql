-- SQL generated by TestMigrations DO NOT EDIT.
-- Instead of editing this file, add a migration to ./migrations.go and run:
--
--     go test -run TestMigrations ./internal/server/data -update
--

CREATE FUNCTION grants_notify() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
PERFORM pg_notify(current_schema() || '.grants_' || NEW.organization_id, row_to_json(NEW)::text);
RETURN NULL;
END; $$;

CREATE FUNCTION listen_on_chan(chan text) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    EXECUTE format('LISTEN %I', current_schema() || '.' || chan);
END; $$;

CREATE FUNCTION uidinttostr(id bigint) RETURNS text
    LANGUAGE plpgsql
    AS $$
			DECLARE
			encodebase58map CONSTANT TEXT := '123456789abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ';
			base_count 			BIGINT DEFAULT 0;
				encoded    			TEXT DEFAULT '';
				divisor    			BIGINT;
				mod        			BIGINT DEFAULT 0;
			
			BEGIN
				IF id <= 0 THEN
					RETURN '';
				END IF;

				IF id < 58 THEN
					RETURN SUBSTRING(encodeBase58Map FROM id FOR 1);
				END IF;

				WHILE id > 0 LOOP
					divisor := id / 58;
					mod := (id - (58 * divisor));
					encoded = CONCAT(SUBSTRING(encodeBase58Map FROM CAST(mod+1 as int) FOR 1), encoded);
					id = id / 58;
				END LOOP;
			
				RETURN encoded;
			
			END; $$;

CREATE FUNCTION uidstrtoint(encoded text) RETURNS bigint
    LANGUAGE plpgsql
    AS $$
			DECLARE
				encodebase58map CONSTANT TEXT := '123456789abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ';
				base_count 			BIGINT := 0;
				result    			BIGINT := 0;
				i    						INT := 1;
				pos        			BIGINT := 0;
				MAX_SIGNED_INT CONSTANT BIGINT := 9223372036854775807;
			
			BEGIN
				WHILE (SUBSTRING(encoded FROM 1 FOR 1) = '1') LOOP
					encoded = SUBSTRING(encoded FROM 2 FOR LENGTH(encoded)-1);
				END LOOP;

				IF LENGTH(encoded) > 11 THEN
					RAISE EXCEPTION 'invalid base58: too long';
				END IF;

				WHILE (i <= LENGTH(encoded)) LOOP
				  IF (result > MAX_SIGNED_INT/58) THEN 
						RAISE EXCEPTION 'invalid base58: value too large: %', encoded;
					END IF;
					result = result * 58;
					pos := POSITION(SUBSTRING(encoded FROM i FOR 1) in encodeBase58Map);
					IF (pos <= 0) THEN
						RAISE EXCEPTION 'invalid base58: byte % is out of range', i-1;
					END IF;
					IF (MAX_SIGNED_INT - (pos - 1) < result) THEN
						RAISE EXCEPTION 'invalid base58: value too large: %', encoded;
					END IF;
					result = result + (pos-1);
					i = i + 1;
				END LOOP;

				RETURN result;
			
			END; $$;

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
    scopes text,
    organization_id bigint
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
    version text,
    resources text,
    roles text,
    organization_id bigint
);

CREATE TABLE device_flow_auth_requests (
    id bigint NOT NULL,
    user_code text NOT NULL,
    device_code text NOT NULL,
    approved boolean,
    access_key_id bigint,
    access_key_token text,
    expires_at timestamp with time zone,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
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
    organization_id bigint,
    update_index bigint
);

CREATE TABLE groups (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text,
    created_by bigint,
    created_by_provider bigint,
    organization_id bigint
);

CREATE TABLE identities (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text,
    last_seen_at timestamp with time zone,
    created_by bigint,
    organization_id bigint,
    verified boolean DEFAULT false NOT NULL,
    verification_token text DEFAULT substr(replace(translate(encode(decode(md5((random())::text), 'hex'::text), 'base64'::text), '/+'::text, '=='::text), '='::text, ''::text), 1, 10) NOT NULL
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
    expires_at timestamp with time zone,
    given_name text DEFAULT ''::text,
    family_name text DEFAULT ''::text,
    active boolean DEFAULT true
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
    private_key text,
    client_email text,
    domain_admin_email text,
    organization_id bigint,
    allowed_domains text DEFAULT ''::text
);

CREATE SEQUENCE seq_update_index
    START WITH 10000
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE settings (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    private_jwk bytea,
    public_jwk bytea,
    lowercase_min bigint DEFAULT 0,
    uppercase_min bigint DEFAULT 0,
    number_min bigint DEFAULT 0,
    symbol_min bigint DEFAULT 0,
    length_min bigint DEFAULT 8,
    organization_id bigint
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
    ADD CONSTRAINT provider_users_pkey PRIMARY KEY (provider_id, identity_id);

ALTER TABLE ONLY providers
    ADD CONSTRAINT providers_pkey PRIMARY KEY (id);

ALTER TABLE ONLY settings
    ADD CONSTRAINT settings_pkey PRIMARY KEY (id);

CREATE UNIQUE INDEX idx_access_keys_key_id ON access_keys USING btree (key_id) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX idx_access_keys_name ON access_keys USING btree (organization_id, name) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX idx_credentials_identity_id ON credentials USING btree (organization_id, identity_id) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX idx_destinations_name ON destinations USING btree (organization_id, name) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX idx_destinations_unique_id ON destinations USING btree (organization_id, unique_id) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX idx_dfar_user_code ON device_flow_auth_requests USING btree (user_code) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX idx_emails_providers ON provider_users USING btree (email, provider_id);

CREATE UNIQUE INDEX idx_encryption_keys_key_id ON encryption_keys USING btree (key_id);

CREATE UNIQUE INDEX idx_grant_srp ON grants USING btree (organization_id, subject, privilege, resource) WHERE (deleted_at IS NULL);

CREATE INDEX idx_grants_update_index ON grants USING btree (organization_id, update_index);

CREATE UNIQUE INDEX idx_groups_name ON groups USING btree (organization_id, name) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX idx_identities_name ON identities USING btree (organization_id, name) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX idx_identities_verified ON identities USING btree (organization_id, verification_token) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX idx_organizations_domain ON organizations USING btree (domain) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX idx_password_reset_tokens_token ON password_reset_tokens USING btree (token);

CREATE UNIQUE INDEX idx_providers_name ON providers USING btree (organization_id, name) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX settings_org_id ON settings USING btree (organization_id) WHERE (deleted_at IS NULL);

CREATE TRIGGER grants_notify_trigger AFTER INSERT OR UPDATE ON grants FOR EACH ROW EXECUTE FUNCTION grants_notify();
