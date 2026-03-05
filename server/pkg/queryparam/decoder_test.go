package queryparam

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestStruct struct {
	Name   string    `query:"name"`
	Age    int       `query:"age"`
	Active bool      `query:"active"`
	Price  float64   `query:"price"`
	ID     uuid.UUID `query:"id"`
}

func TestDecode_SimpleTypes(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		params := &TestStruct{}
		values := url.Values{"name": {"john"}}

		err := Decode(params, values)
		require.NoError(t, err)
		assert.Equal(t, "john", params.Name)
	})

	t.Run("int", func(t *testing.T) {
		params := &TestStruct{}
		values := url.Values{"age": {"25"}}

		err := Decode(params, values)
		require.NoError(t, err)
		assert.Equal(t, 25, params.Age)
	})

	t.Run("bool true", func(t *testing.T) {
		params := &TestStruct{}
		values := url.Values{"active": {"true"}}

		err := Decode(params, values)
		require.NoError(t, err)
		assert.True(t, params.Active)
	})

	t.Run("bool false", func(t *testing.T) {
		params := &TestStruct{}
		values := url.Values{"active": {"false"}}

		err := Decode(params, values)
		require.NoError(t, err)
		assert.False(t, params.Active)
	})

	t.Run("float", func(t *testing.T) {
		params := &TestStruct{}
		values := url.Values{"price": {"19.99"}}

		err := Decode(params, values)
		require.NoError(t, err)
		assert.Equal(t, 19.99, params.Price)
	})

	t.Run("uuid", func(t *testing.T) {
		testUUID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		params := &TestStruct{}
		values := url.Values{"id": {"550e8400-e29b-41d4-a716-446655440000"}}

		err := Decode(params, values)
		require.NoError(t, err)
		assert.Equal(t, testUUID, params.ID)
	})
}

func TestDecode_MultipleValues(t *testing.T) {
	t.Run("multiple values for same key", func(t *testing.T) {
		type Params struct {
			Names []string `query:"name"`
		}
		params := &Params{}
		values := url.Values{
			"name": []string{"john", "jane"},
		}

		err := Decode(params, values)
		require.NoError(t, err)
		assert.Equal(t, []string{"john", "jane"}, params.Names)
	})

	t.Run("comma separated", func(t *testing.T) {
		type Params struct {
			Names []string `query:"name"`
		}
		params := &Params{}
		values := url.Values{
			"name": {"john,jane"},
		}

		err := Decode(params, values)
		require.NoError(t, err)
		assert.Equal(t, []string{"john", "jane"}, params.Names)
	})
}

func TestDecode_SliceTypes(t *testing.T) {
	t.Run("slice int", func(t *testing.T) {
		type Params struct {
			IDs []int `query:"id"`
		}
		params := &Params{}
		values := url.Values{"id": {"1,2,3"}}

		err := Decode(params, values)
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, params.IDs)
	})

	t.Run("slice uuid", func(t *testing.T) {
		type Params struct {
			IDs []uuid.UUID `query:"id"`
		}
		params := &Params{}
		values := url.Values{
			"id": {
				"550e8400-e29b-41d4-a716-446655440000," +
					"550e8400-e29b-41d4-a716-446655440001",
			},
		}

		err := Decode(params, values)
		require.NoError(t, err)
		require.Len(t, params.IDs, 2)
		assert.Equal(t,
			uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			params.IDs[0],
		)
		assert.Equal(t,
			uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
			params.IDs[1],
		)
	})

	t.Run("slice enum", func(t *testing.T) {
		type Params struct {
			Statuses []UserStatus `query:"status"`
		}
		params := &Params{}
		values := url.Values{"status": {"active,banned"}}

		err := Decode(params, values)
		require.NoError(t, err)
		assert.Equal(t, []UserStatus{StatusActive, StatusBanned}, params.Statuses)
	})
}

func TestDecode_Pointer(t *testing.T) {
	t.Run("pointer to string", func(t *testing.T) {
		type Params struct {
			Name *string `query:"name"`
		}
		params := &Params{}
		values := url.Values{"name": {"john"}}

		err := Decode(params, values)
		require.NoError(t, err)
		require.NotNil(t, params.Name)
		assert.Equal(t, "john", *params.Name)
	})

	t.Run("pointer to int", func(t *testing.T) {
		type Params struct {
			Age *int `query:"age"`
		}
		params := &Params{}
		values := url.Values{"age": {"25"}}

		err := Decode(params, values)
		require.NoError(t, err)
		require.NotNil(t, params.Age)
		assert.Equal(t, 25, *params.Age)
	})

	t.Run("pointer to enum", func(t *testing.T) {
		type Params struct {
			Status *UserStatus `query:"status"`
		}
		params := &Params{}
		values := url.Values{"status": {"active"}}

		err := Decode(params, values)
		require.NoError(t, err)
		require.NotNil(t, params.Status)
		assert.Equal(t, StatusActive, *params.Status)
	})
}

