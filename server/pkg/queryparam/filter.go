package queryparam

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
)

// ═══════════════════════════════════════════════════════════════════════════════
// CompareOp — validated operator enum
// ═══════════════════════════════════════════════════════════════════════════════

// CompareOp is a comparison operator used in bracket-style query params.
// e.g. ?price[gte]=10  ?name[like]=widget  ?status[in]=active,banned
type CompareOp string

const (
	OpEq    CompareOp = "eq"    // =
	OpNe    CompareOp = "ne"    // !=
	OpGt    CompareOp = "gt"    // >
	OpGte   CompareOp = "gte"   // >=
	OpLt    CompareOp = "lt"    // <
	OpLte   CompareOp = "lte"   // <=
	OpLike  CompareOp = "like"  // LIKE  (string fields only, auto-wraps in %)
	OpILike CompareOp = "ilike" // ILIKE (string fields only, case-insensitive)
	OpIn    CompareOp = "in"    // IN (a, b, c) — multi-value, comma or repeat
)

// allOps is the canonical iteration order used when scanning url.Values.
var allOps = []CompareOp{
	OpEq, OpNe, OpGt, OpGte, OpLt, OpLte, OpLike, OpILike, OpIn,
}

// validOps is used for fast lookup when validating the `ops` struct tag.
var validOps = map[CompareOp]bool{
	OpEq: true, OpNe: true, OpGt: true, OpGte: true,
	OpLt: true, OpLte: true, OpLike: true, OpILike: true, OpIn: true,
}

// stringOnlyOps lists operators that are only valid on string-kinded fields.
var stringOnlyOps = map[CompareOp]bool{
	OpLike: true, OpILike: true,
}

// ═══════════════════════════════════════════════════════════════════════════════
// Condition[T] and Filter[T]
// ═══════════════════════════════════════════════════════════════════════════════

// Condition is a single parsed comparison: one operator and one typed value.
type Condition[T any] struct {
	Op    CompareOp
	Value T
}

// Filter[T] holds all conditions decoded from the query string for one field.
// Declare it as the field type in your param struct and tag it with `query` and
// optionally `ops` to restrict which operators the client may use.
//
//	type ListProductsParams struct {
//	    queryparam.PaginationParams
//
//	    // Only gte/lte allowed; any other op returns 400.
//	    Price queryparam.Filter[float64] `query:"price" ops:"gte,lte"`
//
//	    // Only like/ilike allowed.
//	    Name  queryparam.Filter[string]  `query:"name"  ops:"like,ilike"`
//
//	    // All operators allowed (no ops tag).
//	    Stock queryparam.Filter[int]     `query:"stock"`
//
//	    // Works with enums and UUIDs too.
//	    Status queryparam.Filter[UserStatus]  `query:"status" ops:"eq,in"`
//	    OwnerID queryparam.Filter[uuid.UUID]  `query:"owner_id" ops:"eq"`
//	}
//
// Supported URL formats:
//
//	?price[gte]=10&price[lte]=500
//	?name[like]=widget             → LIKE '%widget%'
//	?status[in]=active,banned      → IN ('active', 'banned')
//	?status[in]=active&status[in]=banned  (same result)
//	?stock=5                       → shorthand for ?stock[eq]=5
type Filter[T any] struct {
	Conditions []Condition[T]
}

// IsEmpty returns true when no conditions were decoded (param was absent).
func (f *Filter[T]) IsEmpty() bool {
	return len(f.Conditions) == 0
}

// Has returns true if at least one condition with the given operator was decoded.
func (f *Filter[T]) Has(op CompareOp) bool {
	for _, c := range f.Conditions {
		if c.Op == op {
			return true
		}
	}
	return false
}

// Values returns all decoded values regardless of operator.
// Useful for quick iteration when building queries manually.
func (f *Filter[T]) Values() []T {
	out := make([]T, len(f.Conditions))
	for i, c := range f.Conditions {
		out[i] = c.Value
	}
	return out
}

// ─── filterDecoder interface ──────────────────────────────────────────────────

