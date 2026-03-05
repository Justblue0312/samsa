package queryparam

import (
	"fmt"
	"reflect"
	"strings"
)

// ═══════════════════════════════════════════════════════════════════════════════
// SQL dialect
// ═══════════════════════════════════════════════════════════════════════════════

// Dialect controls how placeholder tokens are rendered in SQL output.
type Dialect int

const (
	// DialectMySQL uses positional ? placeholders: WHERE age >= ?
	DialectMySQL Dialect = iota

	// DialectPostgres uses numbered $N placeholders: WHERE age >= $1
	DialectPostgres
)

// opToSQL maps each CompareOp to its SQL operator string.
var opToSQL = map[CompareOp]string{
	OpEq:    "=",
	OpNe:    "!=",
	OpGt:    ">",
	OpGte:   ">=",
	OpLt:    "<",
	OpLte:   "<=",
	OpLike:  "LIKE",
	OpILike: "ILIKE",
	// OpIn is handled separately (IN (...) expansion).
}

// ═══════════════════════════════════════════════════════════════════════════════
// FilterBuilder — accumulates clauses from multiple Filter[T] fields
// ═══════════════════════════════════════════════════════════════════════════════

// FilterBuilder collects WHERE clauses and bound arguments from multiple
// Filter[T] fields. Call Add() for each filter field in your param struct,
// then call Build() to get the combined clause and argument slice.
//
//	b := queryparam.NewFilterBuilder(queryparam.DialectPostgres)
//	params.Price.AppendTo("p.price", b)
//	params.Name.AppendTo("p.name", b)
//	params.Status.AppendTo("p.status", b)
//
//	where, args := b.Build()
//	// where: "p.price >= $1 AND p.price <= $2 AND p.name LIKE $3 AND p.status IN ($4, $5)"
//	// args:  [10.0, 500.0, "%widget%", "active", "banned"]
//
//	query := "SELECT * FROM products"
//	if where != "" {
//	    query += " WHERE " + where
//	}
type FilterBuilder struct {
	dialect Dialect
	clauses []string
	args    []any
	argIdx  int // only used for DialectPostgres
}

// NewFilterBuilder creates a FilterBuilder for the given SQL dialect.
// Defaults to DialectMySQL if no dialect is provided.
func NewFilterBuilder(dialect ...Dialect) *FilterBuilder {
	d := DialectMySQL
	if len(dialect) > 0 {
		d = dialect[0]
	}
	return &FilterBuilder{dialect: d, argIdx: 1}
}

// Build returns the accumulated WHERE clause (without the "WHERE" keyword)
// and the corresponding argument slice, ready to pass to db.Query or db.Exec.
//
// Returns an empty string and nil args when no filters were added.
func (b *FilterBuilder) Build() (clause string, args []any) {
	if len(b.clauses) == 0 {
		return "", nil
	}
	return strings.Join(b.clauses, " AND "), b.args
}

// IsEmpty returns true when no conditions have been added yet.
func (b *FilterBuilder) IsEmpty() bool {
	return len(b.clauses) == 0
}

// placeholder returns the next argument placeholder token for the dialect.
func (b *FilterBuilder) placeholder() string {
	if b.dialect == DialectPostgres {
		p := fmt.Sprintf("$%d", b.argIdx)
		b.argIdx++
		return p
	}
	return "?"
}

// addClause is the internal helper used by AppendTo.
func (b *FilterBuilder) addClause(clause string, arg any) {
	b.clauses = append(b.clauses, clause)
	b.args = append(b.args, arg)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Filter[T].AppendTo — convert one filter field into SQL clauses
// ═══════════════════════════════════════════════════════════════════════════════

// AppendTo converts all conditions in f into SQL clauses and appends them to b.
// col is the column expression to use (e.g. "u.age", "p.price").
//
//	params.Price.AppendTo("p.price", builder)
//	params.Name.AppendTo("p.name", builder)
func (f *Filter[T]) AppendTo(col string, b *FilterBuilder) {
	if f.IsEmpty() {
		return
	}

	// Collect all IN values first so they are grouped into a single IN (...).
	var inValues []any
	for _, c := range f.Conditions {
		if c.Op == OpIn {
			inValues = append(inValues, any(c.Value))
		}
	}

	// Emit non-IN clauses.
	for _, c := range f.Conditions {
		if c.Op == OpIn {
			continue
		}

		sqlOp, ok := opToSQL[c.Op]
		if !ok {
			continue // should never happen after decode validation
		}

		arg := any(c.Value)

		// Wrap LIKE / ILIKE values in % ... % automatically.
		if stringOnlyOps[c.Op] {
			arg = "%" + reflect.ValueOf(arg).String() + "%"
		}

		clause := fmt.Sprintf("%s %s %s", col, sqlOp, b.placeholder())
		b.addClause(clause, arg)
	}

	// Emit a single IN (...) clause for all OpIn conditions.
	if len(inValues) > 0 {
		placeholders := make([]string, len(inValues))
		for i := range inValues {
			placeholders[i] = b.placeholder()
		}
		clause := fmt.Sprintf("%s IN (%s)", col, strings.Join(placeholders, ", "))
		b.clauses = append(b.clauses, clause)
		b.args = append(b.args, inValues...)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Convenience: single-filter ToSQL (no builder needed)
// ═══════════════════════════════════════════════════════════════════════════════

// ToSQL converts a single Filter[T] directly to a WHERE clause fragment and
// its argument slice. Use AppendTo + FilterBuilder when combining multiple fields.
//
//	clause, args := params.Price.ToSQL("p.price", queryparam.DialectPostgres)
//	// "p.price >= $1 AND p.price <= $2",  [10.0, 500.0]
func (f *Filter[T]) ToSQL(col string, dialect ...Dialect) (string, []any) {
	b := NewFilterBuilder(dialect...)
	f.AppendTo(col, b)
	return b.Build()
}
