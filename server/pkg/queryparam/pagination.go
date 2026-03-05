package queryparam

import (
	"strings"

	"github.com/justblue/samsa/config"
)

const (
	SortSeqAsc  = "ASC"
	SortSeqDesc = "DESC"
)

// orderEntry preserves the original insertion order for deterministic SQL output.
type orderEntry struct {
	column string
	seq    string
}

// PaginationParams holds common list-endpoint query params.
// Embed this in every entity-specific param struct.
//
//	type ListUsersParams struct {
//	    queryparam.PaginationParams
//	    Name   *string `query:"name"`
//	    Active *bool   `query:"active"`
//	}
type PaginationParams struct {
	Page    int32    `query:"page" json:"page"`
	Limit   int32    `query:"limit" json:"limit"`
	OrderBy []string `query:"order_by" example:"column_name:desc" json:"order_by"`

	columnMappings map[string]string
	// orderMap is kept for GetOrderBy() callers; orderEntries drives ToSQL()
	// so that output is deterministic (map iteration order is random in Go).
	orderMap     map[string]string
	orderEntries []orderEntry
}

// Normalize applies defaults and clamps Limit. Call after Decode().
func (p *PaginationParams) Normalize(opts ...PaginationOption) {
	cfg := defaultPaginationConfig()
	for _, o := range opts {
		o(&cfg)
	}

	p.applyDefaults(cfg)
	p.buildOrderBy(cfg)
}

func (p *PaginationParams) applyDefaults(cfg paginationConfig) {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.Limit <= 0 {
		p.Limit = cfg.DefaultLimit
	}
	if p.Limit > cfg.MaxLimit {
		p.Limit = cfg.MaxLimit
	}

	if len(p.OrderBy) == 0 && cfg.DefaultOrderBy != "" {
		p.OrderBy = []string{cfg.DefaultOrderBy}
	}
}

func (p *PaginationParams) buildOrderBy(cfg paginationConfig) {
	if len(cfg.AllowOrderWithSQLC) > 0 {
		p.buildOrderFromSQLC(cfg.AllowOrderWithSQLC)
		return
	}

	p.buildOrderFromMapping(cfg.AllowOrderWith)
}

func (p *PaginationParams) buildOrderFromSQLC(entries []string) {
	p.orderMap = make(map[string]string)
	p.orderEntries = p.orderEntries[:0]

	for _, order := range entries {
		entry := p.parseSQLCEntry(order)
		if entry == nil {
			continue
		}

		if _, seen := p.orderMap[entry.column]; seen {
			continue
		}

		p.orderMap[entry.column] = entry.seq
		p.orderEntries = append(p.orderEntries, *entry)
	}

	p.columnMappings = make(map[string]string)
}

func (p *PaginationParams) parseSQLCEntry(order string) *orderEntry {
	if order == "" {
		return nil
	}

	idx := strings.LastIndex(order, "_")
	if idx == -1 || idx == len(order)-1 {
		return nil
	}

	field := order[:idx]
	seqStr := strings.ToLower(order[idx+1:])

	seq := SortSeqAsc
	switch seqStr {
	case "asc":
		seq = SortSeqAsc
	case "desc":
		seq = SortSeqDesc
	default:
		return nil
	}

	return &orderEntry{column: field, seq: seq}
}

func (p *PaginationParams) buildOrderFromMapping(mappings map[string]string) {
	allowedSet := make(map[string]bool, len(mappings))
	for f := range mappings {
		allowedSet[f] = true
	}

	p.orderMap = make(map[string]string)
	p.orderEntries = p.orderEntries[:0]

	for _, order := range p.OrderBy {
		entry := p.parseMappingEntry(order, allowedSet)
		if entry == nil {
			continue
		}

		if _, seen := p.orderMap[entry.column]; seen {
			continue
		}

		col := entry.column
		if mapped, ok := mappings[entry.column]; ok {
			col = mapped
		}

		p.orderMap[entry.column] = entry.seq
		p.orderEntries = append(p.orderEntries, orderEntry{column: col, seq: entry.seq})
	}

	p.columnMappings = mappings
}

func (p *PaginationParams) parseMappingEntry(order string, allowedSet map[string]bool) *orderEntry {
	if order == "" {
		return nil
	}

	parts := strings.SplitN(order, ":", 2)
	field := parts[0]

	if len(allowedSet) > 0 && !allowedSet[field] {
		return nil
	}

	seq := SortSeqAsc
	if len(parts) == 2 {
		switch strings.ToLower(parts[1]) {
		case "asc":
			seq = SortSeqAsc
		case "desc":
			seq = SortSeqDesc
		default:
			return nil
		}
	}

	return &orderEntry{column: field, seq: seq}
}

