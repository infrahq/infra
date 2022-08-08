package model

import (
	"bufio"
	"strings"
)

type Column struct {
	// Name of the column
	Name string
	// DataType is the sql data type of the column.
	DataType string
}

// ParseCreateTable parses a string that contains PostgreSQL statements and
// returns a mapping of table name to column names and types. The parse
// is very limited. Only statements in the form produced by schema.ParseSchema
// are supported.
func ParseCreateTable(schema string) (map[string][]Column, error) {
	result := make(map[string][]Column)
	var state uint8
	var tableName string
	var cols []Column

	endStatement := func() {
		if tableName == "" {
			return
		}
		result[tableName] = cols
		cols = nil
		tableName = ""
	}

	words := bufio.NewScanner(strings.NewReader(schema))
	words.Split(bufio.ScanWords)

	for words.Scan() {
		word := words.Text()

		switch state {
		case scanForStartOfStatement:
			if word != "CREATE" {
				continue
			}
			state = scanForCreateType

		case scanForCreateType:
			switch word {
			case "TABLE":
				state = scanForStartOfColumns
				continue
			case "GLOBAL", "LOCAL", "TEMPORARY", "TEMP", "UNLOGGED":
				// keep looking for type of create
			default:
				// not a CREATE TABLE statement
				state = scanForEndOfStatement
			}

		case scanForStartOfColumns:
			if word == "(" {
				state = scanForColumnName
				continue
			}
			tableName = word // store the last word before "("

		case scanForColumnName:
			cols = append(cols, Column{Name: word})
			state = scanForColumnDataType

		case scanForColumnDataType:
			cols[len(cols)-1].DataType = strings.TrimSuffix(word, ",")

			if strings.HasSuffix(word, ",") { // new column
				state = scanForColumnName
				continue
			}
			state = scanForEndOfColumnDefinition

		case scanForEndOfColumnDefinition: // , or )
			if strings.HasSuffix(word, ",") { // new column
				state = scanForColumnName
				continue
			}
			if strings.HasSuffix(word, ")") {
				state = scanForEndOfStatement
				continue
			}
			if strings.HasSuffix(word, ";") {
				endStatement()
				state = scanForStartOfStatement
			}

		case scanForEndOfStatement:
			if strings.HasSuffix(word, ";") {
				endStatement()
				state = scanForStartOfStatement
			}
		}
	}
	return result, words.Err()
}

const (
	scanForStartOfStatement = iota
	scanForCreateType
	scanForStartOfColumns
	scanForColumnName
	scanForColumnDataType
	scanForEndOfColumnDefinition
	scanForEndOfStatement
)
