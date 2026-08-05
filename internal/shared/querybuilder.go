package shared

import (
	"fmt"
	"strings"
)

type QueryBuilder struct {
	WhereClauses []string
	Args         []interface{}
	ArgIdx       int
}

func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		WhereClauses: []string{"1=1"},
		ArgIdx:       1,
	}
}

func (qb *QueryBuilder) AddClause(clause string, args ...interface{}) {
	if len(args) > 0 {
		sprintfArgs := make([]interface{}, len(args))
		for i := range args {
			sprintfArgs[i] = qb.ArgIdx + i
		}
		qb.WhereClauses = append(qb.WhereClauses, fmt.Sprintf(clause, sprintfArgs...))
		qb.Args = append(qb.Args, args...)
		qb.ArgIdx += len(args)
	}
}

func (qb *QueryBuilder) Where() string {
	return strings.Join(qb.WhereClauses, " ")
}
