package example

import (
	"fmt"
	"strings"
)

// copied from data package
type queryBuilder struct {
	query strings.Builder
}

func newQuery(stmt string) *queryBuilder {
	q := &queryBuilder{}
	q.query.WriteString(stmt + " ")
	return q
}

func (q *queryBuilder) B(clause string, args ...interface{}) {
	q.query.WriteString(clause + " ")
}

func ListThings() {
	qb := newQuery("ok")

	newQuery("bad" + "concat")         // want `argument to newQuery must be a string literal`
	newQuery(fmt.Sprintf("func call")) // want `argument to newQuery must be a string literal`
	newQuery("lit" + giveStr())        // want `argument to newQuery must be a string literal`
	newQuery(giveStr())                // want `argument to newQuery must be a string literal`
	newQuery(couldBeFromAnywhere)      // want `argument to newQuery must be a string literal`

	qb.B("ok")
	qb.B(fmt.Sprintf("func call")) // want `argument to queryBuilder.B must be a string literal`
	qb.B("lit" + giveStr())        // want `argument to queryBuilder.B must be a string literal`
	qb.B(giveStr())                // want `argument to queryBuilder.B must be a string literal`
	qb.B(couldBeFromAnywhere)      // want `argument to queryBuilder.B must be a string literal`

	nQ := newQuery // want `newQuery must be called directly`
	nQ(couldBeFromAnywhere)
	receiveConstructFunc(newQuery) // want `newQuery must be called directly`

	b := qb.B // want `queryBuilder.B must be called directly`
	b(couldBeFromAnywhere)
	receiveQueryBuilderFunc(qb.B) // want `queryBuilder.B must be called directly`
}

func giveStr() string {
	return "something"
}

var couldBeFromAnywhere string

func receiveConstructFunc(fn func(string) *queryBuilder) {}

func receiveQueryBuilderFunc(fn func(string, ...any)) {}
