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
		if strings.HasPrefix(line, "--") {
			continue
		}

		switch state {
		case scanSchemaForStartOfStatement:
			if line == "" {
				continue
			}
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

		case scanSchemaInCodeBlock:
			currentStmt.WriteString(line + "\n")

			if strings.HasSuffix(line, "$$;") {
				endStatement()
			}
		case scanSchemaForEndOfStatement:
			if line == "" {
				continue
			}

			line = strings.ReplaceAll(line, schemaNamePrefix, "")
			currentStmt.WriteString(line + "\n")

			if strings.HasSuffix(line, "AS $$") {
				state = scanSchemaInCodeBlock
				continue
			}
			if strings.HasSuffix(line, ";") {
				endStatement()
			}
		}
	}
	return stmts, lines.Err()
}

const (
	scanSchemaForStartOfStatement = iota
	scanSchemaForEndOfStatement
	scanSchemaInCodeBlock
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
	case strings.HasPrefix(line, "CREATE INDEX"):
	case strings.HasPrefix(line, "CREATE FUNCTION"):
	case strings.HasPrefix(line, "CREATE TRIGGER"):
	default:
		return "", "", fmt.Errorf("unexpected start of statement: %q", line)
	}

	var result strings.Builder
	words := bufio.NewScanner(strings.NewReader(line))
	words.Split(bufio.ScanWords)
	for words.Scan() {
		word := words.Text()
		if table != "" {
			word = strings.ReplaceAll(word, schemaNamePrefix, "")
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

// TrimComments removes blank lines, comment lines, and SET lines from a
// pg_dump output. The --no-comments flag to pg_dump only removes some comments,
// and there does not appear to be any way to remove the SET lines.
func TrimComments(input string) (string, error) {
	lines := bufio.NewScanner(strings.NewReader(input))
	lines.Split(bufio.ScanLines)
	var buf strings.Builder

	for lines.Scan() {
		line := lines.Text()
		switch {
		case strings.TrimSpace(line) == "":
			continue
		case strings.HasPrefix(line, "--"):
			continue
		case strings.HasPrefix(line, "SET "):
			continue
		case strings.HasPrefix(line, "SELECT pg_catalog.set_config"):
			continue
		case strings.HasPrefix(line, "SELECT pg_catalog.setval"):
			continue
		}
		buf.WriteString(line + "\n")
	}
	return buf.String(), lines.Err()
}
