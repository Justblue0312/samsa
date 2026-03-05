package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/internal/testkit"
	"github.com/justblue/samsa/internal/testkit/fixtures"
)

// Suite provides a base test suite for integration tests.
// It manages database connections, transactions, and common test setup.
type Suite struct {
	t       *testing.T
	Pool    *pgxpool.Pool
	Tx      TestTX
	Config  *config.Config
	Queries Querier
	ctx     context.Context
	cancel  context.CancelFunc
}

// TestTX is an interface for database transactions used in tests.
type TestTX interface {
	// Add any methods you need from pgx.Tx
	// This is a placeholder - in practice, you'd embed the actual tx
}

// Querier is an interface for database queries (matches sqlc.Queries).
type Querier interface {
	// Add the query methods you need
	// This is a placeholder - use sqlc.Queries in practice
}

// NewSuite creates a new test suite.
// Call Setup() to initialize the database connection and transaction.
func NewSuite(t *testing.T) *Suite {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())

	return &Suite{
		t:      t,
		Config: testkit.SetupConfig(),
		ctx:    ctx,
		cancel: cancel,
	}
}

// Setup initializes the database connection and starts a transaction.
// Call this in your test's setup phase.
func (s *Suite) Setup() {
	s.t.Helper()

	s.Pool = testkit.NewDB(s.t)
	s.Tx = testkit.NewTx(s.t, s.Pool)

	// Initialize queries with the transaction
	// s.Queries = sqlc.New(s.Tx)

	s.t.Cleanup(s.Teardown)
}

// Teardown cleans up after the test.
// The transaction is automatically rolled back.
func (s *Suite) Teardown() {
	s.t.Helper()
	s.cancel()
}

// Context returns the test context.
func (s *Suite) Context() context.Context {
	return s.ctx
}

// NewContext creates a new context with timeout.
func (s *Suite) NewContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(s.ctx, timeout)
}

// Truncate truncates the specified tables.
func (s *Suite) Truncate(tables ...string) {
	s.t.Helper()
	fixtures.TruncateAll(s.t, s.Pool)
}

// RequestBuilder helps build HTTP requests for testing handlers.
type RequestBuilder struct {
	t       *testing.T
	method  string
	path    string
	body    []byte
	headers map[string]string
	params  map[string]string
}

// NewRequest creates a new request builder.
func NewRequest(t *testing.T, method, path string) *RequestBuilder {
	t.Helper()

	return &RequestBuilder{
		t:       t,
		method:  method,
		path:    path,
		headers: make(map[string]string),
		params:  make(map[string]string),
	}
}

// WithBody sets the request body.
func (b *RequestBuilder) WithBody(body []byte) *RequestBuilder {
	b.body = body
	return b
}

// WithJSONBody sets the request body as JSON.
func (b *RequestBuilder) WithJSONBody(json []byte) *RequestBuilder {
	b.body = json
	b.headers["Content-Type"] = "application/json"
	return b
}

// WithHeader adds a header to the request.
func (b *RequestBuilder) WithHeader(key, value string) *RequestBuilder {
	b.headers[key] = value
	return b
}

// WithParam adds a URL parameter to the request.
func (b *RequestBuilder) WithParam(key, value string) *RequestBuilder {
	b.params[key] = value
	return b
}

// WithAuth adds an authorization header.
func (b *RequestBuilder) WithAuth(token string) *RequestBuilder {
	b.headers["Authorization"] = "Bearer " + token
	return b
}

// Build creates the HTTP request.
func (b *RequestBuilder) Build() *http.Request {
	var req *http.Request
	if b.body != nil {
		req = httptest.NewRequest(b.method, b.path, nil)
		req.Body = &nopCloser{reader: httptest.NewRequest(b.method, b.path, nil).Body}
		// Re-create with body
		req = httptest.NewRequest(b.method, b.path, nil)
	} else {
		req = httptest.NewRequest(b.method, b.path, nil)
	}

	// Set headers
	for key, value := range b.headers {
		req.Header.Set(key, value)
	}

	// Set URL parameters using chi's URLContext
	if len(b.params) > 0 {
		rctx := chi.NewRouteContext()
		for key, value := range b.params {
			rctx.URLParams.Add(key, value)
		}
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	}

	return req
}

// Execute sends the request to the handler and returns the response recorder.
func (b *RequestBuilder) Execute(handler http.HandlerFunc) *httptest.ResponseRecorder {
	req := b.Build()
	rr := httptest.NewRecorder()
	handler(rr, req)
	return rr
}

