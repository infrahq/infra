package model

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestParseCreateTable(t *testing.T) {
	type testCase struct {
		name     string
		table    string
		expected TableDescription
	}

	run := func(t *testing.T, tc testCase) {
		actual, err := ParseCreateTable(tc.table)
		assert.NilError(t, err)

		assert.DeepEqual(t, actual, tc.expected)
	}

	testCases := []testCase{
		{
			name:  "access_keys",
			table: accessKey{}.Schema(),
			expected: TableDescription{
				Name: "access_keys",
				Columns: []Column{
					{Name: "id", DataType: "bigint"},
					{Name: "created_at", DataType: "timestamp"},
					{Name: "updated_at", DataType: "timestamp"},
					{Name: "deleted_at", DataType: "timestamp"},
					{Name: "name", DataType: "text"},
					{Name: "issued_for", DataType: "bigint"},
					{Name: "provider_id", DataType: "bigint"},
					{Name: "scopes", DataType: "text"},
					{Name: "expires_at", DataType: "timestamp"},
					{Name: "extension", DataType: "bigint"},
					{Name: "extension_deadline", DataType: "timestamp"},
					{Name: "key_id", DataType: "text"},
					{Name: "secret_checksum", DataType: "bytea"},
				},
			},
		},
		{
			name:  "destinations",
			table: destination{}.Schema(),
			expected: TableDescription{
				Name: "destinations",
				Columns: []Column{
					{Name: "id", DataType: "bigint"},
					{Name: "created_at", DataType: "timestamp"},
					{Name: "updated_at", DataType: "timestamp"},
					{Name: "deleted_at", DataType: "timestamp"},
					{Name: "name", DataType: "text"},
					{Name: "unique_id", DataType: "text"},
					{Name: "last_seen_at", DataType: "timestamp"},
					{Name: "version", DataType: "text"},
					{Name: "connection_url", DataType: "text"},
					{Name: "connection_ca", DataType: "text"},
					{Name: "resources", DataType: "text"},
					{Name: "roles", DataType: "text"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

type accessKey struct{}

func (a accessKey) Table() string {
	return "access_keys"
}

func (a accessKey) Schema() string {
	return `
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
);`
}

type destination struct{}

func (d destination) Table() string {
	return "destinations"
}

func (d destination) Schema() string {
	return `
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
);`
}
