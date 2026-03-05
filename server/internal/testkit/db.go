package testkit

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/gen/sqlc"
)

var (
	sharedPool     *pgxpool.Pool
	sharedQueries  *sqlc.Queries
	sharedPoolOnce sync.Once
	sharedPoolErr  error
)

// TestDatabase is a helper for managing isolated test databases.
type TestDatabase struct {
	Pool    *pgxpool.Pool
	Queries *sqlc.Queries
	T       *testing.T
}

// NewDB returns a shared pool — the container starts once per test binary.
// Each test gets isolation via NewTx, not via separate containers.
func NewDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	sharedPoolOnce.Do(func() {
		ctx := context.Background()

		connStr := fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=%s",
			getEnv("SAMSA_POSTGRES_USER", "samsa"),
			getEnv("SAMSA_POSTGRES_PWD", "samsa"),
			getEnv("SAMSA_POSTGRES_HOST", "localhost"),
			getEnvInt("SAMSA_POSTGRES_PORT", 5432),
			getEnv("SAMSA_POSTGRES_TEST_DATABASE", "samsa_test"),
			getEnv("SAMSA_POSTGRES_SSLMODE", "disable"),
		)

		pool, err := pgxpool.New(ctx, connStr)
		if err != nil {
			sharedPoolErr = fmt.Errorf("testkit: failed to create pool: %w", err)
			return
		}

		if err := pool.Ping(ctx); err != nil {
			sharedPoolErr = fmt.Errorf("testkit: failed to ping DB: %w", err)
			return
		}

		sharedPool = pool
		sharedQueries = sqlc.New(pool)
	})

	if sharedPoolErr != nil {
		t.Skipf("testkit: skipping — %v", sharedPoolErr)
	}

	return sharedPool
}

// NewQueries returns the shared sqlc.Queries instance backed by the shared pool.
func NewQueries(t *testing.T) *sqlc.Queries {
	t.Helper()
	_ = NewDB(t) // ensure pool is initialised
	return sharedQueries
}

// NewTx starts a transaction for the duration of a single test.
// The transaction is ALWAYS rolled back via t.Cleanup — even if the test panics.
// Pass the returned tx into your repository constructor instead of the pool.
func NewTx(t *testing.T, pool *pgxpool.Pool) pgx.Tx {
	t.Helper()

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("testkit: failed to begin tx: %v", err)
	}

	t.Cleanup(func() {
		_ = tx.Rollback(context.Background())
	})

	return tx
}

// SetupConfig returns a minimal test configuration with caching disabled.
func SetupConfig() *config.Config {
	return &config.Config{
		Cache: struct {
			EnableCache   bool          `default:"true" envx:"ENABLE_CACHE"`
			QueryCacheTTL time.Duration `default:"300s" envx:"QUERY_CACHE_TTL"`
		}{
			EnableCache: false,
		},
	}
}

// ── helpers ─────────────────────────────────────────────────────────────────

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		var i int
		if _, err := fmt.Sscanf(v, "%d", &i); err == nil {
			return i
		}
	}
	return fallback
}

// NewTestDatabase creates a new test database helper.
func NewTestDatabase(t *testing.T) *TestDatabase {
	t.Helper()

	pool := NewDB(t)
	return &TestDatabase{
		Pool:    pool,
		Queries: sqlc.New(pool),
		T:       t,
	}
}

// TruncateAll removes all data from the specified tables.
// Use this to clean up between tests when not using transactions.
func (db *TestDatabase) TruncateAll(tables ...string) {
	db.T.Helper()

	ctx := context.Background()

	if len(tables) == 0 {
		// Default set of tables to truncate
		tables = []string{
			"submission_status_histories",
			"submission_assignments",
			"submissions",
			"shared_files",
			"files",
			"comments",
			"story_posts",
			"story_votes",
			"story_status_histories",
			"stories",
			"comment_reactions",
			"comment_votes",
			"authors",
			"oauth_accounts",
			"sessions",
			"users",
		}
	}

	for _, table := range tables {
		query := fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)
		_, err := db.Pool.Exec(ctx, query)
		if err != nil {
			// Table might not exist, continue
			continue
		}
	}
}

