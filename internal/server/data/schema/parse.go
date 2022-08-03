package schema

import (
	"bufio"
	"fmt"
	"strings"
)

type Statement struct {
	TableName string
	Value     string
}

// ParseSchema parses a string that contains PostgreSQL DDL statements. The parse
// is very limited. Only statements in the form produced by pg_dump are supported.
//
// Statements are returned grouped by the name of the table they apply to.
//
// TODO: add support for statements that are not associated with a single table
// in an "extras" section, when we start to use those statements.
func ParseSchema(schema string) ([]Statement, error) {
	var state uint8
	var currentTable string
	var currentStmt strings.Builder
	var stmts []Statement

	endStatement := func() {
		stmts = append(stmts, Statement{
			TableName: currentTable,
			Value:     currentStmt.String(),
		})
		currentTable = ""
		currentStmt.Reset()
		state = scanSchemaForStartOfStatement
	}

	lines := bufio.NewScanner(strings.NewReader(schema))
	lines.Split(bufio.ScanLines)
	for lines.Scan() {
		line := lines.Text()

		// skip comments and empty lines
		if strings.HasPrefix(line, "--") || line == "" {
			continue
		}

		switch state {
		case scanSchemaForStartOfStatement:
			if isClientConnectionSetting(line) {
				continue
			}
			if strings.HasPrefix(line, "CREATE SCHEMA") {
				continue
			}

			var err error
			if currentTable, line, err = parseStatementLine(line); err != nil {
				return nil, err
			}

			currentStmt.WriteString("\n" + line + "\n")
			if strings.HasSuffix(line, ";") {
				endStatement()
				continue
			}
			state = scanSchemaForEndOfStatement

		case scanSchemaForEndOfStatement:
			line = strings.Replace(line, schemaNamePrefix, "", -1)
			currentStmt.WriteString(line + "\n")

			if strings.HasSuffix(line, ";") {
				endStatement()
			}
		}
	}
	return stmts, lines.Err()
}

const (
	scanSchemaForStartOfStatement = iota
	scanSchemaForEndOfStatement   = iota
)

func isClientConnectionSetting(line string) bool {
	switch {
	case strings.HasPrefix(line, "SET "):
		return true
	case strings.HasPrefix(line, "SELECT pg_catalog.set_config"):
		return true
	}
	return false
}

const schemaNamePrefix = "testing."

// parseStatementLine parses the first line of a PostgreSQL statement. The parse
// is very limited. Only statements in the form produced by pg_dump are supported.
// the parse is also not complete. It only supports statements we use. If new
// statements (views, triggers, etc.) are added this function will need to be updated.
//
// If the statement is not recognized an error is returned.
func parseStatementLine(line string) (table, parsed string, err error) {
	switch {
	case strings.HasPrefix(line, "CREATE TABLE"):
	case strings.HasPrefix(line, "ALTER TABLE"):
	case strings.HasPrefix(line, "CREATE SEQUENCE"):
	case strings.HasPrefix(line, "ALTER SEQUENCE"):
	case strings.HasPrefix(line, "CREATE UNIQUE INDEX"):
	default:
		return "", "", fmt.Errorf("unexpected start of statement: %q", line)
	}

	var result strings.Builder
	words := bufio.NewScanner(strings.NewReader(line))
	words.Split(bufio.ScanWords)
	for words.Scan() {
		word := words.Text()
		if table != "" {
			word = strings.Replace(word, schemaNamePrefix, "", -1)
			result.WriteString(word + " ")
			continue
		}

		_, trimmed, found := strings.Cut(word, schemaNamePrefix)
		if !found {
			result.WriteString(word + " ")
			continue
		}
		table = trimmed

		// Trim id_seq suffix from ALTER/CREATE SEQUENCE statements. This assumes
		// sequences are only created for fields called id.
		if seqTable, _, found := strings.Cut(trimmed, "_id_seq"); found {
			table = seqTable
		}
		result.WriteString(trimmed + " ")
	}
	if table == "" {
		return "", "", fmt.Errorf("failed to parse table name from: %q", line)
	}
	return table, strings.TrimSuffix(result.String(), " "), words.Err()
}
