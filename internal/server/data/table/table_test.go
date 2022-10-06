package table

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestParseCreateTable(t *testing.T) {
	actual, err := ParseCreateTable(testSchema)
	assert.NilError(t, err)

	expected := map[string][]Column{
		"access_keys": {
			{Name: "id", DataType: "bigint"},
			{Name: "issued_for", DataType: "bigint"},
			{Name: "provider_id", DataType: "bigint"},
			{Name: "expires_at", DataType: "timestamp"},
		},
		"destinations": {
			{Name: "id", DataType: "bigint"},
			{Name: "name", DataType: "text"},
			{Name: "url", DataType: "text"},
		},
		"pets": {
			{Name: "id", DataType: "bigint"},
			{Name: "name", DataType: "text"},
			{Name: "species", DataType: "text"},
		},
	}
	assert.DeepEqual(t, actual, expected)
}

var testSchema = `

CREATE TABLE access_keys (
    id bigint NOT NULL,
    issued_for bigint,
    provider_id bigint,
    expires_at timestamp with time zone
);

CREATE SEQUENCE access_keys_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE access_keys_id_seq OWNED BY access_keys.id;

CREATE TABLE destinations (
    id bigint NOT NULL,
    name text,
    url text
);

CREATE TABLE pets (
    id bigint NOT NULL,
    name text,
    species text
);

ALTER TABLE ONLY grants ALTER COLUMN id SET DEFAULT nextval('grants_id_seq'::regclass);

ALTER TABLE ONLY encryption_keys
    ADD CONSTRAINT encryption_keys_pkey PRIMARY KEY (id);

CREATE UNIQUE INDEX idx_identities_name ON identities USING btree (name) WHERE (deleted_at IS NULL);

ALTER TABLE ONLY identities_groups
    ADD CONSTRAINT fk_identities_groups_identity FOREIGN KEY (identity_id) REFERENCES identities(id);

`
