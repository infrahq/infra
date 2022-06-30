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
-- Name: migrations; Type: TABLE; Schema: testing; Owner: postgres
--

CREATE TABLE testing.migrations (
    id character varying(255) NOT NULL
);


ALTER TABLE testing.migrations OWNER TO postgres;

--
-- Name: providers; Type: TABLE; Schema: testing; Owner: postgres
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
    created_by bigint,
    kind text,
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
-- Name: providers id; Type: DEFAULT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.providers ALTER COLUMN id SET DEFAULT nextval('testing.providers_id_seq'::regclass);


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


--
-- Data for Name: providers; Type: TABLE DATA; Schema: testing; Owner: postgres
--

INSERT INTO testing.providers (id, created_at, updated_at, deleted_at, name, url, client_id, client_secret, created_by, kind) VALUES (63423113657131008, '2022-06-25 00:20:48.640037+00', '2022-06-25 00:20:48.640037+00', NULL, 'infra', '', '', 'AAAAEC8bWPoKbfyEQ/0dUXXXobAGYWVzZ2NtBMNQas4bL3Jvb3QvLmluZnJhL3NxbGl0ZTMuZGIua2V5DH+uGwwZiaRaeBM7/w', 1, 'infra');
INSERT INTO testing.providers (id, created_at, updated_at, deleted_at, name, url, client_id, client_secret, created_by, kind) VALUES (63423113657131009, '2022-06-25 00:20:48.640037+00', '2022-06-25 00:20:48.640037+00', NULL, 'okta', 'example.okta.com', 'client-id', 'AAAAEC8bWPoKbfyEQ/0dUXXXobAGYWVzZ2NtBMNQas4bL3Jvb3QvLmluZnJhL3NxbGl0ZTMuZGIua2V5DH+uGwwZiaRaeBM7/w', 1, 'okta');


--
-- Name: providers_id_seq; Type: SEQUENCE SET; Schema: testing; Owner: postgres
--

SELECT pg_catalog.setval('testing.providers_id_seq', 1, false);


--
-- Name: migrations migrations_pkey; Type: CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.migrations
    ADD CONSTRAINT migrations_pkey PRIMARY KEY (id);


--
-- Name: providers providers_pkey; Type: CONSTRAINT; Schema: testing; Owner: postgres
--

ALTER TABLE ONLY testing.providers
    ADD CONSTRAINT providers_pkey PRIMARY KEY (id);


--
-- Name: idx_providers_name; Type: INDEX; Schema: testing; Owner: postgres
--

CREATE UNIQUE INDEX idx_providers_name ON testing.providers USING btree (name) WHERE (deleted_at IS NULL);


--
-- PostgreSQL database dump complete
--
