--
-- PostgreSQL database dump
--

-- Dumped from database version 14.4
-- Dumped by pg_dump version 14.3

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: testing; Type: SCHEMA; Schema: -; Owner: postgres
--

SET search_path TO testing;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: access_keys; Type: TABLE; Schema: testing; Owner: postgres
--

CREATE TABLE access_keys (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text,
    issued_for bigint,
    provider_id bigint,
    scopes text,
    expires_at timestamp with time zone,
    extension bigint,
    extension_deadline timestamp with time zone,
    key_id text,
    secret_checksum bytea
);


ALTER TABLE access_keys OWNER TO postgres;

--
-- Name: access_keys_id_seq; Type: SEQUENCE; Schema: testing; Owner: postgres
--

CREATE SEQUENCE access_keys_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE access_keys_id_seq OWNER TO postgres;

--
-- Name: access_keys_id_seq; Type: SEQUENCE OWNED BY; Schema: testing; Owner: postgres
--

ALTER SEQUENCE access_keys_id_seq OWNED BY access_keys.id;


--
-- Name: credentials; Type: TABLE; Schema: testing; Owner: postgres
--

CREATE TABLE credentials (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    identity_id bigint,
    password_hash bytea,
    one_time_password boolean
);


ALTER TABLE credentials OWNER TO postgres;

--
-- Name: credentials_id_seq; Type: SEQUENCE; Schema: testing; Owner: postgres
--

CREATE SEQUENCE credentials_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE credentials_id_seq OWNER TO postgres;

--
-- Name: credentials_id_seq; Type: SEQUENCE OWNED BY; Schema: testing; Owner: postgres
--

ALTER SEQUENCE credentials_id_seq OWNED BY credentials.id;


--
-- Name: destinations; Type: TABLE; Schema: testing; Owner: postgres
--

CREATE TABLE destinations (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text,
    unique_id text,
    last_seen_at timestamp with time zone,
    version text,
    connection_url text,
    connection_ca text,
    resources text,
    roles text
);


ALTER TABLE destinations OWNER TO postgres;

--
-- Name: destinations_id_seq; Type: SEQUENCE; Schema: testing; Owner: postgres
--

CREATE SEQUENCE destinations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE destinations_id_seq OWNER TO postgres;

--
-- Name: destinations_id_seq; Type: SEQUENCE OWNED BY; Schema: testing; Owner: postgres
--

ALTER SEQUENCE destinations_id_seq OWNED BY destinations.id;


--
-- Name: encryption_keys; Type: TABLE; Schema: testing; Owner: postgres
--

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


ALTER TABLE encryption_keys OWNER TO postgres;

--
-- Name: encryption_keys_id_seq; Type: SEQUENCE; Schema: testing; Owner: postgres
--

CREATE SEQUENCE encryption_keys_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE encryption_keys_id_seq OWNER TO postgres;

--
-- Name: encryption_keys_id_seq; Type: SEQUENCE OWNED BY; Schema: testing; Owner: postgres
--

ALTER SEQUENCE encryption_keys_id_seq OWNED BY encryption_keys.id;


--
-- Name: grants; Type: TABLE; Schema: testing; Owner: postgres
--

CREATE TABLE grants (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    subject text,
    privilege text,
    resource text,
    created_by bigint
);


ALTER TABLE grants OWNER TO postgres;

--
-- Name: grants_id_seq; Type: SEQUENCE; Schema: testing; Owner: postgres
--

CREATE SEQUENCE grants_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE grants_id_seq OWNER TO postgres;

--
-- Name: grants_id_seq; Type: SEQUENCE OWNED BY; Schema: testing; Owner: postgres
--

ALTER SEQUENCE grants_id_seq OWNED BY grants.id;


--
-- Name: groups; Type: TABLE; Schema: testing; Owner: postgres
--

CREATE TABLE groups (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text,
    created_by bigint,
    created_by_provider bigint
);


ALTER TABLE groups OWNER TO postgres;

--
-- Name: groups_id_seq; Type: SEQUENCE; Schema: testing; Owner: postgres
--

CREATE SEQUENCE groups_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE groups_id_seq OWNER TO postgres;

--
-- Name: groups_id_seq; Type: SEQUENCE OWNED BY; Schema: testing; Owner: postgres
--

ALTER SEQUENCE groups_id_seq OWNED BY groups.id;


--
-- Name: identities; Type: TABLE; Schema: testing; Owner: postgres
--

CREATE TABLE identities (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text,
    last_seen_at timestamp with time zone,
    created_by bigint
);


ALTER TABLE identities OWNER TO postgres;

--
-- Name: identities_groups; Type: TABLE; Schema: testing; Owner: postgres
--

CREATE TABLE identities_groups (
    group_id bigint NOT NULL,
    identity_id bigint NOT NULL
);


ALTER TABLE identities_groups OWNER TO postgres;

--
-- Name: identities_id_seq; Type: SEQUENCE; Schema: testing; Owner: postgres
--

CREATE SEQUENCE identities_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE identities_id_seq OWNER TO postgres;

--
-- Name: identities_id_seq; Type: SEQUENCE OWNED BY; Schema: testing; Owner: postgres
--

