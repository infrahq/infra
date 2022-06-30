--
-- PostgreSQL database dump
--

-- Dumped from database version 14.2 (Debian 14.2-1.pgdg110+1)
-- Dumped by pg_dump version 14.2 (Debian 14.2-1.pgdg110+1)

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
-- Name: migrations; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE testing.migrations (
    id character varying(255) NOT NULL
);


ALTER TABLE testing.migrations OWNER TO postgres;

--
-- Name: providers; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE testing.providers (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text,
    url text,
    client_id text,
    client_secret text,
    kind text,
    created_by bigint
);


ALTER TABLE testing.providers OWNER TO postgres;

--
-- Name: providers_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE testing.providers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE testing.providers_id_seq OWNER TO postgres;

--
-- Name: providers_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE testing.providers_id_seq OWNED BY testing.providers.id;


--
-- Name: providers id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY testing.providers ALTER COLUMN id SET DEFAULT nextval('testing.providers_id_seq'::regclass);


--
-- Data for Name: migrations; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY testing.migrations (id) FROM stdin;
SCHEMA_INIT
202203231621
202203241643
202203301642
202203301652
202203301643
202203301645
202203301646
202203301647
202203301648
202204061643
202204111503
202204181613
202204211705
202204281130
202204291613
202206081027
202206151027
202206161733
\.


--
-- Data for Name: providers; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY testing.providers (id, created_at, updated_at, deleted_at, name, url, client_id, client_secret, kind, created_by) FROM stdin;
64820631565574144	2022-06-28 20:54:02.8749+00	2022-06-28 20:54:02.8749+00	\N	okta	example.okta.com	something	AAAAGATRE6Fo/SgPgUZcOfqjFkHGmZuwwJcVrAZhZXNnY20EX+Q8WCYvdmFyL2xpYi9pbmZyYWhxL3NlcnZlci9zcWxpdGUzLmRiLmtleQwZo2b2fhrBmLLPm2A	okta	1
64820631578157056	2022-06-28 20:54:02.877415+00	2022-06-28 20:54:02.877415+00	\N	infra			AAAAEB6xj0FEgRoB8/MhvCgHQEQGYWVzZ2NtBF/kPFgmL3Zhci9saWIvaW5mcmFocS9zZXJ2ZXIvc3FsaXRlMy5kYi5rZXkMGwEKLrM74+XQILWf	infra	1
\.


--
-- Name: providers_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('testing.providers_id_seq', 1, false);


--
-- Name: migrations migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY testing.migrations
    ADD CONSTRAINT migrations_pkey PRIMARY KEY (id);


--
-- Name: providers providers_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY testing.providers
    ADD CONSTRAINT providers_pkey PRIMARY KEY (id);


--
-- Name: idx_providers_name; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX idx_providers_name ON testing.providers USING btree (name) WHERE (deleted_at IS NULL);


--
-- PostgreSQL database dump complete
--

