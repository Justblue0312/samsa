# Advanced sqlc Patterns, Model Reuse & Utility Packages

## Table of Contents

1. [Dynamic Filters](#1-dynamic-filters)
2. [Multiple Sort Orders](#2-multiple-sort-orders)
3. [CTEs](#3-ctes)
4. [UNION Queries](#4-union-queries)
5. [When sqlc Reaches Its Limit — squirrel](#5-when-sqlc-reaches-its-limit--squirrel)
6. [Reuse or Redefine sqlc Models](#6-reuse-or-redefine-sqlc-models)
7. [Utility Package Organization](#7-utility-package-organization)
8. [Rules Cheatsheet](#8-rules-cheatsheet)

---

## 1. Dynamic Filters

sqlc generates static SQL. Dynamic filters — where the caller decides which WHERE clauses to apply at runtime — require a deliberate strategy. There are three levels of complexity.

### Level 1 — `sqlc.narg` for Optional Columns

Use this when the filter set is small and known at compile time. Each optional parameter uses `sqlc.narg` which generates a `*T` in the params struct — `nil` means "skip this filter".

```sql
-- db/queries/user.sql
-- name: ListUsers :many
SELECT id, name, email, role, created_at
FROM users
WHERE deleted_at IS NULL
  AND (sqlc.narg('role')::text   IS NULL OR role  = sqlc.narg('role')::text)
  AND (sqlc.narg('name')::text   IS NULL OR name  ILIKE '%' || sqlc.narg('name')::text || '%')
  AND (sqlc.narg('after')::timestamptz IS NULL OR created_at > sqlc.narg('after')::timestamptz)
ORDER BY created_at DESC
LIMIT  sqlc.arg('limit')
OFFSET sqlc.arg('offset');
```

Generated params:

```go
type ListUsersParams struct {
    Role   *string
    Name   *string
    After  *time.Time
    Limit  int32
    Offset int32
}
```

Repository usage:

```go
// feature/user/repository_impl.go

func (r *repository) List(ctx context.Context, f Filter) ([]*User, error) {
    p := sqlc.ListUsersParams{
        Limit:  int32(f.Limit),
        Offset: int32(f.Offset),
    }
    if f.Role != "" {
        s := string(f.Role)
        p.Role = &s
    }
    if f.Name != "" {
        p.Name = &f.Name
    }
    if !f.After.IsZero() {
        p.After = &f.After
    }

    rows, err := r.q.ListUsers(ctx, p)
    if err != nil {
        return nil, fmt.Errorf("repository.List: %w", err)
    }
    return toUsers(rows), nil
}
```

**Limit:** Works well up to ~5 optional filters. Beyond that, the SQL becomes hard to read and the generated struct has too many nullable fields.

---

### Level 2 — Filter Struct With Validation in the Domain

Define the filter as a domain type — not in the repository, not as raw strings. The usecase constructs it and validates it. The repository converts it to sqlc params.

```go
// feature/user/model.go

// Filter is the domain-level filter type.
// All fields are optional — zero value means "no filter".
type Filter struct {
    Role      Role
    Name      string    // partial match
    After     time.Time // created after
    Limit     int       // default applied in usecase if zero
    Offset    int
}

// SortField is an enum of allowed sort columns — prevents SQL injection.
type SortField string

const (
    SortByCreatedAt SortField = "created_at"
    SortByName      SortField = "name"
    SortByRole      SortField = "role"
)

type SortOrder string

const (
    SortAsc  SortOrder = "ASC"
    SortDesc SortOrder = "DESC"
)

type SortClause struct {
    Field SortField
    Order SortOrder
}

// ListOpts combines filter + sort + pagination.
type ListOpts struct {
    Filter Filter
    Sort   []SortClause // ordered: first element = primary sort
}

// Validate ensures the opts are valid before hitting the DB.
func (o ListOpts) Validate() error {
    for _, s := range o.Sort {
        switch s.Field {
        case SortByCreatedAt, SortByName, SortByRole:
        default:
            return &ValidationError{
                Field:   "sort",
                Message: "invalid sort field: " + string(s.Field),
            }
        }
    }
    if o.Filter.Limit < 0 {
        return &ValidationError{Field: "limit", Message: "must be >= 0"}
    }
    return nil
}
```

---

## 2. Multiple Sort Orders

sqlc cannot generate dynamic ORDER BY — the column and direction are part of the SQL string, not parameters. You have two options depending on complexity.

### Option A — One Query Per Sort Combination (for ≤3 sorts)

```sql
-- name: ListUsersByCreatedAt :many
SELECT id, name, email, role, created_at FROM users
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListUsersByName :many
SELECT id, name, email, role, created_at FROM users
WHERE deleted_at IS NULL
ORDER BY name ASC
LIMIT $1 OFFSET $2;
```

Repository dispatches based on the sort field:

```go
func (r *repository) List(ctx context.Context, opts ListOpts) ([]*User, error) {
    primary := SortByCreatedAt
    if len(opts.Sort) > 0 {
        primary = opts.Sort[0].Field
    }

    switch primary {
    case SortByName:
        rows, err := r.q.ListUsersByName(ctx, sqlc.ListUsersByNameParams{
            Limit: int32(opts.Filter.Limit), Offset: int32(opts.Filter.Offset),
        })
        if err != nil {
            return nil, fmt.Errorf("repository.List by name: %w", err)
        }
        return toUsers(rows), nil
    default:
        rows, err := r.q.ListUsersByCreatedAt(ctx, sqlc.ListUsersByCreatedAtParams{
            Limit: int32(opts.Filter.Limit), Offset: int32(opts.Filter.Offset),
        })
        if err != nil {
            return nil, fmt.Errorf("repository.List by created_at: %w", err)
        }
        return toUsers(rows), nil
    }
}
```

### Option B — Dynamic Builder for Complex Sorts (see Section 5)

When you need arbitrary multi-column sorts, sqlc is the wrong tool. Use `squirrel` instead.

---

## 3. CTEs

CTEs are first-class citizens in sqlc. sqlc parses them correctly and generates clean params. Use CTEs for:

- Breaking complex joins into readable steps
- Recursive hierarchy queries (org charts, nested categories)
- Aggregation + detail in one query

### Simple CTE — User With Project Count

```sql
-- name: GetUserWithStats :one
WITH project_counts AS (
    SELECT owner_id, COUNT(*) AS total
    FROM projects
    WHERE deleted_at IS NULL
    GROUP BY owner_id
)
SELECT
    u.id,
    u.name,
    u.email,
    u.role,
    u.created_at,
    COALESCE(pc.total, 0) AS project_count
FROM users u
LEFT JOIN project_counts pc ON pc.owner_id = u.id
WHERE u.id = $1
  AND u.deleted_at IS NULL;
```

sqlc generates a dedicated row type for this:

```go
// generated: gen/sqlc/user.sql.go
type GetUserWithStatsRow struct {
    ID           string
    Name         string
    Email        string
    Role         string
    CreatedAt    time.Time
    ProjectCount int64
}
```

Map to a richer domain type in the repository:

```go
// feature/user/model.go
type UserWithStats struct {
    User
    ProjectCount int64
}
```

```go
// feature/user/repository_impl.go
func (r *repository) FindWithStats(ctx context.Context, id string) (*UserWithStats, error) {
    row, err := r.q.GetUserWithStats(ctx, id)
    if err != nil {
        return nil, mapErr(err, id)
    }
    return &UserWithStats{
        User:         *toUser(toUserRow(row)), // reuse base mapper
        ProjectCount: row.ProjectCount,
    }, nil
}

// toUserRow bridges GetUserWithStatsRow → GetUserRow fields shared between them
func toUserRow(r sqlc.GetUserWithStatsRow) sqlc.GetUserRow {
    return sqlc.GetUserRow{
        ID: r.ID, Name: r.Name, Email: r.Email,
        Role: r.Role, CreatedAt: r.CreatedAt,
    }
}
```

### Recursive CTE — Organisation Hierarchy

```sql
-- name: GetTeamHierarchy :many
WITH RECURSIVE team_tree AS (
    -- base case: the root team
    SELECT id, name, parent_id, 0 AS depth
    FROM teams
    WHERE id = $1

    UNION ALL

    -- recursive: children of the current level
    SELECT t.id, t.name, t.parent_id, tt.depth + 1
    FROM teams t
    INNER JOIN team_tree tt ON tt.id = t.parent_id
    WHERE t.deleted_at IS NULL
)
SELECT id, name, parent_id, depth
FROM team_tree
ORDER BY depth, name;
```

```go
// feature/team/model.go
type TeamNode struct {
    ID       string
    Name     string
    ParentID *string
    Depth    int
    Children []*TeamNode // assembled in repository, not from DB
}
```

```go
// feature/team/repository_impl.go
func (r *repository) GetHierarchy(ctx context.Context, rootID string) (*TeamNode, error) {
    rows, err := r.q.GetTeamHierarchy(ctx, rootID)
    if err != nil {
        return nil, fmt.Errorf("repository.GetHierarchy: %w", err)
    }
    return buildTree(rows), nil
}

// buildTree assembles flat rows into a nested tree — pure Go, no SQL
func buildTree(rows []sqlc.GetTeamHierarchyRow) *TeamNode {
    nodes := make(map[string]*TeamNode, len(rows))
    var root *TeamNode

    for _, r := range rows {
        node := &TeamNode{
            ID:       r.ID,
            Name:     r.Name,
            ParentID: r.ParentID,
            Depth:    int(r.Depth),
        }
        nodes[r.ID] = node
        if r.Depth == 0 {
            root = node
        }
    }

    for _, r := range rows {
        if r.ParentID != nil {
            if parent, ok := nodes[*r.ParentID]; ok {
                parent.Children = append(parent.Children, nodes[r.ID])
            }
        }
    }

    return root
}
```

---

## 4. UNION Queries

sqlc handles UNION but all SELECT branches must return the same column types. Use UNION for:

- Activity feeds (combine events from multiple tables)
- Search across multiple entity types
- Combining soft-deleted + active records with a status discriminator

### Activity Feed — UNION Across Tables

```sql
-- name: GetActivityFeed :many
SELECT
    'project_created' AS event_type,
    p.id              AS entity_id,
    p.name            AS entity_name,
    p.owner_id        AS actor_id,
    p.created_at      AS occurred_at
FROM projects p
WHERE p.owner_id = $1

UNION ALL

SELECT
    'member_added'    AS event_type,
    pm.project_id     AS entity_id,
    u.name            AS entity_name,
    pm.added_by       AS actor_id,
    pm.created_at     AS occurred_at
FROM project_members pm
JOIN users u ON u.id = pm.user_id
WHERE pm.added_by = $1

ORDER BY occurred_at DESC
LIMIT $2;
```

sqlc generates a unified row type:

```go
// generated
type GetActivityFeedRow struct {
    EventType   string
    EntityID    string
    EntityName  string
    ActorID     string
    OccurredAt  time.Time
}
```

Map to a domain event:

```go
// feature/activity/model.go
type Event struct {
    Type       EventType
    EntityID   string
    EntityName string
    ActorID    string
    OccurredAt time.Time
}

type EventType string

const (
    EventProjectCreated EventType = "project_created"
    EventMemberAdded    EventType = "member_added"
)
```

```go
// feature/activity/repository_impl.go
func (r *repository) GetFeed(ctx context.Context, userID string, limit int) ([]*Event, error) {
    rows, err := r.q.GetActivityFeed(ctx, sqlc.GetActivityFeedParams{
        OwnerID: userID,
        Limit:   int32(limit),
    })
    if err != nil {
        return nil, fmt.Errorf("repository.GetFeed: %w", err)
    }

    events := make([]*Event, len(rows))
    for i, row := range rows {
        events[i] = &Event{
            Type:       EventType(row.EventType),
            EntityID:   row.EntityID,
            EntityName: row.EntityName,
            ActorID:    row.ActorID,
            OccurredAt: row.OccurredAt,
        }
    }
    return events, nil
}
```

---

## 5. When sqlc Reaches Its Limit — squirrel

sqlc is the right tool for 90% of queries. These patterns require a query builder instead:

```
□ Dynamic ORDER BY with user-controlled column + direction
□ Filter sets where any combination of 10+ fields may be active
□ Conditional JOINs (add a JOIN only when a specific filter is set)
□ IN clauses with variable-length slices
□ Complex search with ranked relevance scoring
```

Use **`github.com/Masterminds/squirrel`** alongside sqlc — not instead of it.

```bash
go get github.com/Masterminds/squirrel
```

### Dynamic List With squirrel

```go
// feature/user/repository_impl.go

import sq "github.com/Masterminds/squirrel"

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

func (r *repository) ListDynamic(ctx context.Context, opts ListOpts) ([]*User, error) {
    q := psql.
        Select("id", "name", "email", "role", "created_at").
        From("users").
        Where(sq.Eq{"deleted_at": nil})

    // optional filters — add WHERE clauses only when set
    if opts.Filter.Role != "" {
        q = q.Where(sq.Eq{"role": string(opts.Filter.Role)})
    }
    if opts.Filter.Name != "" {
        q = q.Where(sq.ILike{"name": "%" + opts.Filter.Name + "%"})
    }
    if !opts.Filter.After.IsZero() {
        q = q.Where(sq.Gt{"created_at": opts.Filter.After})
    }

    // dynamic multi-column ORDER BY — validated by domain type
    for _, s := range opts.Sort {
        q = q.OrderBy(string(s.Field) + " " + string(s.Order))
    }
    if len(opts.Sort) == 0 {
        q = q.OrderBy("created_at DESC") // safe default
    }

    q = q.Limit(uint64(opts.Filter.Limit)).
        Offset(uint64(opts.Filter.Offset))

    sql, args, err := q.ToSql()
    if err != nil {
        return nil, fmt.Errorf("repository.ListDynamic build sql: %w", err)
    }

    rows, err := r.pool.Query(ctx, sql, args...)
    if err != nil {
        return nil, fmt.Errorf("repository.ListDynamic exec: %w", err)
    }
    defer rows.Close()

    var users []*User
    for rows.Next() {
        u := &User{}
        if err := rows.Scan(
            &u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt,
        ); err != nil {
            return nil, fmt.Errorf("repository.ListDynamic scan: %w", err)
        }
        users = append(users, u)
    }
    return users, rows.Err()
}
```

### The Decision Matrix

```
Static filter set, ≤5 optional columns  →  sqlc.narg()
Static query, complex logic (CTE/UNION) →  sqlc
Dynamic ORDER BY or 6+ optional filters →  squirrel
Full-text search with ranking            →  squirrel + tsvector
Variable-length IN clause               →  squirrel
```

Both tools live inside `repository_impl.go` — the boundary never changes from the usecase's perspective.

---

## 6. Reuse or Redefine sqlc Models

### The Decision Rule

```
If ALL of these are true  →  reuse the sqlc model directly in the repository
  ✓ All fields are standard Go types (string, int, bool, time.Time, *T)
  ✓ No DB-specific nullable wrappers (no pgtype.*, no sql.NullString)
  ✓ No columns that are DB implementation details (deleted_at, version, row_num)
  ✓ The struct is used only inside repository_impl.go, never returned from the interface
  ✓ The field names already match your domain terminology

If ANY of these are true  →  define your own domain model
  ✗ Fields use pgtype.UUID, pgtype.Timestamptz, sql.NullString, etc.
  ✗ Struct is returned from the Repository interface
  ✗ Struct has DB-only fields (deleted_at, row_num, internal FKs)
  ✗ Field name diverges from domain language (role_id vs Role)
  ✗ You want a richer type (Role type alias vs raw string)
```

### Example A — sqlc Model Clean Enough to Reuse Internally

With proper `sqlc.yaml` overrides (`uuid→string`, `timestamptz→time.Time`):

```go
// generated: gen/sqlc/models.go
// sqlc.yaml overrides applied — all standard Go types, clean field names
type User struct {
    ID        string
    Name      string
    Email     string
    Role      string    // still a raw string, not your Role type
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

Even here — **do not reuse** across the Repository boundary. The `Role` field is `string` not your domain `Role` type, which means usecase code would do `string(user.Role)` and `Role(user.Role)` conversions scattered everywhere.

The mapping is one function, written once:

```go
func toUser(r sqlc.User) *user.User {
    return &user.User{
        ID:        r.ID,
        Name:      r.Name,
        Email:     r.Email,
        Role:      user.Role(r.Role), // typed once here, clean everywhere else
        CreatedAt: r.CreatedAt,
        UpdatedAt: r.UpdatedAt,
    }
}
```

### Example B — sqlc Query Row Type Identical to Your Domain Model

Sometimes a query row struct is genuinely identical to your domain model field-by-field. This happens with simple projection queries.

```go
// generated
type GetTokenRow struct {
    Token     string
    UserID    string
    ExpiresAt time.Time
}

// domain — identical
type Token struct {
    Token     string
    UserID    string
    ExpiresAt time.Time
}
```

In this specific case you may use the sqlc type **within repository_impl.go** as an intermediate, but you still map it at the interface boundary:

```go
// still define the mapping — it costs one function and protects you when the schema evolves
func toToken(r sqlc.GetTokenRow) *auth.Token {
    return &auth.Token{
        Token:     r.Token,
        UserID:    r.UserID,
        ExpiresAt: r.ExpiresAt,
    }
}
```

If the DB schema later adds a `revoked_at` column, your domain model does not change. The mapping function absorbs it.

### The Universal Rule

> Always map at the repository interface boundary — even when the types look identical. The cost is one function. The benefit is that DB schema changes never propagate above the repository layer.

---

## 7. Utility Package Organization

### The Problem

Go projects accumulate helpers over time: type converters, pointer helpers, ID generators, token generators. The wrong approach creates either:

```
pkg/utils/utils.go  ← 2000-line grab-bag that imports everything
pkg/helper/helper.go ← same problem, different name
```

Or the opposite extreme — scattered one-file packages that are hard to find.

### The Right Structure

Split by **what the function operates on**, not by what feature uses it. Each sub-package has a single, narrow responsibility.

```
internal/pkg/
  conv/          ← type conversion: primitives, slices, maps
  ptr/           ← pointer ↔ value helpers
  id/            ← ID generation (UUID, ULID, NanoID)
  token/         ← secure token generation (not JWT — that's security/jwt)
  security/
    jwt/         ← JWT sign/verify
    pwd/         ← password hash/verify
  paginate/      ← pagination math (offset, cursor)
  timeutil/      ← time helpers beyond stdlib
  must/          ← panic-on-error helpers for init-time use only
```

### `pkg/ptr` — Pointer Helpers

The most universally needed package. sqlc optional params use `*string`, `*int`, etc. — you need these constantly.

```go
// pkg/ptr/ptr.go
package ptr

// To returns a pointer to a copy of v.
// Usage: ptr.To("hello") instead of func() *string { s := "hello"; return &s }()
func To[T any](v T) *T {
    return &v
}

// From dereferences p. Returns the zero value of T if p is nil.
func From[T any](p *T) T {
    if p == nil {
        var zero T
        return zero
    }
    return *p
}

// FromOr dereferences p. Returns fallback if p is nil.
func FromOr[T any](p *T, fallback T) T {
    if p == nil {
        return fallback
    }
    return *p
}

// NonNil returns the first non-nil pointer, or nil if all are nil.
func NonNil[T any](ptrs ...*T) *T {
    for _, p := range ptrs {
        if p != nil {
            return p
        }
    }
    return nil
}
```

Usage:

```go
// in repository_impl.go — sqlc optional param
params.Role = ptr.To(string(opts.Filter.Role))

// in handler — optional response field
resp.NickName = ptr.To(user.NickName)
```

### `pkg/conv` — Type Conversions

```go
// pkg/conv/conv.go
package conv

// StringSlice converts []T to []string using a formatter func.
func StringSlice[T any](in []T, fn func(T) string) []string {
    out := make([]string, len(in))
    for i, v := range in {
        out[i] = fn(v)
    }
    return out
}

// SliceMap converts []T to []U element-by-element.
func SliceMap[T, U any](in []T, fn func(T) U) []U {
    out := make([]U, len(in))
    for i, v := range in {
        out[i] = fn(v)
    }
    return out
}

// Keys returns the keys of a map as a slice.
func Keys[K comparable, V any](m map[K]V) []K {
    keys := make([]K, 0, len(m))
    for k := range m {
        keys = append(keys, k)
    }
    return keys
}

// Filter returns elements of in for which keep returns true.
func Filter[T any](in []T, keep func(T) bool) []T {
    out := make([]T, 0, len(in))
    for _, v := range in {
        if keep(v) {
            out = append(out, v)
        }
    }
    return out
}
```

Usage in repository mapping:

```go
// map []sqlc.ListUsersRow → []*user.User in one line
users := conv.SliceMap(rows, func(r sqlc.ListUsersRow) *user.User {
    return toUser(r)
})
```

### `pkg/id` — ID Generation

```go
// pkg/id/id.go
package id

import (
    "crypto/rand"
    "encoding/hex"

    "github.com/google/uuid"
    "github.com/oklog/ulid/v2"
)

// UUID generates a new random UUID string (no dashes stripped).
func UUID() string {
    return uuid.New().String()
}

// ULID generates a new ULID — lexicographically sortable, time-prefixed.
// Use for IDs that benefit from sort-by-creation-order in the DB.
func ULID() string {
    return ulid.Make().String()
}

// Hex generates n random bytes as a lowercase hex string (length = 2*n).
// Use for opaque tokens, confirmation codes, etc.
func Hex(n int) string {
    b := make([]byte, n)
    _, _ = rand.Read(b)
    return hex.EncodeToString(b)
}

// Short generates a short alphanumeric ID of length n.
// Alphabet avoids visually ambiguous characters (0/O, 1/l/I).
func Short(n int) string {
    const alphabet = "23456789abcdefghjkmnpqrstuvwxyz"
    b := make([]byte, n)
    _, _ = rand.Read(b)
    for i, v := range b {
        b[i] = alphabet[v%byte(len(alphabet))]
    }
    return string(b)
}
```

Usage:

```go
// in usecase — never in repository or handler
newUser := &User{
    ID:        id.UUID(),
    CreatedAt: time.Now(),
}

// short invite code
invite.Code = id.Short(8) // e.g. "k3m9pq2r"
```

### `pkg/token` — Secure Token Generation

Distinct from `security/jwt`. This is for opaque random tokens (password reset, email confirm, API keys).

```go
// pkg/token/token.go
package token

import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "fmt"
)

// Generate returns a cryptographically random URL-safe token of byteLen random bytes.
// The returned string is base64url-encoded (length ≈ byteLen * 4/3).
func Generate(byteLen int) string {
    b := make([]byte, byteLen)
    _, _ = rand.Read(b)
    return base64.RawURLEncoding.EncodeToString(b)
}

// Hash returns the SHA-256 hash of a token as a hex string.
// Store the hash in the DB, never the raw token.
func Hash(raw string) string {
    sum := sha256.Sum256([]byte(raw))
    return fmt.Sprintf("%x", sum)
}
```

Usage in usecase:

```go
// feature/auth/usecase_impl.go
func (u *usecase) RequestPasswordReset(ctx context.Context, email string) error {
    raw  := token.Generate(32)      // send to user
    hash := token.Hash(raw)         // store in DB

    if err := u.repo.SaveResetToken(ctx, email, hash, time.Now().Add(1*time.Hour)); err != nil {
        return fmt.Errorf("usecase.RequestPasswordReset: %w", err)
    }

    _ = u.mailer.SendPasswordReset(ctx, email, raw) // raw goes to user
    return nil
}
```

### `pkg/paginate` — Pagination Math

```go
// pkg/paginate/paginate.go
package paginate

const (
    DefaultLimit = 20
    MaxLimit     = 100
)

type Page struct {
    Limit  int
    Offset int
}

type Meta struct {
    Total   int64
    Limit   int
    Offset  int
    HasNext bool
    HasPrev bool
}

// FromRequest sanitises caller-supplied limit/offset values.
func FromRequest(limit, offset int) Page {
    if limit <= 0 || limit > MaxLimit {
        limit = DefaultLimit
    }
    if offset < 0 {
        offset = 0
    }
    return Page{Limit: limit, Offset: offset}
}

// NewMeta builds the pagination metadata for a response.
func NewMeta(total int64, p Page) Meta {
    return Meta{
        Total:   total,
        Limit:   p.Limit,
        Offset:  p.Offset,
        HasNext: int64(p.Offset+p.Limit) < total,
        HasPrev: p.Offset > 0,
    }
}
```

Usage in handler:

```go
// feature/user/handler_http.go
func (h *HTTPHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
    page := paginate.FromRequest(
        queryInt(r, "limit"),
        queryInt(r, "offset"),
    )

    result, total, err := h.usecase.ListUsers(r.Context(), user.ListOpts{
        Filter: user.Filter{Limit: page.Limit, Offset: page.Offset},
    })
    if err != nil {
        respondDomainError(w, r, err)
        return
    }

    respond.OK(w, listUsersResponse{
        Users: toResponses(result),
        Meta:  paginate.NewMeta(total, page),
    })
}
```

### `pkg/must` — Init-Time Panic Helpers

For use **only** at program startup — never inside request handlers.

```go
// pkg/must/must.go
package must

// Do panics if err is non-nil. Use only in main() or init() — never in handlers.
func Do(err error) {
    if err != nil {
        panic(err)
    }
}

// Get returns v or panics if err is non-nil.
func Get[T any](v T, err error) T {
    if err != nil {
        panic(err)
    }
    return v
}
```

Usage:

```go
// bootstrap/config.go — startup only
cfg := must.Get(config.Load())

// compile-time template parsing — also startup only
tmpl := must.Get(template.ParseFS(embedFS, "templates/*.html"))
```

### Package Import Rules

```
pkg/ptr      → imports nothing internal
pkg/conv     → imports nothing internal
pkg/id       → imports only external libs (uuid, ulid)
pkg/token    → imports only stdlib (crypto/rand, crypto/sha256)
pkg/paginate → imports nothing internal
pkg/must     → imports nothing internal

feature/*    → may import any pkg/*
pkg/*        → must never import feature/* or infra/*
infra/*      → must never import pkg/* that import infra (no cycles)
```

---

## 8. Rules Cheatsheet

### sqlc Queries

```
□ sqlc.narg for optional filters up to ~5 columns — generates *T params
□ Always use RETURNING on INSERT/UPDATE — avoids read-after-write round-trip
□ Never SELECT * — list columns explicitly in every query
□ :one for single row, :many for slice, :exec for no-return, :execrows for affected count
□ CTE in sqlc works fully — use for complex joins and recursive hierarchies
□ UNION works in sqlc when all branches return identical column types
□ Dynamic ORDER BY or 6+ optional filters → use squirrel alongside sqlc
□ squirrel lives inside repository_impl.go — boundary never changes for usecase
□ Mirror feature structure in db/queries/ — one .sql file per feature
```

### sqlc Model Reuse

```
□ sqlc types are permitted inside repository_impl.go only
□ sqlc types never cross the Repository interface boundary
□ Always map sqlc row → domain model even when types look identical
□ schema evolves without affecting domain: mapping absorbs the change
□ sqlc.yaml overrides (uuid→string, timestamptz→time.Time) reduce mapping burden
□ toUser, toUsers, toProject etc. — one mapper per sqlc return type, reused for all methods
□ Role string → domain Role type alias: typed once in mapper, clean everywhere else
```

### Utility Packages

```
□ Split by what the function operates on: ptr, conv, id, token, paginate
□ Never create utils/, helper/, or common/ — these become grab-bags
□ Each pkg/* sub-package imports nothing internal
□ pkg/ptr    — To[T], From[T], FromOr[T]   — pointer ↔ value
□ pkg/conv   — SliceMap, Filter, Keys       — generic collection transforms
□ pkg/id     — UUID, ULID, Hex, Short       — ID generation (use in usecase only)
□ pkg/token  — Generate, Hash               — opaque token + SHA-256 hash
□ pkg/paginate — FromRequest, NewMeta       — pagination sanitisation + metadata
□ pkg/must   — Do, Get                      — init-time only, never in handlers
□ ID generation belongs in usecase — never in repository or handler
□ Token hashing: store hash in DB, send raw to user — only usecase knows both
□ must.Get / must.Do only in main(), bootstrap, init() — panic is intentional there
```
