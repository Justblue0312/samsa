# Query Parameters Library

A Go library for decoding and building query parameters with built-in pagination, filtering, sorting, and enum validation.

## Quick Start

```go
import "yourmodule/internal/queryparam"

type ListUsersParams struct {
    queryparam.PaginationParams
    Name   *string    `query:"name"`
    Email  *string    `query:"email"`
    Active *bool      `query:"active"`
}

func ListUsers(w http.ResponseWriter, r *http.Request) {
    var params ListUsersParams
    if err := queryparam.DecodeRequest(&params, r.URL.RawQuery); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    params.Normalize(
        queryparam.WithDefaultLimit(20),
        queryparam.WithMaxLimit(100),
    )
    // params.Page, params.Limit, params.Offset() for pagination
}
```

---

## Table of Contents

1. [Pagination & Sorting](#pagination--sorting)
2. [Basic Filters](#basic-filters)
3. [Enum Support](#enum-support)
4. [Comparison Filters](#comparison-filters)
5. [Full Examples](#full-examples)

---

## Pagination & Sorting

The library provides `PaginationParams` with built-in support:

| Field | Type | Description |
|-------|------|-------------|
| `Page` | `int32` | Page number (1-based) |
| `Limit` | `int32` | Items per page |
| `OrderBy` | `[]string` | Sort specifications (e.g., `"name:asc"`) |

**Usage:**

```go
type Params struct {
    queryparam.PaginationParams
    // filters...
}

params.Normalize(
    queryparam.WithDefaultLimit(20),
    queryparam.WithMaxLimit(100),
)

// Access pagination values
page := params.Page           // 1
limit := params.Limit         // 20
offset := params.GetOffset() // (page-1) * limit
```

### Sorting Options

#### Option 1: AllowOrderWith (Raw SQL Mapping)

Sort syntax: `?order_by=name:asc&order_by=created_at:desc`

```go
params.Normalize(
    queryparam.WithDefaultLimit(20),
    queryparam.WithMaxLimit(100),
    queryparam.AllowOrderWith(map[string]string{
        "name":       "u.name",
        "created_at": "u.created_at",
    }),
)

orderBy := params.ToSQL()
// → "u.name ASC, u.created_at DESC"
```

#### Option 2: AllowOrderWithSQLC (SQLC Integration)

Use for sqlc-generated queries with predefined order entries:

```go
params.Normalize(
    queryparam.WithDefaultLimit(20),
    queryparam.WithMaxLimit(100),
    queryparam.AllowOrderWithSQLC([]string{
        "created_at_asc",
        "created_at_desc",
        "updated_at_asc",
        "updated_at_desc",
    }),
)

orderBy := params.ToSQL()
// → "created_at ASC" (first valid entry from OrderBy query param)
```

**Query format for AllowOrderWithSQLC:**
```
?order_by=created_at_asc
?order_by=updated_at_desc
```

**Methods:**

| Method | Description |
|--------|-------------|
| `GetOffset()` | Returns SQL OFFSET value |
| `GetLimit()` | Returns SQL LIMIT value |
| `ToSQL()` | Returns ORDER BY clause string |
| `GetOrderBy()` | Returns map[field]direction |
| `GetOrderByEntry()` | Returns first entry as `"field_seq"` (e.g., `"created_at_asc"`) |

**Pagination Metadata:**

```go
meta := queryparam.NewPaginationMeta(int(params.Page), int(params.Limit), totalCount)
// meta.Page, meta.Limit, meta.TotalCount, meta.TotalPages, meta.HasNext, meta.HasPrev
```

**Response example:**
```json
{
  "page": 2,
  "limit": 20,
  "total_count": 100,
  "total_pages": 5,
  "has_next": true,
  "has_prev": true
}
```

---

## Basic Filters

Define filters as **pointer fields** (nil = not provided):

```go
type ListUsersParams struct {
    queryparam.PaginationParams
    
    Name     *string    `query:"name"`     // ?name=john
    Email    *string    `query:"email"`    // ?email=john@example.com
    Active   *bool      `query:"active"`   // ?active=true
    MinAge   *int32     `query:"min_age"`  // ?min_age=18
    MaxAge   *int32     `query:"max_age"`  // ?max_age=65
}

// Access: nil check = not provided
if params.Name != nil {
    // filter by name
}
```

**Multi-value arrays:**

```go
Tags []string `query:"tag"` // ?tag=sale&tag=new OR ?tag=sale,new
```

---

## Enum Support

Create validated enums that automatically reject invalid values.

### String-based Enums

```go
type UserStatus string

const (
    UserStatusActive  UserStatus = "active"
    UserStatusBanned  UserStatus = "banned"
    UserStatusPending UserStatus = "pending"
)

func (s *UserStatus) UnmarshalText(b []byte) error {
    v := UserStatus(b)
    switch v {
    case UserStatusActive, UserStatusBanned, UserStatusPending:
        *s = v
        return nil
    }
    return fmt.Errorf("invalid user status %q", string(b))
}

func (s UserStatus) MarshalText() ([]byte, error) {
    return []byte(s), nil
}
```

### Int-based Enums

```go
type Priority int

const (
    PriorityLow    Priority = 1
    PriorityMedium Priority = 2
    PriorityHigh   Priority = 3
)

func (p *Priority) UnmarshalText(b []byte) error {
    var n int
    if _, err := fmt.Sscanf(string(b), "%d", &n); err != nil {
        return fmt.Errorf("invalid priority %q", string(b))
    }
    v := Priority(n)
    switch v {
    case PriorityLow, PriorityMedium, PriorityHigh:
        *p = v
        return nil
    }
    return fmt.Errorf("invalid priority %d", n)
}
```

### Using Enums in Params

```go
type ListUsersParams struct {
    queryparam.PaginationParams
    
    // Optional enum
    Status *queryparam.UserStatus `query:"status"`
    
    // Multi-value enum: ?role=admin&role=editor
    Roles []queryparam.UserStatus `query:"role"`
    
    // Works with UUIDs too
    IDs []uuid.UUID `query:"id"`
}
```

---

## Comparison Filters

Use `Filter[T]` for range queries with operator control:

```go
type ListProductsParams struct {
    queryparam.PaginationParams
    
    // Restricted operators via ops tag
    Price queryparam.Filter[float64] `query:"price" ops:"gte,lte,eq"`
    Stock queryparam.Filter[int]     `query:"stock" ops:"gt,gte,eq"`
    Name  queryparam.Filter[string]  `query:"name"  ops:"like,ilike"`
    
    // All operators allowed (no ops tag)
    Rating queryparam.Filter[float64] `query:"rating"`
    
    // Works with enums & UUIDs
    Status  queryparam.Filter[queryparam.UserStatus] `query:"status" ops:"eq,in"`
    OwnerID queryparam.Filter[uuid.UUID]             `query:"owner_id" ops:"eq"`
}
```

**Supported Operators:**

| Operator | Description | Valid Types |
|----------|-------------|-------------|
| `eq` | Equals | all |
| `in` | In set | all |
| `gt`, `gte` | Greater than | numbers |
| `lt`, `lte` | Less than | numbers |
| `like`, `ilike` | Pattern match | strings |

**Usage:**

```go
// Build WHERE clause
b := queryparam.NewFilterBuilder(queryparam.DialectPostgres)
params.Price.AppendTo("p.price", b)
params.Stock.AppendTo("p.stock", b)
params.Name.AppendTo("p.name", b)

where, args := b.Build()
// → "p.price >= $1 AND p.price <= $2 AND p.stock > $3"
```

**Query Syntax:**

```go
// Range
?price[gte]=10&price[lte]=500

// Shorthand eq (no brackets)
?stock=5

// Like (auto-wrapped in %)
?name[like]=widget
// → p.name LIKE '%widget%'

// IN query
?status[in]=active,banned
// → p.status IN ($1, $2)
```

---

## Full Examples

### Basic Listing Endpoint

```go
package handler

import (
    "encoding/json"
    "net/http"

    "github.com/go-chi/chi/v5"
    "yourmodule/internal/queryparam"
)

type ListUsersParams struct {
    queryparam.PaginationParams

    Name     *string    `query:"name"`
    Email    *string    `query:"email"`
    Active   *bool      `query:"active"`
    RoleID   *uuid.UUID `query:"role_id"`
    MinAge   *int32     `query:"min_age"`
    MaxAge   *int32     `query:"max_age"`
}

func ListUsers(w http.ResponseWriter, r *http.Request) {
    var params ListUsersParams

    if err := queryparam.DecodeRequest(&params, r.URL.RawQuery); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    params.Normalize(
        queryparam.WithDefaultLimit(20),
        queryparam.WithMaxLimit(100),
        queryparam.AllowOrderWith(map[string]string{
            "name":       "u.name",
            "email":      "u.email",
            "created_at": "u.created_at",
        }),
    )

    orderBy := params.ToSQL()

    json.NewEncoder(w).Encode(map[string]any{
        "page":     params.Page,
        "limit":    params.Limit,
        "offset":   params.Offset(),
        "order_by": orderBy,
    })
}

func Routes() *chi.Mux {
    r := chi.NewRouter()
    r.Get("/users", ListUsers)
    return r
}
```

### With Comparison Filters

```go
package handler

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/google/uuid"
    "yourmodule/internal/queryparam"
)

type ListProductsParams struct {
    queryparam.PaginationParams

    Price    queryparam.Filter[float64] `query:"price"      ops:"gte,lte,eq"`
    Stock    queryparam.Filter[int]     `query:"stock"      ops:"gt,gte,eq"`
    Name     queryparam.Filter[string]  `query:"name"       ops:"like,ilike"`
    Category queryparam.Filter[string]  `query:"category"   ops:"eq,in"`
    Rating   queryparam.Filter[float64]  `query:"rating"` // all ops allowed
}

func ListProducts(w http.ResponseWriter, r *http.Request) {
    var params ListProductsParams

    if err := queryparam.DecodeRequest(&params, r.URL.RawQuery); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    params.Normalize(
        queryparam.WithMaxLimit(100),
        queryparam.AllowOrderWith(map[string]string{
            "price":      "p.price",
            "created_at": "p.created_at",
        }),
    )

    b := queryparam.NewFilterBuilder(queryparam.DialectPostgres)
    params.Price.AppendTo("p.price", b)
    params.Stock.AppendTo("p.stock", b)
    params.Name.AppendTo("p.name", b)
    params.Category.AppendTo("p.category", b)
    params.Rating.AppendTo("p.rating", b)

    where, args := b.Build()

    orderBy := params.ToSQL()

    query := "SELECT * FROM products p"
    if where != "" {
        query += " WHERE " + where
    }
    if orderBy != "" {
        query += " ORDER BY " + orderBy
    }
    query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
    args = append(args, params.GetLimit(), params.GetOffset())

    rows, err := db.QueryContext(r.Context(), query, args...)
    // ... handle rows
}
```

---

## Common Query Examples

```
# Pagination
GET /users?page=2&limit=25

# Sorting with AllowOrderWith (colon syntax)
GET /users?order_by=name:asc&order_by=created_at:desc

# Sorting with AllowOrderWithSQLC (underscore syntax)
GET /users?order_by=created_at_asc
GET /users?order_by=updated_at_desc

# Basic filters
GET /users?name=john&active=true

# Range filters
GET /products?price[gte]=10&price[lte]=500

# Like query
GET /products?name[like]=widget

# IN query
GET /products?status[in]=active,banned
```

---

## Error Examples

The decoder returns descriptive 400 errors:

```
GET /users?status=superadmin
→ field "Status" (param "status"): invalid user status "superadmin"

GET /users?id=not-a-uuid
→ field "IDs" (param "id"): element 0: invalid UUID "not-a-uuid"

GET /products?price[lt]=50
→ operator "lt" is not allowed; permitted: gte, lte, eq

GET /products?status[eq]=nonexistent
→ [eq] invalid user status "nonexistent"
```