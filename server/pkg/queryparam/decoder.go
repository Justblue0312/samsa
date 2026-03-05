package queryparam

import (
	"encoding"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

const tagName = "query"

// uuidType is the reflect.Type for uuid.UUID.
var uuidType = reflect.TypeFor[uuid.UUID]()

// Decode maps URL query parameters into a tagged struct.
//
// Supported field types:
//   - string, bool, int/int8/int16/int32/int64
//   - uint/uint8/uint16/uint32/uint64, float32/float64
//   - uuid.UUID
//   - Any type implementing encoding.TextUnmarshaler  ← enums go here
//   - Pointers to any of the above (*string, *int, *uuid.UUID, *Status, ...)
//   - Slices of any of the above ([]string, []uuid.UUID, []Status, ...)
//   - Embedded structs (anonymous fields, no tag needed)
//
// Enum example:
//
//	type Status string
//	const (StatusActive Status = "active"; StatusBanned Status = "banned")
//	func (s *Status) UnmarshalText(b []byte) error {
//	    switch Status(b) {
//	    case StatusActive, StatusBanned:
//	        *s = Status(b); return nil
//	    }
//	    return fmt.Errorf("invalid status %q", b)
//	}
//
//	type ListUsersParams struct {
//	    queryparam.PaginationParams
//	    Status  *Status      `query:"status"`    // optional enum
//	    Roles   []Role       `query:"role"`      // multi-value enum
//	    IDs     []uuid.UUID  `query:"id"`        // multi-value UUID
//	}
func Decode(dst any, values url.Values) error {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("queryparam: dst must be a non-nil pointer to a struct")
	}
	return decodeStruct(v.Elem(), values)
}

// DecodeRequest is a convenience wrapper that reads from *http.Request directly.
func DecodeRequest(dst any, rawQuery string) error {
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return fmt.Errorf("queryparam: failed to parse query string: %w", err)
	}
	return Decode(dst, values)
}

// ─── internal ───────────────────────────────────────────────────────────────

const opsTagName = "ops"

func decodeStruct(v reflect.Value, values url.Values) error {
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldVal := v.Field(i)

		// Recursively handle embedded (anonymous) structs, e.g. PaginationParams.
		if field.Anonymous {
			embedded := fieldVal
			if embedded.Kind() == reflect.Pointer {
				if embedded.IsNil() {
					embedded.Set(reflect.New(embedded.Type().Elem()))
				}
				embedded = embedded.Elem()
			}
			if embedded.Kind() == reflect.Struct {
				if err := decodeStruct(embedded, values); err != nil {
					return err
				}
				continue
			}
		}

		tag := field.Tag.Get(tagName)
		if tag == "" || tag == "-" {
			continue
		}

		// Support `query:"name,omitempty"` — only the name part is used.
		paramName := strings.SplitN(tag, ",", 2)[0]

		// ── Filter[T] path ───────────────────────────────────────────────────
		// Filter fields consume the full url.Values themselves (bracket keys like
		// "price[gte]") so they must be handled before the rawValues lookup below.
		if fieldVal.CanAddr() {
			if fd, ok := fieldVal.Addr().Interface().(filterDecoder); ok {
				allowedOps, err := parseOpsTag(field.Tag.Get(opsTagName))
				if err != nil {
					return fmt.Errorf("queryparam: field %q ops tag: %w", field.Name, err)
				}
				if err := fd.decodeFilter(values, paramName, allowedOps); err != nil {
					return fmt.Errorf("queryparam: field %q (param %q): %w", field.Name, paramName, err)
				}
				continue // do NOT fall through to setField
			}
		}

		// ── Scalar / slice / pointer path ────────────────────────────────────
		rawValues, ok := values[paramName]
		if !ok || len(rawValues) == 0 {
			continue
		}

		if err := setField(fieldVal, rawValues); err != nil {
			return fmt.Errorf("queryparam: field %q (param %q): %w", field.Name, paramName, err)
		}
	}
	return nil
}

