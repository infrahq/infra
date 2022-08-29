package example

import (
	"fmt"

	"github.com/infrahq/infra/internal/server/data/querybuilder"
)

func ListThings() {
	qb := querybuilder.NewQuery("ok")

	querybuilder.NewQuery("bad" + "concat")         // want `argument to NewQuery must be a string literal`
	querybuilder.NewQuery(fmt.Sprintf("func call")) // want `argument to NewQuery must be a string literal`
	querybuilder.NewQuery("lit" + giveStr())        // want `argument to NewQuery must be a string literal`
	querybuilder.NewQuery(giveStr())                // want `argument to NewQuery must be a string literal`
	querybuilder.NewQuery(couldBeFromAnywhere)      // want `argument to NewQuery must be a string literal`

	qb.B("ok")
	qb.B(fmt.Sprintf("func call")) // want `argument to Builder.B must be a string literal`
	qb.B("lit" + giveStr())        // want `argument to Builder.B must be a string literal`
	qb.B(giveStr())                // want `argument to Builder.B must be a string literal`
	qb.B(couldBeFromAnywhere)      // want `argument to Builder.B must be a string literal`

	nQ := querybuilder.NewQuery // want `NewQuery must be called directly`
	nQ(couldBeFromAnywhere)
	receiveConstructFunc(querybuilder.NewQuery) // want `NewQuery must be called directly`

	b := qb.B // want `Builder.B must be called directly`
	b(couldBeFromAnywhere)
	receiveQueryBuilderFunc(qb.B) // want `Builder.B must be called directly`
}

func giveStr() string {
	return "something"
}

var couldBeFromAnywhere string

func receiveConstructFunc(_ func(string) *querybuilder.Builder) {}

func receiveQueryBuilderFunc(_ func(string, ...any)) {}

func sneakyInjection(qb *querybuilder.Builder) {
	qb.B(couldBeFromAnywhere) // want `argument to Builder.B must be a string literal`
}
