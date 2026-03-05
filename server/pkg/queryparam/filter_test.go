package queryparam

import (
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type UserStatus string

const (
	StatusActive  UserStatus = "active"
	StatusBanned  UserStatus = "banned"
	StatusPending UserStatus = "pending"
)

func (s *UserStatus) UnmarshalText(b []byte) error {
	switch UserStatus(b) {
	case StatusActive, StatusBanned, StatusPending:
		*s = UserStatus(b)
		return nil
	}
	return assert.AnError
}

func TestFilter_IsEmpty(t *testing.T) {
	t.Run("returns true when no conditions", func(t *testing.T) {
		f := Filter[string]{}
		assert.True(t, f.IsEmpty())
	})

	t.Run("returns false when has conditions", func(t *testing.T) {
		f := Filter[string]{
			Conditions: []Condition[string]{{Op: OpEq, Value: "test"}},
		}
		assert.False(t, f.IsEmpty())
	})
}

func TestFilter_Has(t *testing.T) {
	f := Filter[string]{
		Conditions: []Condition[string]{
			{Op: OpEq, Value: "test"},
			{Op: OpNe, Value: "other"},
		},
	}

	t.Run("returns true for existing operator", func(t *testing.T) {
		assert.True(t, f.Has(OpEq))
	})

	t.Run("returns false for non-existing operator", func(t *testing.T) {
		assert.False(t, f.Has(OpGt))
	})
}

func TestFilter_Values(t *testing.T) {
	f := Filter[string]{
		Conditions: []Condition[string]{
			{Op: OpEq, Value: "a"},
			{Op: OpNe, Value: "b"},
		},
	}

	values := f.Values()
	assert.Equal(t, []string{"a", "b"}, values)
}

func TestFilter_Decode_BareKey(t *testing.T) {
	t.Run("bare key eq", func(t *testing.T) {
		f := Filter[float64]{}
		values := url.Values{"price": {"10"}}

		err := f.decodeFilter(values, "price", nil)
		require.NoError(t, err)

		require.Len(t, f.Conditions, 1)
		assert.Equal(t, OpEq, f.Conditions[0].Op)
		assert.Equal(t, float64(10), f.Conditions[0].Value)
	})
}

func TestFilter_Decode_BracketOps(t *testing.T) {
	t.Run("gte", func(t *testing.T) {
		f := Filter[float64]{}
		values := url.Values{"price[gte]": {"10"}}

		err := f.decodeFilter(values, "price", nil)
		require.NoError(t, err)

		require.Len(t, f.Conditions, 1)
		assert.Equal(t, OpGte, f.Conditions[0].Op)
		assert.Equal(t, float64(10), f.Conditions[0].Value)
	})

	t.Run("lte", func(t *testing.T) {
		f := Filter[float64]{}
		values := url.Values{"price[lte]": {"500"}}

		err := f.decodeFilter(values, "price", nil)
		require.NoError(t, err)

		require.Len(t, f.Conditions, 1)
		assert.Equal(t, OpLte, f.Conditions[0].Op)
		assert.Equal(t, float64(500), f.Conditions[0].Value)
	})

	t.Run("gt", func(t *testing.T) {
		f := Filter[float64]{}
		values := url.Values{"price[gt]": {"100"}}

		err := f.decodeFilter(values, "price", nil)
		require.NoError(t, err)

		require.Len(t, f.Conditions, 1)
		assert.Equal(t, OpGt, f.Conditions[0].Op)
	})

	t.Run("lt", func(t *testing.T) {
		f := Filter[float64]{}
		values := url.Values{"price[lt]": {"50"}}

		err := f.decodeFilter(values, "price", nil)
		require.NoError(t, err)

		require.Len(t, f.Conditions, 1)
		assert.Equal(t, OpLt, f.Conditions[0].Op)
	})

	t.Run("ne", func(t *testing.T) {
		f := Filter[string]{}
		values := url.Values{"name[ne]": {"john"}}

		err := f.decodeFilter(values, "name", nil)
		require.NoError(t, err)

		require.Len(t, f.Conditions, 1)
		assert.Equal(t, OpNe, f.Conditions[0].Op)
		assert.Equal(t, "john", f.Conditions[0].Value)
	})
}

func TestFilter_Decode_MultipleOps(t *testing.T) {
	f := Filter[float64]{}
	values := url.Values{
		"price[gte]": {"10"},
		"price[lte]": {"500"},
	}

	err := f.decodeFilter(values, "price", nil)
	require.NoError(t, err)

	require.Len(t, f.Conditions, 2)
	assert.Equal(t, OpGte, f.Conditions[0].Op)
	assert.Equal(t, float64(10), f.Conditions[0].Value)
	assert.Equal(t, OpLte, f.Conditions[1].Op)
	assert.Equal(t, float64(500), f.Conditions[1].Value)
}

func TestFilter_Decode_Like(t *testing.T) {
	t.Run("like on string field", func(t *testing.T) {
		f := Filter[string]{}
		values := url.Values{"name[like]": {"widget"}}

		err := f.decodeFilter(values, "name", nil)
		require.NoError(t, err)

		require.Len(t, f.Conditions, 1)
		assert.Equal(t, OpLike, f.Conditions[0].Op)
		assert.Equal(t, "widget", f.Conditions[0].Value)
	})

	t.Run("ilike on string field", func(t *testing.T) {
		f := Filter[string]{}
		values := url.Values{"name[ilike]": {"widget"}}

		err := f.decodeFilter(values, "name", nil)
		require.NoError(t, err)

		require.Len(t, f.Conditions, 1)
		assert.Equal(t, OpILike, f.Conditions[0].Op)
	})

	t.Run("like on non-string field returns error", func(t *testing.T) {
		f := Filter[int]{}
		values := url.Values{"price[like]": {"10"}}

		err := f.decodeFilter(values, "price", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only valid for string fields")
	})
}

func TestFilter_Decode_In(t *testing.T) {
	t.Run("in single value", func(t *testing.T) {
		f := Filter[UserStatus]{}
		values := url.Values{"status[in]": {"active"}}

		err := f.decodeFilter(values, "status", nil)
		require.NoError(t, err)

		require.Len(t, f.Conditions, 1)
		assert.Equal(t, OpIn, f.Conditions[0].Op)
		assert.Equal(t, StatusActive, f.Conditions[0].Value)
	})

	t.Run("in comma separated", func(t *testing.T) {
		f := Filter[UserStatus]{}
		values := url.Values{"status[in]": {"active,banned"}}

		err := f.decodeFilter(values, "status", nil)
		require.NoError(t, err)

		require.Len(t, f.Conditions, 2)
		assert.Equal(t, StatusActive, f.Conditions[0].Value)
		assert.Equal(t, StatusBanned, f.Conditions[1].Value)
	})

	t.Run("in multiple params", func(t *testing.T) {
		f := Filter[UserStatus]{}
		values := url.Values{
			"status[in]": {"active", "banned"},
		}

		err := f.decodeFilter(values, "status", nil)
		require.NoError(t, err)

		require.Len(t, f.Conditions, 2)
	})
}

func TestFilter_Decode_AllowedOps(t *testing.T) {
	t.Run("restricted ops accepted", func(t *testing.T) {
		f := Filter[float64]{}
		values := url.Values{
			"price[gte]": {"10"},
			"price[lte]": {"500"},
		}

		err := f.decodeFilter(values, "price", []CompareOp{OpGte, OpLte})
		require.NoError(t, err)
		require.Len(t, f.Conditions, 2)
	})

	t.Run("restricted ops - disallowed returns error", func(t *testing.T) {
		f := Filter[float64]{}
		values := url.Values{"price[like]": {"10"}}

		err := f.decodeFilter(values, "price", []CompareOp{OpGte, OpLte})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "is not allowed")
	})

	t.Run("bare key restricted - eq not allowed", func(t *testing.T) {
		f := Filter[float64]{}
		values := url.Values{"price": {"10"}}

		err := f.decodeFilter(values, "price", []CompareOp{OpGte})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "is not allowed")
	})
}