func TestDecode_Enum(t *testing.T) {
	t.Run("valid enum", func(t *testing.T) {
		type Params struct {
			Status UserStatus `query:"status"`
		}
		params := &Params{}
		values := url.Values{"status": {"active"}}

		err := Decode(params, values)
		require.NoError(t, err)
		assert.Equal(t, StatusActive, params.Status)
	})

	t.Run("invalid enum returns error", func(t *testing.T) {
		type Params struct {
			Status UserStatus `query:"status"`
		}
		params := &Params{}
		values := url.Values{"status": {"invalid"}}

		err := Decode(params, values)
		assert.Error(t, err)
	})
}

func TestDecode_StructTags(t *testing.T) {
	t.Run("omitempty - present", func(t *testing.T) {
		type Params struct {
			Name string `query:"name,omitempty"`
		}
		params := &Params{}
		values := url.Values{"name": {"john"}}

		err := Decode(params, values)
		require.NoError(t, err)
		assert.Equal(t, "john", params.Name)
	})

	t.Run("skip tag", func(t *testing.T) {
		type Params struct {
			Name string `query:"-"`
		}
		params := &Params{Name: "original"}
		values := url.Values{"name": {"john"}}

		err := Decode(params, values)
		require.NoError(t, err)
		assert.Equal(t, "original", params.Name)
	})

	t.Run("no tag", func(t *testing.T) {
		type Params struct {
			Name string
		}
		params := &Params{}
		values := url.Values{"Name": {"john"}}

		err := Decode(params, values)
		require.NoError(t, err)
		assert.Empty(t, params.Name)
	})
}

func TestDecode_EmbeddedStruct(t *testing.T) {
	type Params struct {
		PaginationParams
		Name string `query:"name"`
	}

	t.Run("embedded struct decoded", func(t *testing.T) {
		params := &Params{}
		values := url.Values{
			"page":     {"2"},
			"limit":    {"50"},
			"order_by": {"created_at:desc"},
			"name":     {"john"},
		}

		err := Decode(params, values)
		require.NoError(t, err)
		assert.Equal(t, int32(2), params.Page)
		assert.Equal(t, int32(50), params.Limit)
		assert.Equal(t, []string{"created_at:desc"}, params.OrderBy)
		assert.Equal(t, "john", params.Name)
	})

	t.Run("embedded struct with options", func(t *testing.T) {
		params := &Params{}
		values := url.Values{
			"page": {"1"},
			"name": {"john"},
		}
		params.Normalize(WithDefaultLimit(20))

		err := Decode(params, values)
		require.NoError(t, err)
		assert.Equal(t, int32(20), params.Limit)
	})
}

func TestDecode_Filter(t *testing.T) {
	type Params struct {
		PaginationParams
		Price  Filter[float64]    `query:"price" ops:"gte,lte"`
		Name   Filter[string]     `query:"name" ops:"like"`
		Status Filter[UserStatus] `query:"status" ops:"eq,in"`
	}

	t.Run("filter with allowed ops", func(t *testing.T) {
		params := &Params{}
		values := url.Values{
			"price[gte]": {"10"},
			"price[lte]": {"100"},
			"name[like]": {"john"},
			"status[in]": {"active,banned"},
		}

		err := Decode(params, values)
		require.NoError(t, err)

		require.Len(t, params.Price.Conditions, 2)
		assert.Equal(t, OpGte, params.Price.Conditions[0].Op)
		assert.Equal(t, OpLte, params.Price.Conditions[1].Op)

		require.Len(t, params.Name.Conditions, 1)
		assert.Equal(t, OpLike, params.Name.Conditions[0].Op)

		require.Len(t, params.Status.Conditions, 2)
		assert.Equal(t, StatusActive, params.Status.Conditions[0].Value)
		assert.Equal(t, StatusBanned, params.Status.Conditions[1].Value)
	})

	t.Run("filter disallowed ops returns error", func(t *testing.T) {
		type Params struct {
			Price Filter[float64] `query:"price" ops:"gte"`
		}
		params := &Params{}
		values := url.Values{
			"price[lte]": {"100"},
		}

		err := Decode(params, values)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "is not allowed")
	})
}

func TestDecode_Errors(t *testing.T) {
	t.Run("non-pointer", func(t *testing.T) {
		params := TestStruct{}
		err := Decode(params, url.Values{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be a non-nil pointer")
	})

	t.Run("nil pointer", func(t *testing.T) {
		var params *TestStruct
		err := Decode(params, url.Values{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be a non-nil pointer")
	})

	t.Run("invalid int", func(t *testing.T) {
		params := &TestStruct{}
		values := url.Values{"age": {"abc"}}

		err := Decode(params, values)
		assert.Error(t, err)
	})

	t.Run("invalid uuid", func(t *testing.T) {
		params := &TestStruct{}
		values := url.Values{"id": {"invalid-uuid"}}

		err := Decode(params, values)
		assert.Error(t, err)
	})
}

func TestDecodeRequest(t *testing.T) {
	t.Run("parses query string", func(t *testing.T) {
		params := &TestStruct{}
		req := &http.Request{
			URL: &url.URL{
				RawQuery: "name=john&age=25",
			},
		}

		err := DecodeRequest(params, req.URL.RawQuery)
		require.NoError(t, err)
		assert.Equal(t, "john", params.Name)
		assert.Equal(t, 25, params.Age)
	})

	t.Run("invalid query string", func(t *testing.T) {
		params := &TestStruct{}
		err := DecodeRequest(params, "name=%")
		assert.Error(t, err)
	})
}
