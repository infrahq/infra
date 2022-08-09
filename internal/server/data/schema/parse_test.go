package schema

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestParseSchema(t *testing.T) {
	raw := golden.Get(t, "full-schema.sql")

	stmts, err := ParseSchema(string(raw))
	assert.NilError(t, err)

	expected := []Statement{
		{
			TableName: "flowers",
			Value: `
CREATE TABLE flowers (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text
);
`,
		},
		{
			TableName: "flowers",
			Value: `
CREATE SEQUENCE flowers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
`,
		},
		{
			TableName: "flowers",
			Value: `
ALTER SEQUENCE flowers_id_seq OWNED BY flowers.id;
`,
		},
		{
			TableName: "stars",
			Value: `
CREATE TABLE stars (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text
);
`,
		},
		{
			TableName: "flowers",
			Value: `
ALTER TABLE ONLY flowers ALTER COLUMN id SET DEFAULT nextval('flowers_id_seq'::regclass);
`,
		},
		{
			TableName: "stars",
			Value: `
ALTER TABLE ONLY stars
    ADD CONSTRAINT stars_pkey PRIMARY KEY (id);
`,
		},
		{
			TableName: "stars",
			Value: `
CREATE UNIQUE INDEX idx_stars_unique_id ON stars USING btree (unique_id) WHERE (deleted_at IS NULL);
`,
		},
		{
			TableName: "flowers",
			Value: `
ALTER TABLE ONLY flowers
    ADD CONSTRAINT fk_flowers_issued_for_identity FOREIGN KEY (issued_for) REFERENCES identities(id);
`,
		},
	}

	assert.DeepEqual(t, stmts, expected)
}

func TestParseStatementLine(t *testing.T) {
	type testCase struct {
		name          string
		line          string
		expectedTable string
		expectedLine  string
	}

	run := func(t *testing.T, tc testCase) {
		table, actual, err := parseStatementLine(tc.line)
		assert.NilError(t, err)

		assert.Equal(t, table, tc.expectedTable, tc.line)
		assert.Equal(t, actual, tc.expectedLine, tc.line)
	}

	testCases := []testCase{
		{
			name:          "CREATE TABLE",
			line:          `CREATE TABLE testing.grants (`,
			expectedTable: "grants",
			expectedLine:  `CREATE TABLE grants (`,
		},
		{
			name:          "CREATE SEQUENCE",
			line:          `CREATE SEQUENCE testing.grants_id_seq`,
			expectedTable: "grants",
			expectedLine:  `CREATE SEQUENCE grants_id_seq`,
		},
		{
			name:          "ALTER SEQUENCE",
			line:          `ALTER SEQUENCE testing.grants_id_seq OWNED BY testing.grants.id;`,
			expectedTable: "grants",
			expectedLine:  `ALTER SEQUENCE grants_id_seq OWNED BY grants.id;`,
		},
		{
			name:          "ALTER TABLE set default",
			line:          `ALTER TABLE ONLY testing.stars ALTER COLUMN id SET DEFAULT nextval('testing.stars_id_seq'::regclass);`,
			expectedTable: "stars",
			expectedLine:  `ALTER TABLE ONLY stars ALTER COLUMN id SET DEFAULT nextval('stars_id_seq'::regclass);`,
		},
		{
			name:          "CREATE INDEX",
			line:          `CREATE UNIQUE INDEX idx_flowers_key_id ON testing.flowers USING btree (key_id) WHERE (deleted_at IS NULL);`,
			expectedTable: "flowers",
			expectedLine:  `CREATE UNIQUE INDEX idx_flowers_key_id ON flowers USING btree (key_id) WHERE (deleted_at IS NULL);`,
		},
		{
			name:          "ALTER TABLE add constraint",
			line:          `ALTER TABLE ONLY testing.identities_organizations`,
			expectedTable: "identities_organizations",
			expectedLine:  `ALTER TABLE ONLY identities_organizations`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
