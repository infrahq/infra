package data

import (
	"fmt"

	"github.com/scim2/filter-parser/v2"

	"github.com/infrahq/infra/internal/server/data/querybuilder"
)

func filterSQL(e filter.Expression, query *querybuilder.Query) error {
	switch v := e.(type) {
	case *filter.LogicalExpression:
		err := filterSQL(v.Left, query)
		if err != nil {
			return fmt.Errorf("left: %w", err)
		}
		switch v.Operator {
		case filter.AND:
			query.B("AND")
		case filter.OR:
			query.B("OR")
		default:
			return fmt.Errorf("unsupported operator %q", v.Operator)
		}
		err = filterSQL(v.Right, query)
		if err != nil {
			return fmt.Errorf("right: %w", err)
		}
		return nil
	case *filter.AttributeExpression:
		err := sqlColumn(v.AttributePath, query)
		if err != nil {
			return fmt.Errorf("attribute path: %w", err)
		}
		err = sqlComparator(v.Operator, v.CompareValue, query)
		if err != nil {
			return fmt.Errorf("attribute comparator: %w", err)
		}
		return nil
	}
	return fmt.Errorf("unable to parse filter, unrecognized format")
}

// sqlColumns maps SCIM input filters to provider user database columns
func sqlColumn(a filter.AttributePath, query *querybuilder.Query) error {
	switch a.String() {
	case "id":
		query.B("identity_id")
	case "userName":
		query.B("email")
	case "email":
		query.B("email")
	case "name.givenName":
		query.B("givenName")
	case "name.familyName":
		query.B("familyName")
	case "active":
		query.B("active")
	default:
		return fmt.Errorf("unsupported filter attribute: %q", a)
	}
	return nil
}

func sqlComparator(c filter.CompareOperator, compare any, query *querybuilder.Query) error {
	switch c {
	case filter.PR:
		query.B("IS NOT NULL")
	case filter.EQ:
		query.B("= ?", compare)
	case filter.NE:
		query.B("!= ?", compare)
	case filter.SW:
		cmp, ok := compare.(string)
		if !ok {
			return fmt.Errorf("upsupported match comparator: %q", c)
		}
		query.B("LIKE ?", cmp+"%")
	case filter.CO:
		cmp, ok := compare.(string)
		if !ok {
			return fmt.Errorf("upsupported match comparator: %q", c)
		}
		query.B("LIKE ?", "%"+cmp+"%")
	case filter.EW:
		cmp, ok := compare.(string)
		if !ok {
			return fmt.Errorf("upsupported match comparator: %q", c)
		}
		query.B("LIKE ?", "%"+cmp)
	case filter.GE, filter.GT, filter.LE, filter.LT:
		return fmt.Errorf("upsupported comparator: %q", c)
	}

	return nil
}
