--
-- PostgreSQL database dump
--

-- Dumped from database version 14.4 (Debian 14.4-1.pgdg110+1)
-- Dumped by pg_dump version 14.4 (Debian 14.4-1.pgdg110+1)

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

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: access_keys; Type: TABLE; Schema: testing; Owner: postgres
--

CREATE TABLE testing.access_keys (
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


ALTER TABLE testing.access_keys OWNER TO postgres;

--
-- Name: access_keys_id_seq; Type: SEQUENCE; Schema: testing; Owner: postgres
--

CREATE SEQUENCE testing.access_keys_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE testing.access_keys_id_seq OWNER TO postgres;

--
-- Name: access_keys_id_seq; Type: SEQUENCE OWNED BY; Schema: testing; Owner: postgres
--

ALTER SEQUENCE testing.access_keys_id_seq OWNED BY testing.access_keys.id;


--
-- Name: credentials; Type: TABLE; Schema: testing; Owner: postgres
--

CREATE TABLE testing.credentials (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    identity_id bigint,
    password_hash bytea,
    one_time_password boolean
);


ALTER TABLE testing.credentials OWNER TO postgres;

--
-- Name: credentials_id_seq; Type: SEQUENCE; Schema: testing; Owner: postgres
--

CREATE SEQUENCE testing.credentials_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE testing.credentials_id_seq OWNER TO postgres;

--
-- Name: credentials_id_seq; Type: SEQUENCE OWNED BY; Schema: testing; Owner: postgres
--

ALTER SEQUENCE testing.credentials_id_seq OWNED BY testing.credentials.id;


--
-- Name: destinations; Type: TABLE; Schema: testing; Owner: postgres
--

CREATE TABLE testing.destinations (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text,
    unique_id text,
    connection_url text,
    connection_ca text,
    resources text,
    roles text
);


ALTER TABLE testing.destinations OWNER TO postgres;

--
-- Name: destinations_id_seq; Type: SEQUENCE; Schema: testing; Owner: postgres
--

CREATE SEQUENCE testing.destinations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE testing.destinations_id_seq OWNER TO postgres;

--
-- Name: destinations_id_seq; Type: SEQUENCE OWNED BY; Schema: testing; Owner: postgres
--

ALTER SEQUENCE testing.destinations_id_seq OWNED BY testing.destinations.id;


--
-- Name: encryption_keys; Type: TABLE; Schema: testing; Owner: postgres
--

CREATE TABLE testing.encryption_keys (
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


ALTER TABLE testing.encryption_keys OWNER TO postgres;

--
-- Name: encryption_keys_id_seq; Type: SEQUENCE; Schema: testing; Owner: postgres
--

CREATE SEQUENCE testing.encryption_keys_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE testing.encryption_keys_id_seq OWNER TO postgres;

--
-- Name: encryption_keys_id_seq; Type: SEQUENCE OWNED BY; Schema: testing; Owner: postgres
--

ALTER SEQUENCE testing.encryption_keys_id_seq OWNED BY testing.encryption_keys.id;


--
-- Name: grants; Type: TABLE; Schema: testing; Owner: postgres
--

CREATE TABLE testing.grants (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    subject text,
    privilege text,
    resource text,
    created_by bigint
);


ALTER TABLE testing.grants OWNER TO postgres;

--
-- Name: grants_id_seq; Type: SEQUENCE; Schema: testing; Owner: postgres
--

CREATE SEQUENCE testing.grants_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE testing.grants_id_seq OWNER TO postgres;

--
-- Name: grants_id_seq; Type: SEQUENCE OWNED BY; Schema: testing; Owner: postgres
--

ALTER SEQUENCE testing.grants_id_seq OWNED BY testing.grants.id;


--
-- Name: groups; Type: TABLE; Schema: testing; Owner: postgres
--

CREATE TABLE testing.groups (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text,
    created_by bigint,
    created_by_provider bigint
);


ALTER TABLE testing.groups OWNER TO postgres;

--
-- Name: groups_id_seq; Type: SEQUENCE; Schema: testing; Owner: postgres
--

CREATE SEQUENCE testing.groups_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE testing.groups_id_seq OWNER TO postgres;

--
-- Name: groups_id_seq; Type: SEQUENCE OWNED BY; Schema: testing; Owner: postgres
--

ALTER SEQUENCE testing.groups_id_seq OWNED BY testing.groups.id;


--
-- Name: identities; Type: TABLE; Schema: testing; Owner: postgres
--

CREATE TABLE testing.identities (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text,
    last_seen_at timestamp with time zone,
    created_by bigint
);


ALTER TABLE testing.identities OWNER TO postgres;

--
-- Name: identities_groups; Type: TABLE; Schema: testing; Owner: postgres
--

CREATE TABLE testing.identities_groups (
    group_id bigint NOT NULL,
    identity_id bigint NOT NULL
);


ALTER TABLE testing.identities_groups OWNER TO postgres;

--
-- Name: identities_id_seq; Type: SEQUENCE; Schema: testing; Owner: postgres
--

CREATE SEQUENCE testing.identities_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE testing.identities_id_seq OWNER TO postgres;

--
-- Name: identities_id_seq; Type: SEQUENCE OWNED BY; Schema: testing; Owner: postgres
--

ALTER SEQUENCE testing.identities_id_seq OWNED BY testing.identities.id;


--
-- Name: migrations; Type: TABLE; Schema: testing; Owner: postgres
--

CREATE TABLE testing.migrations (
    id character varying(255) NOT NULL
);


ALTER TABLE testing.migrations OWNER TO postgres;

--
-- Name: provider_users; Type: TABLE; Schema: testing; Owner: postgres
--

CREATE TABLE testing.provider_users (
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


ALTER TABLE testing.provider_users OWNER TO postgres;

--
-- Name: provider_users_id_seq; Type: SEQUENCE; Schema: testing; Owner: postgres
--

CREATE SEQUENCE testing.provider_users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE testing.provider_users_id_seq OWNER TO postgres;

--
-- Name: provider_users_id_seq; Type: SEQUENCE OWNED BY; Schema: testing; Owner: postgres
--

ALTER SEQUENCE testing.provider_users_id_seq OWNED BY testing.provider_users.id;


--
-- Name: providers; Type: TABLE; Schema: testing; Owner: postgres
--

CREATE TABLE testing.providers (
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
    created_by bigint
);


ALTER TABLE testing.providers OWNER TO postgres;

--
-- Name: providers_id_seq; Type: SEQUENCE; Schema: testing; Owner: postgres
--

CREATE SEQUENCE testing.providers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE testing.providers_id_seq OWNER TO postgres;

--
-- Name: providers_id_seq; Type: SEQUENCE OWNED BY; Schema: testing; Owner: postgres
--

ALTER SEQUENCE testing.providers_id_seq OWNED BY testing.providers.id;


--
-- Name: settings; Type: TABLE; Schema: testing; Owner: postgres
--

CREATE TABLE testing.settings (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    private_jwk bytea,
    public_jwk bytea
);


ALTER TABLE testing.settings OWNER TO postgres;

--
-- Name: settings_id_seq; Type: SEQUENCE; Schema: testing; Owner: postgres
--

CREATE SEQUENCE testing.settings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE testing.settings_id_seq OWNER TO postgres;

--
-- Name: settings_id_seq; Type: SEQUENCE OWNED BY; Schema: testing; Owner: postgres
--

ALTER SEQUENCE testing.settings_id_seq OWNED BY testing.settings.id;


--
-- Name: access_keys id; Type: DEFAULT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.access_keys ALTER COLUMN id SET DEFAULT nextval('testing.access_keys_id_seq'::regclass);


--
-- Name: credentials id; Type: DEFAULT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.credentials ALTER COLUMN id SET DEFAULT nextval('testing.credentials_id_seq'::regclass);


--
-- Name: destinations id; Type: DEFAULT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.destinations ALTER COLUMN id SET DEFAULT nextval('testing.destinations_id_seq'::regclass);


--
-- Name: encryption_keys id; Type: DEFAULT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.encryption_keys ALTER COLUMN id SET DEFAULT nextval('testing.encryption_keys_id_seq'::regclass);


--
-- Name: grants id; Type: DEFAULT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.grants ALTER COLUMN id SET DEFAULT nextval('testing.grants_id_seq'::regclass);


--
-- Name: groups id; Type: DEFAULT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.groups ALTER COLUMN id SET DEFAULT nextval('testing.groups_id_seq'::regclass);


--
-- Name: identities id; Type: DEFAULT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.identities ALTER COLUMN id SET DEFAULT nextval('testing.identities_id_seq'::regclass);


--
-- Name: provider_users id; Type: DEFAULT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.provider_users ALTER COLUMN id SET DEFAULT nextval('testing.provider_users_id_seq'::regclass);


--
-- Name: providers id; Type: DEFAULT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.providers ALTER COLUMN id SET DEFAULT nextval('testing.providers_id_seq'::regclass);


--
-- Name: settings id; Type: DEFAULT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.settings ALTER COLUMN id SET DEFAULT nextval('testing.settings_id_seq'::regclass);


--
-- Data for Name: access_keys; Type: TABLE DATA; Schema: testing; Owner: postgres
--



--
-- Data for Name: credentials; Type: TABLE DATA; Schema: testing; Owner: postgres
--



--
-- Data for Name: destinations; Type: TABLE DATA; Schema: testing; Owner: postgres
--

INSERT INTO testing.destinations (id, created_at, updated_at, deleted_at, name, unique_id, connection_url, connection_ca, resources, roles) VALUES (67067378731917312, '2022-07-05 00:41:49.143574+00', '2022-07-05 01:41:49.143574+00', NULL, 'docker-desktop', 'unique-id', 'localhost:123', '', '', '');

--
-- Data for Name: encryption_keys; Type: TABLE DATA; Schema: testing; Owner: postgres
--

INSERT INTO testing.encryption_keys (id, created_at, updated_at, deleted_at, key_id, name, encrypted, algorithm, root_key_id) VALUES (67303707122147328, '2022-07-05 17:20:54.220702+00', '2022-07-05 17:20:54.220702+00', NULL, 693382051, 'dbkey', '\x414141414d444158787243635a59594c566165725a313951324832566a386f6a547861746c494d366f4869795173657176785764526a432b4e496848647a796f3034686f73675a685a584e6e59323045343744455167414d582b364b70426e354b5246517676634d', 'aesgcm', '/root/.infra/sqlite3.db.key');


--
-- Data for Name: grants; Type: TABLE DATA; Schema: testing; Owner: postgres
--

INSERT INTO testing.grants (id, created_at, updated_at, deleted_at, subject, privilege, resource, created_by) VALUES (67303707466080256, '2022-07-05 17:20:54.302104+00', '2022-07-05 17:20:54.302104+00', NULL, 'i:a4xYgwWVu5', 'connector', 'infra', 1);


--
-- Data for Name: groups; Type: TABLE DATA; Schema: testing; Owner: postgres
--



--
-- Data for Name: identities; Type: TABLE DATA; Schema: testing; Owner: postgres
--

INSERT INTO testing.identities (id, created_at, updated_at, deleted_at, name, last_seen_at, created_by) VALUES (67303707453497344, '2022-07-05 17:20:54.29989+00', '2022-07-05 17:20:54.29989+00', NULL, 'connector', '0001-01-01 00:00:00+00', 1);


--
-- Data for Name: identities_groups; Type: TABLE DATA; Schema: testing; Owner: postgres
--



--
-- Data for Name: migrations; Type: TABLE DATA; Schema: testing; Owner: postgres
--

INSERT INTO testing.migrations (id) VALUES ('SCHEMA_INIT');
INSERT INTO testing.migrations (id) VALUES ('202203231621');
INSERT INTO testing.migrations (id) VALUES ('202203241643');
INSERT INTO testing.migrations (id) VALUES ('202203301642');
INSERT INTO testing.migrations (id) VALUES ('202203301652');
INSERT INTO testing.migrations (id) VALUES ('202203301643');
INSERT INTO testing.migrations (id) VALUES ('202203301645');
INSERT INTO testing.migrations (id) VALUES ('202203301646');
INSERT INTO testing.migrations (id) VALUES ('202203301647');
INSERT INTO testing.migrations (id) VALUES ('202203301648');
INSERT INTO testing.migrations (id) VALUES ('202204061643');
INSERT INTO testing.migrations (id) VALUES ('202204111503');
INSERT INTO testing.migrations (id) VALUES ('202204181613');
INSERT INTO testing.migrations (id) VALUES ('202204211705');
INSERT INTO testing.migrations (id) VALUES ('202204281130');
INSERT INTO testing.migrations (id) VALUES ('202204291613');
INSERT INTO testing.migrations (id) VALUES ('202206081027');
INSERT INTO testing.migrations (id) VALUES ('202206151027');
INSERT INTO testing.migrations (id) VALUES ('202206161733');
INSERT INTO testing.migrations (id) VALUES ('202206281027');


--
-- Data for Name: provider_users; Type: TABLE DATA; Schema: testing; Owner: postgres
--



--
-- Data for Name: providers; Type: TABLE DATA; Schema: testing; Owner: postgres
--

INSERT INTO testing.providers (id, created_at, updated_at, deleted_at, name, kind, url, client_id, client_secret, auth_url, scopes, created_by) VALUES (67303707445108736, '2022-07-05 17:20:54.29773+00', '2022-07-05 17:20:54.29773+00', NULL, 'infra', 'infra', '', '', 'AAAAEL/fO4GYTiWmIebAG9xDqfoGYWVzZ2NtBEEtkvcbL3Jvb3QvLmluZnJhL3NxbGl0ZTMuZGIua2V5DLMvi/B5vzbatFVeoA', '', '', 1);


--
-- Data for Name: settings; Type: TABLE DATA; Schema: testing; Owner: postgres
--

INSERT INTO testing.settings (id, created_at, updated_at, deleted_at, private_jwk, public_jwk) VALUES (67303707436720128, '2022-07-05 17:20:54.295862+00', '2022-07-05 17:20:54.295862+00', NULL, '\x4141414134727546534a62655176534c7865582f58746b757a754f6b4b694f527868706a375330373139796d6d597a654862367747496351534576574565495878694d353633455548706f4b454372784461574f364676704d48636832436c6559746f5577303753656c783679696e4f4844514b453572356e5152626a62744744647831676a7354385150413958774851794c437a65715042396c4e76554a743753765a476c4c4b63466b74627832366e316f6e333959762b326f37556148423950645668784a687477724968363049694953695458314768745856546d7a306e3269354a2f7649426c6235492b4d6f6b6964514a6454516d55645949574646485251642f654975714e6337397769634877795138764d53783478746b4a525056704434786c396b426c564f4e4a2f415a6a6b475957567a5a324e74424545746b7663624c334a76623351764c6d6c755a6e4a684c334e7862476c305a544d755a474975613256354442706d2f314972796853306a6c57343277', '\x7b22757365223a22736967222c226b7479223a224f4b50222c226b6964223a223074586764377135354b6e544d546e69374a456a494144635455773669676a3761336b366a4653527571303d222c22637276223a2245643235353139222c22616c67223a2245443235353139222c2278223a2244325a474f7331562d5933433059786d4b704a3441303363315a4854696f63486131793944364f74394b59227d');


--
-- Name: access_keys_id_seq; Type: SEQUENCE SET; Schema: testing; Owner: postgres
--

SELECT pg_catalog.setval('testing.access_keys_id_seq', 1, false);


--
-- Name: credentials_id_seq; Type: SEQUENCE SET; Schema: testing; Owner: postgres
--

SELECT pg_catalog.setval('testing.credentials_id_seq', 1, false);


--
-- Name: destinations_id_seq; Type: SEQUENCE SET; Schema: testing; Owner: postgres
--

SELECT pg_catalog.setval('testing.destinations_id_seq', 1, false);


--
-- Name: encryption_keys_id_seq; Type: SEQUENCE SET; Schema: testing; Owner: postgres
--

SELECT pg_catalog.setval('testing.encryption_keys_id_seq', 1, false);


--
-- Name: grants_id_seq; Type: SEQUENCE SET; Schema: testing; Owner: postgres
--

SELECT pg_catalog.setval('testing.grants_id_seq', 1, false);


--
-- Name: groups_id_seq; Type: SEQUENCE SET; Schema: testing; Owner: postgres
--

SELECT pg_catalog.setval('testing.groups_id_seq', 1, false);


--
-- Name: identities_id_seq; Type: SEQUENCE SET; Schema: testing; Owner: postgres
--

SELECT pg_catalog.setval('testing.identities_id_seq', 1, false);


--
-- Name: provider_users_id_seq; Type: SEQUENCE SET; Schema: testing; Owner: postgres
--

SELECT pg_catalog.setval('testing.provider_users_id_seq', 1, false);


--
-- Name: providers_id_seq; Type: SEQUENCE SET; Schema: testing; Owner: postgres
--

SELECT pg_catalog.setval('testing.providers_id_seq', 1, false);


--
-- Name: settings_id_seq; Type: SEQUENCE SET; Schema: testing; Owner: postgres
--

SELECT pg_catalog.setval('testing.settings_id_seq', 1, false);


--
-- Name: access_keys access_keys_pkey; Type: CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.access_keys
    ADD CONSTRAINT access_keys_pkey PRIMARY KEY (id);


--
-- Name: credentials credentials_pkey; Type: CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.credentials
    ADD CONSTRAINT credentials_pkey PRIMARY KEY (id);


--
-- Name: destinations destinations_pkey; Type: CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.destinations
    ADD CONSTRAINT destinations_pkey PRIMARY KEY (id);


--
-- Name: encryption_keys encryption_keys_pkey; Type: CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.encryption_keys
    ADD CONSTRAINT encryption_keys_pkey PRIMARY KEY (id);


--
-- Name: grants grants_pkey; Type: CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.grants
    ADD CONSTRAINT grants_pkey PRIMARY KEY (id);


--
-- Name: groups groups_pkey; Type: CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.groups
    ADD CONSTRAINT groups_pkey PRIMARY KEY (id);


--
-- Name: identities_groups identities_groups_pkey; Type: CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.identities_groups
    ADD CONSTRAINT identities_groups_pkey PRIMARY KEY (group_id, identity_id);


--
-- Name: identities identities_pkey; Type: CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.identities
    ADD CONSTRAINT identities_pkey PRIMARY KEY (id);


--
-- Name: migrations migrations_pkey; Type: CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.migrations
    ADD CONSTRAINT migrations_pkey PRIMARY KEY (id);


--
-- Name: provider_users provider_users_pkey; Type: CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.provider_users
    ADD CONSTRAINT provider_users_pkey PRIMARY KEY (identity_id, provider_id);


--
-- Name: providers providers_pkey; Type: CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.providers
    ADD CONSTRAINT providers_pkey PRIMARY KEY (id);


--
-- Name: settings settings_pkey; Type: CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.settings
    ADD CONSTRAINT settings_pkey PRIMARY KEY (id);


--
-- Name: idx_access_keys_key_id; Type: INDEX; Schema: testing; Owner: postgres
--

CREATE UNIQUE INDEX idx_access_keys_key_id ON testing.access_keys USING btree (key_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_access_keys_name; Type: INDEX; Schema: testing; Owner: postgres
--

CREATE UNIQUE INDEX idx_access_keys_name ON testing.access_keys USING btree (name) WHERE (deleted_at IS NULL);


--
-- Name: idx_credentials_identity_id; Type: INDEX; Schema: testing; Owner: postgres
--

CREATE UNIQUE INDEX idx_credentials_identity_id ON testing.credentials USING btree (identity_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_destinations_unique_id; Type: INDEX; Schema: testing; Owner: postgres
--

CREATE UNIQUE INDEX idx_destinations_unique_id ON testing.destinations USING btree (unique_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_encryption_keys_key_id; Type: INDEX; Schema: testing; Owner: postgres
--

CREATE UNIQUE INDEX idx_encryption_keys_key_id ON testing.encryption_keys USING btree (key_id);


--
-- Name: idx_groups_name; Type: INDEX; Schema: testing; Owner: postgres
--

CREATE UNIQUE INDEX idx_groups_name ON testing.groups USING btree (name) WHERE (deleted_at IS NULL);


--
-- Name: idx_identities_name; Type: INDEX; Schema: testing; Owner: postgres
--

CREATE UNIQUE INDEX idx_identities_name ON testing.identities USING btree (name) WHERE (deleted_at IS NULL);


--
-- Name: idx_providers_name; Type: INDEX; Schema: testing; Owner: postgres
--

CREATE UNIQUE INDEX idx_providers_name ON testing.providers USING btree (name) WHERE (deleted_at IS NULL);


--
-- Name: access_keys fk_access_keys_issued_for_identity; Type: FK CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.access_keys
    ADD CONSTRAINT fk_access_keys_issued_for_identity FOREIGN KEY (issued_for) REFERENCES testing.identities(id);


--
-- Name: identities_groups fk_identities_groups_group; Type: FK CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.identities_groups
    ADD CONSTRAINT fk_identities_groups_group FOREIGN KEY (group_id) REFERENCES testing.groups(id);


--
-- Name: identities_groups fk_identities_groups_identity; Type: FK CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.identities_groups
    ADD CONSTRAINT fk_identities_groups_identity FOREIGN KEY (identity_id) REFERENCES testing.identities(id);


--
-- Name: provider_users fk_provider_users_identity; Type: FK CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.provider_users
    ADD CONSTRAINT fk_provider_users_identity FOREIGN KEY (identity_id) REFERENCES testing.identities(id);


--
-- Name: provider_users fk_provider_users_provider; Type: FK CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.provider_users
    ADD CONSTRAINT fk_provider_users_provider FOREIGN KEY (provider_id) REFERENCES testing.providers(id);


--
-- PostgreSQL database dump complete
--

