package assert

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// HTTP assertions for testing handler layer

// StatusCode checks if the response has the expected status code.
func StatusCode(t *testing.T, rr *httptest.ResponseRecorder, expected int) bool {
	t.Helper()
	return assert.Equal(t, expected, rr.Code)
}

// StatusOK checks if the response status is 200 OK.
func StatusOK(t *testing.T, rr *httptest.ResponseRecorder) bool {
	t.Helper()
	return StatusCode(t, rr, http.StatusOK)
}

// StatusCreated checks if the response status is 201 Created.
func StatusCreated(t *testing.T, rr *httptest.ResponseRecorder) bool {
	t.Helper()
	return StatusCode(t, rr, http.StatusCreated)
}

// StatusNoContent checks if the response status is 204 No Content.
func StatusNoContent(t *testing.T, rr *httptest.ResponseRecorder) bool {
	t.Helper()
	return StatusCode(t, rr, http.StatusNoContent)
}

// StatusBadRequest checks if the response status is 400 Bad Request.
func StatusBadRequest(t *testing.T, rr *httptest.ResponseRecorder) bool {
	t.Helper()
	return StatusCode(t, rr, http.StatusBadRequest)
}

// StatusUnauthorized checks if the response status is 401 Unauthorized.
func StatusUnauthorized(t *testing.T, rr *httptest.ResponseRecorder) bool {
	t.Helper()
	return StatusCode(t, rr, http.StatusUnauthorized)
}

// StatusForbidden checks if the response status is 403 Forbidden.
func StatusForbidden(t *testing.T, rr *httptest.ResponseRecorder) bool {
	t.Helper()
	return StatusCode(t, rr, http.StatusForbidden)
}

// StatusNotFound checks if the response status is 404 Not Found.
func StatusNotFound(t *testing.T, rr *httptest.ResponseRecorder) bool {
	t.Helper()
	return StatusCode(t, rr, http.StatusNotFound)
}

// StatusConflict checks if the response status is 409 Conflict.
func StatusConflict(t *testing.T, rr *httptest.ResponseRecorder) bool {
	t.Helper()
	return StatusCode(t, rr, http.StatusConflict)
}

// StatusInternalServerError checks if the response status is 500 Internal Server Error.
func StatusInternalServerError(t *testing.T, rr *httptest.ResponseRecorder) bool {
	t.Helper()
	return StatusCode(t, rr, http.StatusInternalServerError)
}

// HeaderEquals checks if a response header has the expected value.
func HeaderEquals(t *testing.T, rr *httptest.ResponseRecorder, key, expected string) bool {
	t.Helper()
	return assert.Equal(t, expected, rr.Header().Get(key))
}

// HeaderContains checks if a response header contains the given value.
func HeaderContains(t *testing.T, rr *httptest.ResponseRecorder, key, substring string) bool {
	t.Helper()
	value := rr.Header().Get(key)
	return assert.Contains(t, value, substring)
}

// ContentTypeJSON checks if the response content type is application/json.
func ContentTypeJSON(t *testing.T, rr *httptest.ResponseRecorder) bool {
	t.Helper()
	return HeaderContains(t, rr, "Content-Type", "application/json")
}

// JSONBody decodes the response body as JSON into the provided struct.
func JSONBody(t *testing.T, rr *httptest.ResponseRecorder, target interface{}) error {
	t.Helper()
	err := json.NewDecoder(rr.Body).Decode(target)
	if err != nil {
		t.Fatalf("assert: failed to decode JSON body: %v", err)
	}
	return err
}

// JSONFieldEquals checks if a specific field in the JSON response equals the expected value.
func JSONFieldEquals(t *testing.T, rr *httptest.ResponseRecorder, field string, expected interface{}) bool {
	t.Helper()

	var body map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("assert: failed to decode JSON body: %v", err)
	}

	value, ok := body[field]
	if !ok {
		return assert.Fail(t, "field not found", "field: %s", field)
	}

	return assert.Equal(t, expected, value)
}