ALTER SEQUENCE identities_id_seq OWNED BY identities.id;


--
-- Name: migrations; Type: TABLE; Schema: testing; Owner: postgres
--

CREATE TABLE migrations (
    id character varying(255) NOT NULL
);


ALTER TABLE migrations OWNER TO postgres;

INSERT INTO migrations VALUES('SCHEMA_INIT');
INSERT INTO migrations VALUES('202203231621');
INSERT INTO migrations VALUES('202203241643');
INSERT INTO migrations VALUES('202203301642');
INSERT INTO migrations VALUES('202203301652');
INSERT INTO migrations VALUES('202203301643');
INSERT INTO migrations VALUES('202203301645');
INSERT INTO migrations VALUES('202203301646');
INSERT INTO migrations VALUES('202203301647');
INSERT INTO migrations VALUES('202203301648');
INSERT INTO migrations VALUES('202204061643');
INSERT INTO migrations VALUES('202204111503');
INSERT INTO migrations VALUES('202204181613');
INSERT INTO migrations VALUES('202204211705');
INSERT INTO migrations VALUES('202204281130');
INSERT INTO migrations VALUES('202204291613');
INSERT INTO migrations VALUES('202206081027');
INSERT INTO migrations VALUES('202206151027');
INSERT INTO migrations VALUES('202206161733');
INSERT INTO migrations VALUES('202206281027');
INSERT INTO migrations VALUES('202207041724');
INSERT INTO migrations VALUES('202207081217');
INSERT INTO migrations VALUES('202207211828');


--
-- Name: provider_users; Type: TABLE; Schema: testing; Owner: postgres
--

CREATE TABLE provider_users (
    identity_id bigint NOT NULL,
    provider_id bigint NOT NULL,
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    email text,
    groups text,
    last_update timestamp with time zone,
    redirect_url text,
    access_token text,
    refresh_token text,
    expires_at timestamp with time zone
);


ALTER TABLE provider_users OWNER TO postgres;

--
-- Name: provider_users_id_seq; Type: SEQUENCE; Schema: testing; Owner: postgres
--

CREATE SEQUENCE provider_users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE provider_users_id_seq OWNER TO postgres;

--
-- Name: provider_users_id_seq; Type: SEQUENCE OWNED BY; Schema: testing; Owner: postgres
--

ALTER SEQUENCE provider_users_id_seq OWNED BY provider_users.id;


--
-- Name: providers; Type: TABLE; Schema: testing; Owner: postgres
--

CREATE TABLE providers (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text,
    kind text,
    url text,
    client_id text,
    client_secret text,
    auth_url text,
    scopes text,
    created_by bigint,
    private_key text,
    client_email text,
    domain_admin_email text
);


ALTER TABLE providers OWNER TO postgres;

--
-- Name: providers_id_seq; Type: SEQUENCE; Schema: testing; Owner: postgres
--

CREATE SEQUENCE providers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE providers_id_seq OWNER TO postgres;

--
-- Name: providers_id_seq; Type: SEQUENCE OWNED BY; Schema: testing; Owner: postgres
--

ALTER SEQUENCE providers_id_seq OWNED BY providers.id;


--
-- Name: settings; Type: TABLE; Schema: testing; Owner: postgres
--

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
    length_min bigint DEFAULT 8
);


ALTER TABLE settings OWNER TO postgres;

--
-- Name: settings_id_seq; Type: SEQUENCE; Schema: testing; Owner: postgres
--

CREATE SEQUENCE settings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE settings_id_seq OWNER TO postgres;

--
-- Name: settings_id_seq; Type: SEQUENCE OWNED BY; Schema: testing; Owner: postgres
--

ALTER SEQUENCE settings_id_seq OWNED BY settings.id;


--
-- Name: access_keys id; Type: DEFAULT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY access_keys ALTER COLUMN id SET DEFAULT nextval('access_keys_id_seq'::regclass);


--
-- Name: credentials id; Type: DEFAULT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY credentials ALTER COLUMN id SET DEFAULT nextval('credentials_id_seq'::regclass);


--
-- Name: destinations id; Type: DEFAULT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY destinations ALTER COLUMN id SET DEFAULT nextval('destinations_id_seq'::regclass);


--
-- Name: encryption_keys id; Type: DEFAULT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY encryption_keys ALTER COLUMN id SET DEFAULT nextval('encryption_keys_id_seq'::regclass);


--
-- Name: grants id; Type: DEFAULT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY grants ALTER COLUMN id SET DEFAULT nextval('grants_id_seq'::regclass);


--
-- Name: groups id; Type: DEFAULT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY groups ALTER COLUMN id SET DEFAULT nextval('groups_id_seq'::regclass);


--
-- Name: identities id; Type: DEFAULT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY identities ALTER COLUMN id SET DEFAULT nextval('identities_id_seq'::regclass);


--
-- Name: provider_users id; Type: DEFAULT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY provider_users ALTER COLUMN id SET DEFAULT nextval('provider_users_id_seq'::regclass);


--
-- Name: providers id; Type: DEFAULT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY providers ALTER COLUMN id SET DEFAULT nextval('providers_id_seq'::regclass);