// GetOffset returns the SQL OFFSET value derived from Page and Limit.
func (p *PaginationParams) GetOffset() int32 {
	if p.Page <= 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}

// GetLimit returns the SQL LIMIT value derived from Limit.
func (p *PaginationParams) GetLimit() int32 {
	return p.Limit
}

// GetOrderBy returns field → direction ("ASC"/"DESC") pairs.
func (p *PaginationParams) GetOrderBy() map[string]string {
	return p.orderMap
}

// ToSQL generates a deterministic ORDER BY clause.
//
//	params.Normalize(
//	    queryparam.AllowOrderWith(map[string]string{
//	        "name":       "u.name",
//	        "created_at": "u.created_at",
//	    }),
//	)
//	clause := params.ToSQL() // → "u.name ASC, u.created_at DESC"
func (p *PaginationParams) ToSQL() string {
	if len(p.orderEntries) == 0 {
		return ""
	}
	clauses := make([]string, len(p.orderEntries))
	for i, e := range p.orderEntries {
		clauses[i] = e.column + " " + e.seq
	}
	return strings.Join(clauses, ", ")
}

// GetOrderByEntry returns the first order entry.
func (p *PaginationParams) GetOrderByEntry() string {
	if len(p.orderEntries) == 0 {
		return ""
	}
	for _, e := range p.orderEntries {
		return strings.ToLower(e.column + "_" + e.seq)
	}
	return ""
}

// ─── Options ──────────────────────────────────────────────────────────────────

type paginationConfig struct {
	DefaultLimit       int32
	MaxLimit           int32
	DefaultOrderBy     string
	AllowOrderWith     map[string]string
	AllowOrderWithSQLC []string
}

func defaultPaginationConfig() paginationConfig {
	return paginationConfig{DefaultLimit: 20, MaxLimit: 100}
}

// PaginationOption is a functional option for Normalize.
type PaginationOption func(*paginationConfig)

// WithDefaultLimit sets the default page size.
func WithDefaultLimit(n int32) PaginationOption {
	return func(c *paginationConfig) { c.DefaultLimit = n }
}

// WithMaxLimit sets the maximum allowed page size.
func WithMaxLimit(n int32) PaginationOption {
	return func(c *paginationConfig) { c.MaxLimit = n }
}

// WithDefaultOrderBy sets the fallback order-by expression (e.g. "created_at:desc").
func WithDefaultOrderBy(orderBy string) PaginationOption {
	return func(c *paginationConfig) { c.DefaultOrderBy = orderBy }
}

// AllowOrderWith maps client-facing field names to their SQL column expressions.
// Only fields present in this map will be accepted; pass nil to allow all.
func AllowOrderWith(fields map[string]string) PaginationOption {
	return func(c *paginationConfig) { c.AllowOrderWith = fields }
}

// AllowOrderWithSQLC defines allowed order entries for sqlc integration.
// Input format: []string{"created_at_asc", "updated_at_desc", ...}
// Parses field and direction from underscore-separated string (e.g., "created_at_asc" -> field: "created_at", seq: "ASC")
func AllowOrderWithSQLC(entries []string) PaginationOption {
	return func(c *paginationConfig) { c.AllowOrderWithSQLC = entries }
}

// ─── Pagination Metadata ──────────────────────────────────────────────────────

// PaginationMeta is the pagination envelope included in list responses.
type PaginationMeta struct {
	Page       int32 `json:"page"`
	Limit      int32 `json:"limit"`
	TotalCount int64 `json:"total_count"`
	TotalPages int64 `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// NewPaginationMeta builds PaginationMeta from normalised params and a total count.
func NewPaginationMeta(page, limit int32, totalCount int64) PaginationMeta {
	if page <= 0 {
		page = config.APIPaginationDefaultPage
	}
	if limit <= 0 {
		limit = config.APIPaginationDefaultLimit
	}

	totalPages := int64(1)
	if totalCount > 0 {
		l := int64(limit)
		totalPages = (totalCount + l - 1) / l
	}

	return PaginationMeta{
		Page:       page,
		Limit:      limit,
		TotalCount: totalCount,
		TotalPages: totalPages,
		HasNext:    int64(page) < totalPages,
		HasPrev:    page > 1,
	}
}