func setField(fv reflect.Value, raws []string) error {
	// ── Pointer: allocate then recurse into the element ─────────────────────
	if fv.Kind() == reflect.Pointer {
		elem := reflect.New(fv.Type().Elem())
		if err := setField(elem.Elem(), raws); err != nil {
			return err
		}
		fv.Set(elem)
		return nil
	}

	// ── Slice ────────────────────────────────────────────────────────────────
	if fv.Kind() == reflect.Slice {
		return setSliceField(fv, raws)
	}

	// Delegate to parseValue which handles TextUnmarshaler, UUID, and scalars.
	val, err := parseValue(raws[0], fv.Type())
	if err != nil {
		return err
	}
	fv.Set(val)
	return nil
}

// parseValue parses a raw string into an addressable reflect.Value of targetType.
// Resolution order: encoding.TextUnmarshaler → uuid.UUID → scalar kinds.
// Shared by setField and Filter[T].decodeFilter so both paths stay in sync.
func parseValue(raw string, targetType reflect.Type) (reflect.Value, error) {
	fv := reflect.New(targetType).Elem()

	// ── TextUnmarshaler (enums and any custom type that implements it) ───────
	if u, ok := fv.Addr().Interface().(encoding.TextUnmarshaler); ok {
		if err := u.UnmarshalText([]byte(raw)); err != nil {
			return reflect.Value{}, err
		}
		return fv, nil
	}

	// ── uuid.UUID ────────────────────────────────────────────────────────────
	if targetType == uuidType {
		id, err := uuid.Parse(raw)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid UUID %q: %w", raw, err)
		}
		fv.Set(reflect.ValueOf(id))
		return fv, nil
	}

	// ── Scalar (string, bool, int*, uint*, float*) ───────────────────────────
	if err := setScalar(fv, raw); err != nil {
		return reflect.Value{}, err
	}
	return fv, nil
}

func setSliceField(fv reflect.Value, raws []string) error {
	// Expand both forms: ?tag=a,b  and  ?tag=a&tag=b  → ["a", "b"]
	var expanded []string
	for _, r := range raws {
		for part := range strings.SplitSeq(r, ",") {
			if part = strings.TrimSpace(part); part != "" {
				expanded = append(expanded, part)
			}
		}
	}

	elemType := fv.Type().Elem()
	slice := reflect.MakeSlice(fv.Type(), len(expanded), len(expanded))

	for i, raw := range expanded {
		elem := slice.Index(i)

		// ── []uuid.UUID ───────────────────────────────────────────────────────
		if elemType == uuidType {
			id, err := uuid.Parse(raw)
			if err != nil {
				return fmt.Errorf("element %d: invalid UUID %q: %w", i, raw, err)
			}
			elem.Set(reflect.ValueOf(id))
			continue
		}

		// ── []*T (pointer slice) ──────────────────────────────────────────────
		// Allocate a *T, then delegate to setField so all type logic is reused.
		if elemType.Kind() == reflect.Pointer {
			inner := reflect.New(elemType.Elem())
			if err := setField(inner.Elem(), []string{raw}); err != nil {
				return fmt.Errorf("element %d: %w", i, err)
			}
			elem.Set(inner)
			continue
		}

		// ── []SomeEnum / []string / []int etc. ────────────────────────────────
		// Delegate to setField so TextUnmarshaler and scalars are handled uniformly.
		if err := setField(elem, []string{raw}); err != nil {
			return fmt.Errorf("element %d: %w", i, err)
		}
	}

	fv.Set(slice)
	return nil
}

func setScalar(fv reflect.Value, raw string) error {
	switch fv.Kind() {
	case reflect.String:
		fv.SetString(raw)

	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("invalid bool %q", raw)
		}
		fv.SetBool(b)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 10, fv.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid int %q", raw)
		}
		fv.SetInt(n)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, fv.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid uint %q", raw)
		}
		fv.SetUint(n)

	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(raw, fv.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid float %q", raw)
		}
		fv.SetFloat(f)

	default:
		return fmt.Errorf("unsupported kind %s for type %s", fv.Kind(), fv.Type())
	}
	return nil
}