// ExecuteWithMiddleware sends the request through middleware and handler.
func (b *RequestBuilder) ExecuteWithMiddleware(middleware func(http.Handler) http.Handler, handler http.HandlerFunc) *httptest.ResponseRecorder {
	req := b.Build()
	rr := httptest.NewRecorder()

	h := middleware(handler)
	h.ServeHTTP(rr, req)

	return rr
}

// ResponseAssertions provides methods for asserting HTTP response properties.
type ResponseAssertions struct {
	t  *testing.T
	rr *httptest.ResponseRecorder
}

// Assert creates a new response assertions helper.
func Assert(t *testing.T, rr *httptest.ResponseRecorder) *ResponseAssertions {
	t.Helper()
	return &ResponseAssertions{
		t:  t,
		rr: rr,
	}
}

// Status asserts the response status code.
func (a *ResponseAssertions) Status(expected int) *ResponseAssertions {
	a.t.Helper()
	if a.rr.Code != expected {
		a.t.Errorf("expected status %d, got %d", expected, a.rr.Code)
	}
	return a
}

// OK asserts status 200.
func (a *ResponseAssertions) OK() *ResponseAssertions {
	a.t.Helper()
	return a.Status(http.StatusOK)
}

// Created asserts status 201.
func (a *ResponseAssertions) Created() *ResponseAssertions {
	a.t.Helper()
	return a.Status(http.StatusCreated)
}

// NoContent asserts status 204.
func (a *ResponseAssertions) NoContent() *ResponseAssertions {
	a.t.Helper()
	return a.Status(http.StatusNoContent)
}

// BadRequest asserts status 400.
func (a *ResponseAssertions) BadRequest() *ResponseAssertions {
	a.t.Helper()
	return a.Status(http.StatusBadRequest)
}

// Unauthorized asserts status 401.
func (a *ResponseAssertions) Unauthorized() *ResponseAssertions {
	a.t.Helper()
	return a.Status(http.StatusUnauthorized)
}

// Forbidden asserts status 403.
func (a *ResponseAssertions) Forbidden() *ResponseAssertions {
	a.t.Helper()
	return a.Status(http.StatusForbidden)
}

// NotFound asserts status 404.
func (a *ResponseAssertions) NotFound() *ResponseAssertions {
	a.t.Helper()
	return a.Status(http.StatusNotFound)
}

// ServerError asserts status 500.
func (a *ResponseAssertions) ServerError() *ResponseAssertions {
	a.t.Helper()
	return a.Status(http.StatusInternalServerError)
}

// BodyContains asserts the response body contains a substring.
func (a *ResponseAssertions) BodyContains(substring string) *ResponseAssertions {
	a.t.Helper()
	body := a.rr.Body.String()
	if !contains(body, substring) {
		a.t.Errorf("expected body to contain %q, got %q", substring, body)
	}
	return a
}

// BodyEquals asserts the response body equals the expected string.
func (a *ResponseAssertions) BodyEquals(expected string) *ResponseAssertions {
	a.t.Helper()
	body := a.rr.Body.String()
	if body != expected {
		a.t.Errorf("expected body %q, got %q", expected, body)
	}
	return a
}

// HeaderEquals asserts a response header equals the expected value.
func (a *ResponseAssertions) HeaderEquals(key, expected string) *ResponseAssertions {
	a.t.Helper()
	value := a.rr.Header().Get(key)
	if value != expected {
		a.t.Errorf("expected header %s to be %q, got %q", key, expected, value)
	}
	return a
}

// ContentTypeJSON asserts the content type is application/json.
func (a *ResponseAssertions) ContentTypeJSON() *ResponseAssertions {
	a.t.Helper()
	return a.HeaderContains("Content-Type", "application/json")
}

// HeaderContains asserts a response header contains a substring.
func (a *ResponseAssertions) HeaderContains(key, substring string) *ResponseAssertions {
	a.t.Helper()
	value := a.rr.Header().Get(key)
	if !contains(value, substring) {
		a.t.Errorf("expected header %s to contain %q, got %q", key, substring, value)
	}
	return a
}

// nopCloser implements io.ReadCloser for request bodies.
type nopCloser struct {
	reader interface{}
}

func (n *nopCloser) Read(p []byte) (int, error) {
	return 0, nil
}

func (n *nopCloser) Close() error {
	return nil
}

// contains is a helper for string containment.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