--
-- Name: settings id; Type: DEFAULT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY settings ALTER COLUMN id SET DEFAULT nextval('settings_id_seq'::regclass);


--
-- Name: access_keys access_keys_pkey; Type: CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY access_keys
    ADD CONSTRAINT access_keys_pkey PRIMARY KEY (id);


--
-- Name: credentials credentials_pkey; Type: CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY credentials
    ADD CONSTRAINT credentials_pkey PRIMARY KEY (id);


--
-- Name: destinations destinations_pkey; Type: CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY destinations
    ADD CONSTRAINT destinations_pkey PRIMARY KEY (id);


--
-- Name: encryption_keys encryption_keys_pkey; Type: CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY encryption_keys
    ADD CONSTRAINT encryption_keys_pkey PRIMARY KEY (id);


--
-- Name: grants grants_pkey; Type: CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY grants
    ADD CONSTRAINT grants_pkey PRIMARY KEY (id);


--
-- Name: groups groups_pkey; Type: CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY groups
    ADD CONSTRAINT groups_pkey PRIMARY KEY (id);


--
-- Name: identities_groups identities_groups_pkey; Type: CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY identities_groups
    ADD CONSTRAINT identities_groups_pkey PRIMARY KEY (group_id, identity_id);


--
-- Name: identities identities_pkey; Type: CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY identities
    ADD CONSTRAINT identities_pkey PRIMARY KEY (id);


--
-- Name: migrations migrations_pkey; Type: CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY migrations
    ADD CONSTRAINT migrations_pkey PRIMARY KEY (id);


--
-- Name: provider_users provider_users_pkey; Type: CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY provider_users
    ADD CONSTRAINT provider_users_pkey PRIMARY KEY (identity_id, provider_id);


--
-- Name: providers providers_pkey; Type: CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY providers
    ADD CONSTRAINT providers_pkey PRIMARY KEY (id);


--
-- Name: settings settings_pkey; Type: CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY settings
    ADD CONSTRAINT settings_pkey PRIMARY KEY (id);


--
-- Name: idx_access_keys_key_id; Type: INDEX; Schema: testing; Owner: postgres
--

CREATE UNIQUE INDEX idx_access_keys_key_id ON access_keys USING btree (key_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_access_keys_name; Type: INDEX; Schema: testing; Owner: postgres
--

CREATE UNIQUE INDEX idx_access_keys_name ON access_keys USING btree (name) WHERE (deleted_at IS NULL);


--
-- Name: idx_credentials_identity_id; Type: INDEX; Schema: testing; Owner: postgres
--

CREATE UNIQUE INDEX idx_credentials_identity_id ON credentials USING btree (identity_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_destinations_unique_id; Type: INDEX; Schema: testing; Owner: postgres
--

CREATE UNIQUE INDEX idx_destinations_unique_id ON destinations USING btree (unique_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_encryption_keys_key_id; Type: INDEX; Schema: testing; Owner: postgres
--

CREATE UNIQUE INDEX idx_encryption_keys_key_id ON encryption_keys USING btree (key_id);


--
-- Name: idx_grant_srp; Type: INDEX; Schema: testing; Owner: postgres
--

CREATE UNIQUE INDEX idx_grant_srp ON grants USING btree (subject, privilege, resource) WHERE (deleted_at IS NULL);


--
-- Name: idx_groups_name; Type: INDEX; Schema: testing; Owner: postgres
--

CREATE UNIQUE INDEX idx_groups_name ON groups USING btree (name) WHERE (deleted_at IS NULL);


--
-- Name: idx_identities_name; Type: INDEX; Schema: testing; Owner: postgres
--

CREATE UNIQUE INDEX idx_identities_name ON identities USING btree (name) WHERE (deleted_at IS NULL);


--
-- Name: idx_providers_name; Type: INDEX; Schema: testing; Owner: postgres
--

CREATE UNIQUE INDEX idx_providers_name ON providers USING btree (name) WHERE (deleted_at IS NULL);


--
-- Name: access_keys fk_access_keys_issued_for_identity; Type: FK CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY access_keys
    ADD CONSTRAINT fk_access_keys_issued_for_identity FOREIGN KEY (issued_for) REFERENCES identities(id);


--
-- Name: identities_groups fk_identities_groups_group; Type: FK CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY identities_groups
    ADD CONSTRAINT fk_identities_groups_group FOREIGN KEY (group_id) REFERENCES groups(id);


--
-- Name: identities_groups fk_identities_groups_identity; Type: FK CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY identities_groups
    ADD CONSTRAINT fk_identities_groups_identity FOREIGN KEY (identity_id) REFERENCES identities(id);


--
-- Name: provider_users fk_provider_users_identity; Type: FK CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY provider_users
    ADD CONSTRAINT fk_provider_users_identity FOREIGN KEY (identity_id) REFERENCES identities(id);


--
-- Name: provider_users fk_provider_users_provider; Type: FK CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY provider_users
    ADD CONSTRAINT fk_provider_users_provider FOREIGN KEY (provider_id) REFERENCES providers(id);


--
-- PostgreSQL database dump complete
--

