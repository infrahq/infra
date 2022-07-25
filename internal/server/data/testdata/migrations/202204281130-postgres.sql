--
-- PostgreSQL database dump
--

-- Dumped from database version 14.4
-- Dumped by pg_dump version 14.4

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
-- Name: access_keys; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.access_keys (
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
    secret_checksum bytea
);


ALTER TABLE public.access_keys OWNER TO postgres;

--
-- Name: access_keys_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.access_keys_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.access_keys_id_seq OWNER TO postgres;

--
-- Name: access_keys_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.access_keys_id_seq OWNED BY public.access_keys.id;


--
-- Name: credentials; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.credentials (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    identity_id bigint,
    password_hash bytea,
    one_time_password boolean,
    one_time_password_used boolean
);


ALTER TABLE public.credentials OWNER TO postgres;

--
-- Name: credentials_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.credentials_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.credentials_id_seq OWNER TO postgres;

--
-- Name: credentials_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.credentials_id_seq OWNED BY public.credentials.id;


--
-- Name: destinations; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.destinations (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text,
    unique_id text,
    connection_url text,
    connection_ca text
);


ALTER TABLE public.destinations OWNER TO postgres;

--
-- Name: destinations_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.destinations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.destinations_id_seq OWNER TO postgres;

--
-- Name: destinations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.destinations_id_seq OWNED BY public.destinations.id;


