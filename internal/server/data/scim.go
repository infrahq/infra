package data

import (
	"fmt"
	"strings"

	"github.com/scim2/filter-parser/v2"
)

// supportedColumns maps SCIM input filters to provider user database columns
var supportedColumns = map[string]string{
	"id":              "identity_id",
	"userName":        "email",
	"email":           "email",
	"name.givenName":  "givenName",
	"name.familyName": "familyName",
	"active":          "active",
}

func filterSQL(e filter.Expression) (string, error) {
	switch v := e.(type) {
	case *filter.LogicalExpression:
		l, err := filterSQL(v.Left)
		if err != nil {
			return "", fmt.Errorf("left: %w", err)
		}
		r, err := filterSQL(v.Right)
		if err != nil {
			return "", fmt.Errorf("right: %w", err)
		}
		return fmt.Sprintf("%s %s %s", l, strings.ToUpper(string(v.Operator)), r), nil
	case *filter.AttributeExpression:
		comparison, err := sqlComparator(v.Operator, v.CompareValue)
		if err != nil {
			return "", fmt.Errorf("attribute comparator: %w", err)
		}
		column, err := sqlColumn(v.AttributePath)
		if err != nil {
			return "", fmt.Errorf("attribute path: %w", err)
		}
		return fmt.Sprintf("%s %s", column, comparison), nil
	}
	return "", fmt.Errorf("unable to parse filter, unrecognized format")
}

func sqlColumn(a filter.AttributePath) (string, error) {
	if supportedColumns[a.String()] == "" {
		return "", fmt.Errorf("unsupported filter attribute: %q", a)
	}
	return supportedColumns[a.String()], nil
}

func sqlComparator(c filter.CompareOperator, compare any) (string, error) {
	switch {
	case c == filter.PR:
		return "IS NOT NULL", nil
	case c == filter.EQ:
		return fmt.Sprintf("= '%s'", compare), nil
	case c == filter.NE:
		return fmt.Sprintf("!= '%s'", compare), nil
	case c == filter.SW:
		return fmt.Sprintf("LIKE '%s%%'", compare), nil
	case c == filter.CO:
		return fmt.Sprintf("LIKE '%%%s%%'", compare), nil
	case c == filter.EW:
		return fmt.Sprintf("LIKE '%%%s'", compare), nil
	}
	return "", fmt.Errorf("upsupported comparator: %q", c)
}
