package queryparam

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPagination_AllowOrderWithSQLC(t *testing.T) {
	t.Run("valid asc entry", func(t *testing.T) {
		params := &PaginationParams{}
		params.Normalize(AllowOrderWithSQLC([]string{"created_at_asc"}))

		assert.Len(t, params.orderEntries, 1)
		assert.Equal(t, "created_at", params.orderEntries[0].column)
		assert.Equal(t, "ASC", params.orderEntries[0].seq)
	})

	t.Run("valid desc entry", func(t *testing.T) {
		params := &PaginationParams{}
		params.Normalize(AllowOrderWithSQLC([]string{"updated_at_desc"}))

		assert.Len(t, params.orderEntries, 1)
		assert.Equal(t, "updated_at", params.orderEntries[0].column)
		assert.Equal(t, "DESC", params.orderEntries[0].seq)
	})

	t.Run("multiple entries", func(t *testing.T) {
		params := &PaginationParams{}
		params.Normalize(AllowOrderWithSQLC([]string{
			"created_at_asc",
			"updated_at_desc",
		}))

		assert.Len(t, params.orderEntries, 2)
		assert.Equal(t, "created_at", params.orderEntries[0].column)
		assert.Equal(t, "ASC", params.orderEntries[0].seq)
		assert.Equal(t, "updated_at", params.orderEntries[1].column)
		assert.Equal(t, "DESC", params.orderEntries[1].seq)
	})

	t.Run("case insensitive direction", func(t *testing.T) {
		params := &PaginationParams{}
		params.Normalize(AllowOrderWithSQLC([]string{"created_at_DESC"}))

		assert.Len(t, params.orderEntries, 1)
		assert.Equal(t, "DESC", params.orderEntries[0].seq)
	})

	t.Run("invalid direction ignored", func(t *testing.T) {
		params := &PaginationParams{}
		params.Normalize(AllowOrderWithSQLC([]string{"created_at_invalid"}))

		assert.Len(t, params.orderEntries, 0)
	})

	t.Run("no underscore ignored", func(t *testing.T) {
		params := &PaginationParams{}
		params.Normalize(AllowOrderWithSQLC([]string{"created_at"}))

		assert.Len(t, params.orderEntries, 0)
	})

	t.Run("empty string ignored", func(t *testing.T) {
		params := &PaginationParams{}
		params.Normalize(AllowOrderWithSQLC([]string{""}))

		assert.Len(t, params.orderEntries, 0)
	})

	t.Run("duplicate fields - first wins", func(t *testing.T) {
		params := &PaginationParams{}
		params.Normalize(AllowOrderWithSQLC([]string{
			"created_at_asc",
			"created_at_desc",
		}))

		assert.Len(t, params.orderEntries, 1)
		assert.Equal(t, "ASC", params.orderEntries[0].seq)
	})

	t.Run("empty array", func(t *testing.T) {
		params := &PaginationParams{}
		params.Normalize(AllowOrderWithSQLC([]string{}))

		assert.Len(t, params.orderEntries, 0)
	})

	t.Run("nil array", func(t *testing.T) {
		params := &PaginationParams{}
		params.Normalize(AllowOrderWithSQLC(nil))

		assert.Len(t, params.orderEntries, 0)
	})
}

func TestPagination_AllowOrderWithSQLC_Precedence(t *testing.T) {
	t.Run("AllowOrderWithSQLC takes precedence over AllowOrderWith", func(t *testing.T) {
		params := &PaginationParams{
			OrderBy: []string{"name:asc"},
		}
		params.Normalize(
			AllowOrderWith(map[string]string{
				"name": "u.name",
			}),
			AllowOrderWithSQLC([]string{"created_at_asc"}),
		)

		assert.Len(t, params.orderEntries, 1)
		assert.Equal(t, "created_at", params.orderEntries[0].column)
		assert.Equal(t, "ASC", params.orderEntries[0].seq)
	})
}

func TestPagination_AllowOrderWith(t *testing.T) {
	t.Run("valid entries with column mapping", func(t *testing.T) {
		params := &PaginationParams{
			OrderBy: []string{"name:asc", "created_at:desc"},
		}
		params.Normalize(AllowOrderWith(map[string]string{
			"name":       "u.name",
			"created_at": "u.created_at",
		}))

		assert.Len(t, params.orderEntries, 2)
		assert.Equal(t, "u.name", params.orderEntries[0].column)
		assert.Equal(t, "ASC", params.orderEntries[0].seq)
		assert.Equal(t, "u.created_at", params.orderEntries[1].column)
		assert.Equal(t, "DESC", params.orderEntries[1].seq)
	})

	t.Run("default order applied", func(t *testing.T) {
		params := &PaginationParams{}
		params.Normalize(
			AllowOrderWith(map[string]string{"created_at": "u.created_at"}),
			WithDefaultOrderBy("created_at:desc"),
		)

		assert.Len(t, params.orderEntries, 1)
		assert.Equal(t, "DESC", params.orderEntries[0].seq)
	})

	t.Run("invalid order entries ignored", func(t *testing.T) {
		params := &PaginationParams{
			OrderBy: []string{"invalid_field:asc", "name:asc"},
		}
		params.Normalize(AllowOrderWith(map[string]string{
			"name": "u.name",
		}))

		assert.Len(t, params.orderEntries, 1)
		assert.Equal(t, "u.name", params.orderEntries[0].column)
	})

	t.Run("duplicate fields - first wins", func(t *testing.T) {
		params := &PaginationParams{
			OrderBy: []string{"name:asc", "name:desc"},
		}
		params.Normalize(AllowOrderWith(map[string]string{
			"name": "u.name",
		}))

		assert.Len(t, params.orderEntries, 1)
		assert.Equal(t, "ASC", params.orderEntries[0].seq)
	})
}