// filterDecoder is the internal interface checked by decodeStruct.
// Filter[T] satisfies it; nothing else in the package does.
type filterDecoder interface {
	decodeFilter(values url.Values, paramName string, allowedOps []CompareOp) error
}

// decodeFilter implements filterDecoder for Filter[T].
func (f *Filter[T]) decodeFilter(values url.Values, paramName string, allowedOps []CompareOp) error {
	// Build a set for O(1) lookup.
	allowedSet := make(map[CompareOp]bool, len(allowedOps))
	for _, op := range allowedOps {
		allowedSet[op] = true
	}
	hasRestriction := len(allowedOps) > 0

	// Determine the target type once; used for string-only op validation.
	var zero T
	targetType := reflect.TypeOf(zero)
	isStringKind := targetType.Kind() == reflect.String

	// ── Bare key shorthand: ?price=10  →  eq ────────────────────────────────
	if raws, ok := values[paramName]; ok && len(raws) > 0 {
		if hasRestriction && !allowedSet[OpEq] {
			return fmt.Errorf("operator %q is not allowed; permitted: %s", OpEq, joinOps(allowedOps))
		}
		val, err := parseValue(raws[0], targetType)
		if err != nil {
			return fmt.Errorf("[eq] %w", err)
		}
		f.Conditions = append(f.Conditions, Condition[T]{
			Op:    OpEq,
			Value: val.Interface().(T),
		})
	}

	// ── Bracket keys: ?price[gte]=10 ────────────────────────────────────────
	for _, op := range allOps {
		key := paramName + "[" + string(op) + "]"
		raws, ok := values[key]
		if !ok || len(raws) == 0 {
			continue
		}

		// Restrict to ops tag if present.
		if hasRestriction && !allowedSet[op] {
			return fmt.Errorf("operator %q is not allowed; permitted: %s", op, joinOps(allowedOps))
		}

		// Reject string-only operators on non-string types.
		if stringOnlyOps[op] && !isStringKind {
			return fmt.Errorf("operator %q is only valid for string fields, got %s", op, targetType)
		}

		// ── in: expand comma + multi-value, build one Condition per value ────
		if op == OpIn {
			var expanded []string
			for _, r := range raws {
				for part := range strings.SplitSeq(r, ",") {
					if part = strings.TrimSpace(part); part != "" {
						expanded = append(expanded, part)
					}
				}
			}
			for i, raw := range expanded {
				val, err := parseValue(raw, targetType)
				if err != nil {
					return fmt.Errorf("[in] element %d: %w", i, err)
				}
				f.Conditions = append(f.Conditions, Condition[T]{
					Op:    OpIn,
					Value: val.Interface().(T),
				})
			}
			continue
		}

		// ── All other ops: single value, duplicate op is an error ────────────
		if f.Has(op) {
			return fmt.Errorf("duplicate operator %q; provide each operator at most once", op)
		}

		val, err := parseValue(raws[0], targetType)
		if err != nil {
			return fmt.Errorf("[%s] %w", op, err)
		}
		f.Conditions = append(f.Conditions, Condition[T]{Op: op, Value: val.Interface().(T)})
	}

	return nil
}

// ─── ops tag parsing ──────────────────────────────────────────────────────────

// parseOpsTag parses the value of an `ops` struct tag into a []CompareOp.
//
//	ops:"gte,lte"   →  [OpGte, OpLte]
//	ops:""          →  nil  (all ops permitted)
//	ops:"bad"       →  error
func parseOpsTag(tag string) ([]CompareOp, error) {
	if strings.TrimSpace(tag) == "" {
		return nil, nil // no restriction
	}
	parts := strings.Split(tag, ",")
	ops := make([]CompareOp, 0, len(parts))
	for _, p := range parts {
		op := CompareOp(strings.TrimSpace(p))
		if !validOps[op] {
			return nil, fmt.Errorf("unknown operator %q; valid: %s", op, joinOps(allOps))
		}
		ops = append(ops, op)
	}
	return ops, nil
}

// joinOps formats a []CompareOp as a human-readable list for error messages.
func joinOps(ops []CompareOp) string {
	parts := make([]string, len(ops))
	for i, op := range ops {
		parts[i] = string(op)
	}
	return strings.Join(parts, ", ")
}
