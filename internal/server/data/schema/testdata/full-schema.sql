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
-- Name: testing; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA testing;


SET default_table_access_method = heap;

--
-- Name: flowers; Type: TABLE; Schema: testing; Owner: -
--

CREATE TABLE testing.flowers (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text
);


--
-- Name: flowers_id_seq; Type: SEQUENCE; Schema: testing; Owner: -
--

CREATE SEQUENCE testing.flowers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: flowers_id_seq; Type: SEQUENCE OWNED BY; Schema: testing; Owner: -
--

ALTER SEQUENCE testing.flowers_id_seq OWNED BY testing.flowers.id;


--
-- Name: stars; Type: TABLE; Schema: testing; Owner: -
--

CREATE TABLE testing.stars (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text
);


--
-- Name: flowers id; Type: DEFAULT; Schema: testing; Owner: -
--

ALTER TABLE ONLY testing.flowers ALTER COLUMN id SET DEFAULT nextval('testing.flowers_id_seq'::regclass);


--
-- Name: stars stars_pkey; Type: CONSTRAINT; Schema: testing; Owner: -
--

ALTER TABLE ONLY testing.stars
    ADD CONSTRAINT stars_pkey PRIMARY KEY (id);


--
-- Name: idx_stars_unique_id; Type: INDEX; Schema: testing; Owner: -
--

CREATE UNIQUE INDEX idx_stars_unique_id ON testing.stars USING btree (unique_id) WHERE (deleted_at IS NULL);


--
-- Name: flowers fk_flowers_issued_for_identity; Type: FK CONSTRAINT; Schema: testing; Owner: -
--

ALTER TABLE ONLY testing.flowers
    ADD CONSTRAINT fk_flowers_issued_for_identity FOREIGN KEY (issued_for) REFERENCES testing.identities(id);


--
-- PostgreSQL database dump complete
--

