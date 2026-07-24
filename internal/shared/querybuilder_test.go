package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewQueryBuilder(t *testing.T) {
	qb := NewQueryBuilder()
	assert.NotNil(t, qb)
	assert.Equal(t, []string{"1=1"}, qb.WhereClauses)
	assert.Equal(t, 1, qb.ArgIdx)
	assert.Empty(t, qb.Args)
}

func TestQueryBuilder_AddClause(t *testing.T) {
	t.Run("single clause with args", func(t *testing.T) {
		qb := NewQueryBuilder()
		qb.AddClause(" AND name ILIKE $%d", "%search%")
		assert.Equal(t, "1=1  AND name ILIKE $1", qb.Where())
		assert.Equal(t, []interface{}{"%search%"}, qb.Args)
		assert.Equal(t, 2, qb.ArgIdx)
	})

	t.Run("multiple clauses", func(t *testing.T) {
		qb := NewQueryBuilder()
		qb.AddClause(" AND name ILIKE $%d", "%search%")
		qb.AddClause(" AND price >= $%d", 100)
		assert.Equal(t, "1=1  AND name ILIKE $1  AND price >= $2", qb.Where())
		assert.Equal(t, []interface{}{"%search%", 100}, qb.Args)
	})

	t.Run("no args does not add clause", func(t *testing.T) {
		qb := NewQueryBuilder()
		qb.AddClause(" AND active = true")
		assert.Equal(t, "1=1", qb.Where())
		assert.Empty(t, qb.Args)
		assert.Equal(t, 1, qb.ArgIdx)
	})

	t.Run("blank clause with no args is ignored", func(t *testing.T) {
		qb := NewQueryBuilder()
		qb.AddClause("")
		assert.Equal(t, "1=1", qb.Where())
	})
}

func TestQueryBuilder_Where(t *testing.T) {
	t.Run("default returns 1=1", func(t *testing.T) {
		qb := NewQueryBuilder()
		assert.Equal(t, "1=1", qb.Where())
	})

	t.Run("single clause", func(t *testing.T) {
		qb := NewQueryBuilder()
		qb.AddClause(" AND status = $%d", "active")
		assert.Equal(t, "1=1  AND status = $1", qb.Where())
	})
}