func TestFilter_Decode_DuplicateOp(t *testing.T) {
	f := Filter[float64]{}
	values := url.Values{
		"price[gte]": {"10"},
	}

	err := f.decodeFilter(values, "price", nil)
	require.NoError(t, err)

	err = f.decodeFilter(values, "price", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate operator")
}

func TestFilter_Decode_InvalidType(t *testing.T) {
	f := Filter[int]{}
	values := url.Values{"price": {"abc"}}

	err := f.decodeFilter(values, "price", nil)
	assert.Error(t, err)
}

func TestFilter_Decode_Enum(t *testing.T) {
	t.Run("valid enum", func(t *testing.T) {
		f := Filter[UserStatus]{}
		values := url.Values{"status": {"active"}}

		err := f.decodeFilter(values, "status", nil)
		require.NoError(t, err)

		require.Len(t, f.Conditions, 1)
		assert.Equal(t, OpEq, f.Conditions[0].Op)
		assert.Equal(t, StatusActive, f.Conditions[0].Value)
	})

	t.Run("invalid enum returns error", func(t *testing.T) {
		f := Filter[UserStatus]{}
		values := url.Values{"status": {"invalid"}}

		err := f.decodeFilter(values, "status", nil)
		assert.Error(t, err)
	})
}

func TestFilter_Decode_UUID(t *testing.T) {
	testUUID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	f := Filter[uuid.UUID]{}
	values := url.Values{"id": {"550e8400-e29b-41d4-a716-446655440000"}}

	err := f.decodeFilter(values, "id", nil)
	require.NoError(t, err)

	require.Len(t, f.Conditions, 1)
	assert.Equal(t, testUUID, f.Conditions[0].Value)
}

func TestParseOpsTag(t *testing.T) {
	t.Run("valid ops", func(t *testing.T) {
		ops, err := parseOpsTag("gte,lte")
		require.NoError(t, err)
		assert.Equal(t, []CompareOp{OpGte, OpLte}, ops)
	})

	t.Run("empty string returns nil", func(t *testing.T) {
		ops, err := parseOpsTag("")
		require.NoError(t, err)
		assert.Nil(t, ops)
	})

	t.Run("unknown operator returns error", func(t *testing.T) {
		_, err := parseOpsTag("gte,invalid")
		assert.Error(t, err)
	})
}