// JSONPathEquals checks if a nested field in the JSON response equals the expected value.
// Path is dot-separated, e.g., "data.user.id".
func JSONPathEquals(t *testing.T, rr *httptest.ResponseRecorder, path string, expected interface{}) bool {
	t.Helper()

	var body interface{}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("assert: failed to decode JSON body: %v", err)
	}

	keys := splitPath(path)
	current := body

	for i, key := range keys {
		switch v := current.(type) {
		case map[string]interface{}:
			var ok bool
			current, ok = v[key]
			if !ok {
				return assert.Fail(t, "path not found", "path: %s, at key: %s", path, key)
			}
		case []interface{}:
			// Try to parse key as index
			var index int
			if _, err := fmt.Sscanf(key, "%d", &index); err != nil {
				return assert.Fail(t, "invalid array index", "key: %s", key)
			}
			if index < 0 || index >= len(v) {
				return assert.Fail(t, "array index out of bounds", "index: %d, length: %d", index, len(v))
			}
			current = v[index]
		default:
			return assert.Fail(t, "invalid path", "path: %s, at position: %d", path, i)
		}
	}

	return assert.Equal(t, expected, current)
}

// BodyEquals checks if the response body equals the expected string.
func BodyEquals(t *testing.T, rr *httptest.ResponseRecorder, expected string) bool {
	t.Helper()
	return assert.Equal(t, expected, rr.Body.String())
}

// BodyContains checks if the response body contains the given substring.
func BodyContains(t *testing.T, rr *httptest.ResponseRecorder, substring string) bool {
	t.Helper()
	return assert.Contains(t, rr.Body.String(), substring)
}

// BodyNotEmpty checks if the response body is not empty.
func BodyNotEmpty(t *testing.T, rr *httptest.ResponseRecorder) bool {
	t.Helper()
	return assert.NotEmpty(t, rr.Body.String())
}

// ResponseStruct decodes the JSON response and compares it with the expected struct.
func ResponseStruct(t *testing.T, rr *httptest.ResponseRecorder, expected interface{}) bool {
	t.Helper()

	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		t.Fatalf("assert: failed to marshal expected struct: %v", err)
	}

	var expectedMap map[string]interface{}
	if err := json.Unmarshal(expectedJSON, &expectedMap); err != nil {
		t.Fatalf("assert: failed to unmarshal expected JSON: %v", err)
	}

	var actualMap map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&actualMap); err != nil {
		t.Fatalf("assert: failed to decode actual JSON: %v", err)
	}

	return assert.Equal(t, expectedMap, actualMap)
}

// PaginationMeta checks if the response contains correct pagination metadata.
func PaginationMeta(t *testing.T, rr *httptest.ResponseRecorder, expectedTotal, expectedLimit, expectedOffset int64) bool {
	t.Helper()

	var body map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("assert: failed to decode JSON body: %v", err)
	}

	meta, ok := body["meta"].(map[string]interface{})
	if !ok {
		return assert.Fail(t, "meta field not found or not an object")
	}

	success := true
	if total, ok := meta["total"].(float64); ok {
		if int64(total) != expectedTotal {
			success = assert.Fail(t, "total mismatch", "expected: %d, got: %d", expectedTotal, int64(total))
		}
	}
	if limit, ok := meta["limit"].(float64); ok {
		if int64(limit) != expectedLimit {
			success = assert.Fail(t, "limit mismatch", "expected: %d, got: %d", expectedLimit, int64(limit))
		}
	}
	if offset, ok := meta["offset"].(float64); ok {
		if int64(offset) != expectedOffset {
			success = assert.Fail(t, "offset mismatch", "expected: %d, got: %d", expectedOffset, int64(offset))
		}
	}

	return success
}

// splitPath splits a dot-separated path into keys.
func splitPath(path string) []string {
	// Simple implementation - can be enhanced to handle brackets for arrays
	return strings.Split(path, ".")
}