// ResetSequences resets all sequences to start from 1.
func (db *TestDatabase) ResetSequences() {
	db.T.Helper()

	ctx := context.Background()

	query := `
		SELECT sequencename 
		FROM pg_sequences 
		WHERE schemaname = 'public'
	`

	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		db.T.Logf("testkit: failed to query sequences: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var seqName string
		if err := rows.Scan(&seqName); err != nil {
			continue
		}

		resetQuery := fmt.Sprintf("ALTER SEQUENCE %s RESTART WITH 1", seqName)
		_, err := db.Pool.Exec(ctx, resetQuery)
		if err != nil {
			db.T.Logf("testkit: failed to reset sequence %s: %v", seqName, err)
		}
	}
}

// CountRows returns the number of rows in a table.
func (db *TestDatabase) CountRows(table string) int64 {
	db.T.Helper()

	ctx := context.Background()
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)

	var count int64
	err := db.Pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		db.T.Fatalf("testkit: failed to count rows in %s: %v", table, err)
	}

	return count
}

// TableExists checks if a table exists.
func (db *TestDatabase) TableExists(table string) bool {
	db.T.Helper()

	ctx := context.Background()
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = $1
		)
	`

	var exists bool
	err := db.Pool.QueryRow(ctx, query, table).Scan(&exists)
	if err != nil {
		db.T.Fatalf("testkit: failed to check if table exists: %v", err)
	}

	return exists
}

// ExecuteSQL executes raw SQL statements.
func (db *TestDatabase) ExecuteSQL(sql string) {
	db.T.Helper()

	ctx := context.Background()

	// Split by semicolons to handle multiple statements
	statements := strings.Split(sql, ";")

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}

		_, err := db.Pool.Exec(ctx, stmt)
		if err != nil {
			db.T.Fatalf("testkit: failed to execute statement: %v\nSQL: %s", err, stmt)
		}
	}
}

// WithConnection executes a function with a dedicated connection.
func (db *TestDatabase) WithConnection(fn func(ctx context.Context, conn *pgxpool.Conn) error) error {
	db.T.Helper()

	ctx := context.Background()
	conn, err := db.Pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("testkit: failed to acquire connection: %w", err)
	}
	defer conn.Release()

	return fn(ctx, conn)
}

// Ping checks if the database is accessible.
func (db *TestDatabase) Ping() error {
	db.T.Helper()

	ctx := context.Background()
	return db.Pool.Ping(ctx)
}

// Close closes the database pool.
// Only call this if you created a dedicated pool (not the shared one).
func (db *TestDatabase) Close() {
	db.Pool.Close()
}

// TruncateAllTables is a package-level helper for truncating tables.
func TruncateAllTables(t *testing.T, pool *pgxpool.Pool, tables ...string) {
	t.Helper()

	ctx := context.Background()

	if len(tables) == 0 {
		tables = []string{
			"submission_status_histories",
			"submission_assignments",
			"submissions",
			"shared_files",
			"files",
			"comments",
			"story_posts",
			"story_votes",
			"story_status_histories",
			"stories",
			"comment_reactions",
			"comment_votes",
			"authors",
			"oauth_accounts",
			"sessions",
			"users",
		}
	}

	for _, table := range tables {
		query := fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)
		_, err := pool.Exec(ctx, query)
		if err != nil {
			// Table might not exist, continue
			continue
		}
	}
}

// IsTestDatabase checks if the current database is a test database.
func IsTestDatabase(pool *pgxpool.Pool) bool {
	ctx := context.Background()

	var dbName string
	err := pool.QueryRow(ctx, "SELECT current_database()").Scan(&dbName)
	if err != nil {
		return false
	}

	return strings.Contains(dbName, "test")
}

// RequireTestDatabase panics if the database is not a test database.
func RequireTestDatabase(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	if !IsTestDatabase(pool) {
		t.Fatal("testkit: not connected to a test database - refusing to run on production data")
	}
}