--
-- Name: encryption_keys; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.encryption_keys (
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


ALTER TABLE public.encryption_keys OWNER TO postgres;

--
-- Name: encryption_keys_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.encryption_keys_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.encryption_keys_id_seq OWNER TO postgres;

--
-- Name: encryption_keys_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.encryption_keys_id_seq OWNED BY public.encryption_keys.id;


--
-- Name: grants; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.grants (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    subject text,
    privilege text,
    resource text,
    created_by bigint
);


ALTER TABLE public.grants OWNER TO postgres;

--
-- Name: grants_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.grants_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.grants_id_seq OWNER TO postgres;

--
-- Name: grants_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.grants_id_seq OWNED BY public.grants.id;


--
-- Name: groups; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.groups (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text,
    created_by bigint
);


ALTER TABLE public.groups OWNER TO postgres;

--
-- Name: groups_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.groups_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.groups_id_seq OWNER TO postgres;

--
-- Name: groups_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.groups_id_seq OWNED BY public.groups.id;


--
-- Name: identities; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.identities (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    kind text,
    name text,
    last_seen_at timestamp with time zone,
    created_by bigint
);


ALTER TABLE public.identities OWNER TO postgres;

--
-- Name: identities_groups; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.identities_groups (
    group_id bigint NOT NULL,
    identity_id bigint NOT NULL
);


ALTER TABLE public.identities_groups OWNER TO postgres;

--
-- Name: identities_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.identities_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.identities_id_seq OWNER TO postgres;

--
-- Name: identities_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.identities_id_seq OWNED BY public.identities.id;


--
-- Name: migrations; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.migrations (
    id character varying(255) NOT NULL
);


ALTER TABLE public.migrations OWNER TO postgres;

--
-- Name: provider_users; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.provider_users (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    provider_id bigint,
    identity_id bigint,
    email text,
    groups text,
    last_update timestamp with time zone,
    redirect_url text,
    access_token text,
    refresh_token text,
    expires_at timestamp with time zone
);


ALTER TABLE public.provider_users OWNER TO postgres;

--
-- Name: provider_users_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.provider_users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.provider_users_id_seq OWNER TO postgres;

--
-- Name: provider_users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.provider_users_id_seq OWNED BY public.provider_users.id;


--
-- Name: providers; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.providers (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text,
    url text,
    client_id text,
    client_secret text,
    created_by bigint
);


ALTER TABLE public.providers OWNER TO postgres;

--
-- Name: providers_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.providers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.providers_id_seq OWNER TO postgres;

--
-- Name: providers_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.providers_id_seq OWNED BY public.providers.id;


--
-- Name: root_certificates; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.root_certificates (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    key_algorithm text,
    signing_algorithm text,
    public_key text,
    private_key text,
    signed_cert text,
    expires_at timestamp with time zone
);


ALTER TABLE public.root_certificates OWNER TO postgres;

--
-- Name: root_certificates_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.root_certificates_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.root_certificates_id_seq OWNER TO postgres;

--
-- Name: root_certificates_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.root_certificates_id_seq OWNED BY public.root_certificates.id;


--
-- Name: settings; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.settings (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    private_jwk bytea,
    public_jwk bytea,
    signup_enabled boolean
);


ALTER TABLE public.settings OWNER TO postgres;

--
-- Name: settings_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.settings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.settings_id_seq OWNER TO postgres;

--
-- Name: settings_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.settings_id_seq OWNED BY public.settings.id;


--
-- Name: trusted_certificates; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.trusted_certificates (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    key_algorithm text,
    signing_algorithm text,
    public_key text,
    cert_pem bytea,
    identity text,
    expires_at timestamp with time zone,
    one_time_use boolean
);


ALTER TABLE public.trusted_certificates OWNER TO postgres;

--
-- Name: trusted_certificates_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.trusted_certificates_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.trusted_certificates_id_seq OWNER TO postgres;

--
-- Name: trusted_certificates_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.trusted_certificates_id_seq OWNED BY public.trusted_certificates.id;


--
-- Name: access_keys id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.access_keys ALTER COLUMN id SET DEFAULT nextval('public.access_keys_id_seq'::regclass);


--
-- Name: credentials id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.credentials ALTER COLUMN id SET DEFAULT nextval('public.credentials_id_seq'::regclass);


--
-- Name: destinations id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.destinations ALTER COLUMN id SET DEFAULT nextval('public.destinations_id_seq'::regclass);


--
-- Name: encryption_keys id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.encryption_keys ALTER COLUMN id SET DEFAULT nextval('public.encryption_keys_id_seq'::regclass);


--
-- Name: grants id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.grants ALTER COLUMN id SET DEFAULT nextval('public.grants_id_seq'::regclass);


--
-- Name: groups id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.groups ALTER COLUMN id SET DEFAULT nextval('public.groups_id_seq'::regclass);


--
-- Name: identities id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.identities ALTER COLUMN id SET DEFAULT nextval('public.identities_id_seq'::regclass);


--
-- Name: provider_users id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.provider_users ALTER COLUMN id SET DEFAULT nextval('public.provider_users_id_seq'::regclass);


--
-- Name: providers id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.providers ALTER COLUMN id SET DEFAULT nextval('public.providers_id_seq'::regclass);


--
-- Name: root_certificates id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.root_certificates ALTER COLUMN id SET DEFAULT nextval('public.root_certificates_id_seq'::regclass);


--
-- Name: settings id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.settings ALTER COLUMN id SET DEFAULT nextval('public.settings_id_seq'::regclass);


--
-- Name: trusted_certificates id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.trusted_certificates ALTER COLUMN id SET DEFAULT nextval('public.trusted_certificates_id_seq'::regclass);


--
-- Data for Name: access_keys; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.access_keys (id, created_at, updated_at, deleted_at, name, issued_for, provider_id, expires_at, extension, extension_deadline, key_id, secret_checksum) FROM stdin;
\.


--
-- Data for Name: credentials; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.credentials (id, created_at, updated_at, deleted_at, identity_id, password_hash, one_time_password, one_time_password_used) FROM stdin;
\.


--
-- Data for Name: destinations; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.destinations (id, created_at, updated_at, deleted_at, name, unique_id, connection_url, connection_ca) FROM stdin;
\.


--
-- Data for Name: encryption_keys; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.encryption_keys (id, created_at, updated_at, deleted_at, key_id, name, encrypted, algorithm, root_key_id) FROM stdin;
\.


--
-- Data for Name: grants; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.grants (id, created_at, updated_at, deleted_at, subject, privilege, resource, created_by) FROM stdin;
\.


--
-- Data for Name: groups; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.groups (id, created_at, updated_at, deleted_at, name, created_by) FROM stdin;
\.


--
-- Data for Name: identities; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.identities (id, created_at, updated_at, deleted_at, kind, name, last_seen_at, created_by) FROM stdin;
\.


--
-- Data for Name: identities_groups; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.identities_groups (group_id, identity_id) FROM stdin;
\.


--
-- Data for Name: migrations; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.migrations (id) FROM stdin;
SCHEMA_INIT
202203231621
202203241643
202203301642
202203301652
202203301643
202203301644
202203301645
202203301646
202203301647
202203301648
202204061643
202204111503
202204181613
202204211705
\.


--
-- Data for Name: provider_users; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.provider_users (id, created_at, updated_at, deleted_at, provider_id, identity_id, email, groups, last_update, redirect_url, access_token, refresh_token, expires_at) FROM stdin;
\.


--
-- Data for Name: providers; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.providers (id, created_at, updated_at, deleted_at, name, url, client_id, client_secret, created_by) FROM stdin;
\.


--
-- Data for Name: root_certificates; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.root_certificates (id, created_at, updated_at, deleted_at, key_algorithm, signing_algorithm, public_key, private_key, signed_cert, expires_at) FROM stdin;
\.


--
-- Data for Name: settings; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.settings (id, created_at, updated_at, deleted_at, private_jwk, public_jwk, signup_enabled) FROM stdin;
\.


--
-- Data for Name: trusted_certificates; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.trusted_certificates (id, created_at, updated_at, deleted_at, key_algorithm, signing_algorithm, public_key, cert_pem, identity, expires_at, one_time_use) FROM stdin;
\.


--
-- Name: access_keys_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.access_keys_id_seq', 1, false);


--
-- Name: credentials_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.credentials_id_seq', 1, false);


--
-- Name: destinations_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.destinations_id_seq', 1, false);


--
-- Name: encryption_keys_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.encryption_keys_id_seq', 1, false);


--
-- Name: grants_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.grants_id_seq', 1, false);


--
-- Name: groups_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.groups_id_seq', 1, false);


--
-- Name: identities_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.identities_id_seq', 1, false);


--
-- Name: provider_users_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.provider_users_id_seq', 1, false);


--
-- Name: providers_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.providers_id_seq', 1, false);


--
-- Name: root_certificates_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.root_certificates_id_seq', 1, false);


--
-- Name: settings_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.settings_id_seq', 1, false);


--
-- Name: trusted_certificates_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.trusted_certificates_id_seq', 1, false);


--
-- Name: access_keys access_keys_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.access_keys
    ADD CONSTRAINT access_keys_pkey PRIMARY KEY (id);


--
-- Name: credentials credentials_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.credentials
    ADD CONSTRAINT credentials_pkey PRIMARY KEY (id);


--
-- Name: destinations destinations_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.destinations
    ADD CONSTRAINT destinations_pkey PRIMARY KEY (id);


--
-- Name: encryption_keys encryption_keys_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.encryption_keys
    ADD CONSTRAINT encryption_keys_pkey PRIMARY KEY (id);


--
-- Name: grants grants_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.grants
    ADD CONSTRAINT grants_pkey PRIMARY KEY (id);


--
-- Name: groups groups_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.groups
    ADD CONSTRAINT groups_pkey PRIMARY KEY (id);


--
-- Name: identities_groups identities_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.identities_groups
    ADD CONSTRAINT identities_groups_pkey PRIMARY KEY (group_id, identity_id);


--
-- Name: identities identities_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.identities
    ADD CONSTRAINT identities_pkey PRIMARY KEY (id);


--
-- Name: migrations migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.migrations
    ADD CONSTRAINT migrations_pkey PRIMARY KEY (id);


--
-- Name: provider_users provider_users_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.provider_users
    ADD CONSTRAINT provider_users_pkey PRIMARY KEY (id);


--
-- Name: providers providers_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.providers
    ADD CONSTRAINT providers_pkey PRIMARY KEY (id);


--
-- Name: root_certificates root_certificates_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.root_certificates
    ADD CONSTRAINT root_certificates_pkey PRIMARY KEY (id);


--
-- Name: settings settings_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.settings
    ADD CONSTRAINT settings_pkey PRIMARY KEY (id);


--
-- Name: trusted_certificates trusted_certificates_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.trusted_certificates
    ADD CONSTRAINT trusted_certificates_pkey PRIMARY KEY (id);


--
-- Name: idx_access_keys_key_id; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX idx_access_keys_key_id ON public.access_keys USING btree (key_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_access_keys_name; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX idx_access_keys_name ON public.access_keys USING btree (name) WHERE (deleted_at IS NULL);


--
-- Name: idx_credentials_identity_id; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX idx_credentials_identity_id ON public.credentials USING btree (identity_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_destinations_unique_id; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX idx_destinations_unique_id ON public.destinations USING btree (unique_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_encryption_keys_key_id; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX idx_encryption_keys_key_id ON public.encryption_keys USING btree (key_id);


--
-- Name: idx_groups_name_provider_id; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX idx_groups_name_provider_id ON public.groups USING btree (name) WHERE (deleted_at IS NULL);


--
-- Name: idx_identities_name; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX idx_identities_name ON public.identities USING btree (name) WHERE (deleted_at IS NULL);


--
-- Name: idx_providers_name; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX idx_providers_name ON public.providers USING btree (name) WHERE (deleted_at IS NULL);


--
-- Name: access_keys fk_access_keys_issued_for_identity; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.access_keys
    ADD CONSTRAINT fk_access_keys_issued_for_identity FOREIGN KEY (issued_for) REFERENCES public.identities(id);


--
-- Name: identities_groups fk_identities_groups_group; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.identities_groups
    ADD CONSTRAINT fk_identities_groups_group FOREIGN KEY (group_id) REFERENCES public.groups(id);


--
-- Name: identities_groups fk_identities_groups_identity; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.identities_groups
    ADD CONSTRAINT fk_identities_groups_identity FOREIGN KEY (identity_id) REFERENCES public.identities(id);


--
-- PostgreSQL database dump complete
--

