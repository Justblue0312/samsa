package assert

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

// Database assertions for testing repository layer

// RowExists checks if a row exists in the database with the given query and arguments.
func RowExists(t *testing.T, pool *pgxpool.Pool, query string, args ...interface{}) bool {
	t.Helper()

	var exists bool
	err := pool.QueryRow(context.Background(), query, args...).Scan(&exists)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		t.Fatalf("assert: failed to check row existence: %v", err)
	}
	return exists
}

// CountRows returns the count of rows matching the given query and arguments.
func CountRows(t *testing.T, pool *pgxpool.Pool, query string, args ...interface{}) int64 {
	t.Helper()

	var count int64
	err := pool.QueryRow(context.Background(), query, args...).Scan(&count)
	if err != nil {
		t.Fatalf("assert: failed to count rows: %v", err)
	}
	return count
}

// TableIsEmpty checks if the given table is empty.
func TableIsEmpty(t *testing.T, pool *pgxpool.Pool, tableName string) bool {
	t.Helper()

	query := "SELECT COUNT(*) FROM " + tableName
	return CountRows(t, pool, query) == 0
}

// TableCount returns the number of rows in the given table.
func TableCount(t *testing.T, pool *pgxpool.Pool, tableName string) int64 {
	t.Helper()

	query := "SELECT COUNT(*) FROM " + tableName
	return CountRows(t, pool, query)
}

// ColumnEquals checks if a specific column value matches the expected value.
func ColumnEquals(t *testing.T, pool *pgxpool.Pool, table, column string, id uuid.UUID, expected interface{}) bool {
	t.Helper()

	query := "SELECT " + column + " FROM " + table + " WHERE id = $1"
	var actual interface{}
	err := pool.QueryRow(context.Background(), query, id).Scan(&actual)
	if err != nil {
		t.Fatalf("assert: failed to get column value: %v", err)
	}
	return assert.Equal(t, expected, actual)
}

// IsDeleted checks if a record is marked as deleted.
func IsDeleted(t *testing.T, pool *pgxpool.Pool, table string, id uuid.UUID) bool {
	t.Helper()

	query := "SELECT is_deleted FROM " + table + " WHERE id = $1"
	var deleted bool
	err := pool.QueryRow(context.Background(), query, id).Scan(&deleted)
	if err == sql.ErrNoRows {
		return true // Consider non-existent rows as deleted
	}
	if err != nil {
		t.Fatalf("assert: failed to check deleted status: %v", err)
	}
	return deleted
}

// HasDeletedAt checks if a record has a deleted_at timestamp.
func HasDeletedAt(t *testing.T, pool *pgxpool.Pool, table string, id uuid.UUID) bool {
	t.Helper()

	query := "SELECT deleted_at FROM " + table + " WHERE id = $1"
	var deletedAt *time.Time
	err := pool.QueryRow(context.Background(), query, id).Scan(&deletedAt)
	if err == sql.ErrNoRows {
		return true
	}
	if err != nil {
		t.Fatalf("assert: failed to check deleted_at: %v", err)
	}
	return deletedAt != nil
}

// RecordExists checks if a record exists in the given table with the specified ID.
func RecordExists(t *testing.T, pool *pgxpool.Pool, table string, id uuid.UUID) bool {
	t.Helper()

	query := "SELECT EXISTS(SELECT 1 FROM " + table + " WHERE id = $1)"
	var exists bool
	err := pool.QueryRow(context.Background(), query, id).Scan(&exists)
	if err != nil {
		t.Fatalf("assert: failed to check record existence: %v", err)
	}
	return exists
}

// JSONEquals compares two JSON structures for equality.
func JSONEquals(t *testing.T, expected, actual interface{}) bool {
	t.Helper()

	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		t.Fatalf("assert: failed to marshal expected JSON: %v", err)
	}

	actualJSON, err := json.Marshal(actual)
	if err != nil {
		t.Fatalf("assert: failed to marshal actual JSON: %v", err)
	}

	var expectedMap, actualMap interface{}
	if err := json.Unmarshal(expectedJSON, &expectedMap); err != nil {
		t.Fatalf("assert: failed to unmarshal expected JSON: %v", err)
	}
	if err := json.Unmarshal(actualJSON, &actualMap); err != nil {
		t.Fatalf("assert: failed to unmarshal actual JSON: %v", err)
	}

	return assert.Equal(t, expectedMap, actualMap)
}

// TimeEquals checks if two times are equal within a tolerance.
func TimeEquals(t *testing.T, expected, actual *time.Time, tolerance time.Duration) bool {
	t.Helper()

	if expected == nil && actual == nil {
		return true
	}
	if expected == nil || actual == nil {
		return assert.Fail(t, "time mismatch: one is nil", "expected: %v, actual: %v", expected, actual)
	}

	diff := expected.Sub(*actual)
	if diff < 0 {
		diff = -diff
	}

	if diff > tolerance {
		return assert.Fail(t, "time mismatch", "expected: %v, actual: %v, diff: %v", expected, actual, diff)
	}

	return true
}

// SliceContains checks if a slice contains the given element.
func SliceContains[T comparable](t *testing.T, slice []T, element T) bool {
	t.Helper()

	for _, item := range slice {
		if item == element {
			return true
		}
	}
	return assert.Fail(t, "slice does not contain element", "element: %v", element)
}

// SliceLen checks if a slice has the expected length.
func SliceLen[T any](t *testing.T, slice []T, expectedLen int) bool {
	t.Helper()
	return assert.Len(t, slice, expectedLen)
}

// SliceEmpty checks if a slice is empty.
func SliceEmpty[T any](t *testing.T, slice []T) bool {
	t.Helper()
	return assert.Empty(t, slice)
}

// SliceNotEmpty checks if a slice is not empty.
func SliceNotEmpty[T any](t *testing.T, slice []T) bool {
	t.Helper()
	return assert.NotEmpty(t, slice)
}

// UUIDEquals checks if two UUIDs are equal.
func UUIDEquals(t *testing.T, expected, actual uuid.UUID) bool {
	t.Helper()
	return assert.Equal(t, expected, actual)
}

// UUIDIsNil checks if a UUID is nil (all zeros).
func UUIDIsNil(t *testing.T, id uuid.UUID) bool {
	t.Helper()
	return assert.Equal(t, uuid.Nil, id)
}

// UUIDNotNil checks if a UUID is not nil.
func UUIDNotNil(t *testing.T, id uuid.UUID) bool {
	t.Helper()
	return assert.NotEqual(t, uuid.Nil, id)
}

// PtrEquals checks if two pointers have equal values.
func PtrEquals[T comparable](t *testing.T, expected, actual *T) bool {
	t.Helper()

	if expected == nil && actual == nil {
		return true
	}
	if expected == nil || actual == nil {
		return assert.Fail(t, "pointer mismatch: one is nil", "expected: %v, actual: %v", expected, actual)
	}
	return assert.Equal(t, *expected, *actual)
}
