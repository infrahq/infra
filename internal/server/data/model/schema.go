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

type TableDescription struct {
	Name    string
	Columns []Column
}

func (d TableDescription) ColumnNames() []string {
	c := make([]string, len(d.Columns))
	for i := range d.Columns {
		c[i] = d.Columns[i].Name
	}
	return c
}

// ParseCreateTable parses an SQL CREATE TABLE statement and returns the table name
// and all column names and types. The createTableSQL string is expected to start with
// the following structure. Any [...] can be replaced by any words.
//
//    CREATE TABLE [...] TableName (
//        Column.Name Column.DataType [...] ,
//        <more columns>
//    ) [...]`
//
func ParseCreateTable(schema string) (TableDescription, error) {
	var cols []Column
	var state uint8
	var name string
	words := bufio.NewScanner(strings.NewReader(schema))
	words.Split(bufio.ScanWords)

SCAN:
	for words.Scan() {
		word := words.Text()
		switch state {
		case scanForStartOfColumns:
			if word == "(" {
				state = scanForColumnName
				continue
			}
			name = strings.TrimSuffix(word, "(") // store the last word before "("

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
			if strings.HasSuffix(word, ")") || strings.HasSuffix(word, ");") {
				break SCAN // end of possible columns
			}
		}
	}

	return TableDescription{Name: name, Columns: cols}, words.Err()
}

const (
	scanForStartOfColumns = iota
	scanForColumnName
	scanForColumnDataType
	scanForEndOfColumnDefinition
)