func TestPagination_Normalize(t *testing.T) {
	t.Run("default limit applied", func(t *testing.T) {
		params := &PaginationParams{}
		params.Normalize()

		assert.Equal(t, int32(1), params.Page)
		assert.Equal(t, int32(20), params.Limit)
	})

	t.Run("custom default limit", func(t *testing.T) {
		params := &PaginationParams{}
		params.Normalize(WithDefaultLimit(50))

		assert.Equal(t, int32(50), params.Limit)
	})

	t.Run("max limit enforced", func(t *testing.T) {
		params := &PaginationParams{Limit: 200}
		params.Normalize(WithMaxLimit(100))

		assert.Equal(t, int32(100), params.Limit)
	})

	t.Run("negative page defaults to 1", func(t *testing.T) {
		params := &PaginationParams{Page: -1}
		params.Normalize()

		assert.Equal(t, int32(1), params.Page)
	})

	t.Run("zero page defaults to 1", func(t *testing.T) {
		params := &PaginationParams{Page: 0}
		params.Normalize()

		assert.Equal(t, int32(1), params.Page)
	})
}

func TestPagination_ToSQL(t *testing.T) {
	t.Run("generates ORDER BY clause", func(t *testing.T) {
		params := &PaginationParams{}
		params.Normalize(AllowOrderWithSQLC([]string{
			"created_at_desc",
			"name_asc",
		}))

		clause := params.ToSQL()
		assert.Equal(t, "created_at DESC, name ASC", clause)
	})

	t.Run("empty when no entries", func(t *testing.T) {
		params := &PaginationParams{}
		params.Normalize()

		assert.Empty(t, params.ToSQL())
	})
}

func TestPagination_GetOrderBy(t *testing.T) {
	params := &PaginationParams{}
	params.Normalize(AllowOrderWithSQLC([]string{"created_at_asc"}))

	orderBy := params.GetOrderBy()
	assert.Equal(t, "ASC", orderBy["created_at"])
}

func TestPagination_GetOrderByEntry(t *testing.T) {
	t.Run("returns first entry", func(t *testing.T) {
		params := &PaginationParams{}
		params.Normalize(AllowOrderWithSQLC([]string{
			"created_at_desc",
			"name_asc",
		}))

		entry := params.GetOrderByEntry()
		assert.Equal(t, "created_at_desc", entry)
	})

	t.Run("empty when no entries", func(t *testing.T) {
		params := &PaginationParams{}
		params.Normalize()

		assert.Empty(t, params.GetOrderByEntry())
	})
}

func TestPagination_GetOffset(t *testing.T) {
	t.Run("page 1 returns 0", func(t *testing.T) {
		params := &PaginationParams{Page: 1, Limit: 20}
		params.Normalize()

		assert.Equal(t, int32(0), params.GetOffset())
	})

	t.Run("page 2 returns offset", func(t *testing.T) {
		params := &PaginationParams{Page: 2, Limit: 20}
		params.Normalize()

		assert.Equal(t, int32(20), params.GetOffset())
	})

	t.Run("page 3 with limit 10", func(t *testing.T) {
		params := &PaginationParams{Page: 3, Limit: 10}
		params.Normalize()

		assert.Equal(t, int32(20), params.GetOffset())
	})
}

func TestPagination_GetLimit(t *testing.T) {
	params := &PaginationParams{Limit: 50}
	params.Normalize()

	assert.Equal(t, int32(50), params.GetLimit())
}

func TestNewPaginationMeta(t *testing.T) {
	t.Run("calculates total pages", func(t *testing.T) {
		meta := NewPaginationMeta(2, 20, 50)

		assert.Equal(t, int32(2), meta.Page)
		assert.Equal(t, int32(20), meta.Limit)
		assert.Equal(t, int64(50), meta.TotalCount)
		assert.Equal(t, int64(3), meta.TotalPages)
		assert.True(t, meta.HasNext)
		assert.True(t, meta.HasPrev)
	})

	t.Run("no next on last page", func(t *testing.T) {
		meta := NewPaginationMeta(3, 20, 50)

		assert.False(t, meta.HasNext)
		assert.True(t, meta.HasPrev)
	})

	t.Run("no prev on first page", func(t *testing.T) {
		meta := NewPaginationMeta(1, 20, 50)

		assert.True(t, meta.HasNext)
		assert.False(t, meta.HasPrev)
	})

	t.Run("zero count defaults to 1 page", func(t *testing.T) {
		meta := NewPaginationMeta(1, 20, 0)

		assert.Equal(t, int64(1), meta.TotalPages)
		assert.False(t, meta.HasNext)
		assert.False(t, meta.HasPrev)
	})

	t.Run("handles zero limit", func(t *testing.T) {
		meta := NewPaginationMeta(1, 0, 0)

		assert.Equal(t, int64(1), meta.TotalPages)
	})
}
