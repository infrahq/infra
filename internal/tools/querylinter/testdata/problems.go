package example

import (
	"fmt"

	"github.com/infrahq/infra/internal/server/data/querybuilder"
)

func Problems() {
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

	qb.B(otherSignatures{}.Table(couldBeFromAnywhere)) // want `argument to Builder.B must be a string literal`
}

func GoodExamples() {
	qb := querybuilder.NewQuery("ok")

	table := goodExample{}
	qb.B(table.Table())
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

type exampleOne struct{}

func (exampleOne) Table() string { // want `Table method must only return a string literal`
	a := "something"
	return a
}

func (exampleOne) Columns() []string { // want `Columns method must only return a single slice literal`
	// comments are ok
	a := []string{}
	return a
}

type exampleTwo struct{}

func (exampleTwo) Table() string {
	return couldBeFromAnywhere // want `Table method must return a string literal`
}

func (exampleTwo) Columns() []string {
	return exampleOne{}.Columns() // want `Columns method must return a slice literal`
}

type exampleThree struct{}

func (exampleThree) Table() string {
	return giveStr() // want `Table method must return a string literal`
}

func (exampleThree) Columns() []string {
	return []string{
		"ok",
		couldBeFromAnywhere, // want `Columns method return value must contain only string literals`
		giveStr(),           // want `Columns method return value must contain only string literals`
	}
}

type goodExample struct{}

func (goodExample) Table() string {
	return "this is ok"
}

func (goodExample) Columns() []string {
	return []string{"this", "is", "ok"}
}

type otherSignatures struct{}

func (otherSignatures) Table(v string) string {
	return v
}
