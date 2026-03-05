package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justblue/samsa/gen/sqlc"
)

// FixturesLoader handles loading SQL fixtures into the test database.

// FixturesLoader loads and executes SQL fixtures.
type FixturesLoader struct {
	t       *testing.T
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewFixturesLoader creates a new fixtures loader.
func NewFixturesLoader(t *testing.T, pool *pgxpool.Pool) *FixturesLoader {
	t.Helper()
	return &FixturesLoader{
		t:       t,
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// Load executes a single SQL fixture file.
func (f *FixturesLoader) Load(path string) error {
	f.t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read fixture file: %w", err)
	}

	return f.Execute(string(content))
}

// LoadAll executes all SQL fixture files in a directory.
func (f *FixturesLoader) LoadAll(dir string) error {
	f.t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read fixtures directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		if err := f.Load(path); err != nil {
			return fmt.Errorf("failed to load fixture %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// Execute executes raw SQL statements.
func (f *FixturesLoader) Execute(sql string) error {
	f.t.Helper()

	ctx := context.Background()

	// Split by semicolons to handle multiple statements
	statements := splitSQLStatements(sql)

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}

		_, err := f.pool.Exec(ctx, stmt)
		if err != nil {
			return fmt.Errorf("failed to execute statement: %w\nSQL: %s", err, stmt)
		}
	}

	return nil
}

// LoadFixtureData loads test data using Go factories.
// This is preferred over SQL fixtures for complex test data.
func (f *FixturesLoader) LoadFixtureData(data interface{}) error {
	f.t.Helper()
	// This is a placeholder - in practice, you'd use the fixtures package
	// to create test data programmatically
	return nil
}

// splitSQLStatements splits SQL content into individual statements.
// This is a simple implementation - for production, use a proper SQL parser.
func splitSQLStatements(sql string) []string {
	var statements []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(sql); i++ {
		char := sql[i]

		// Handle quotes
		if char == '\'' || char == '"' {
			if !inQuote {
				inQuote = true
				quoteChar = char
			} else if char == quoteChar {
				inQuote = false
				quoteChar = 0
			}
			current.WriteByte(char)
			continue
		}

		// Handle semicolons outside quotes
		if char == ';' && !inQuote {
			statements = append(statements, current.String())
			current.Reset()
			continue
		}

		current.WriteByte(char)
	}

	// Add remaining content
	if current.Len() > 0 {
		statements = append(statements, current.String())
	}

	return statements
}

// TruncateTables removes all data from specified tables.
func (f *FixturesLoader) TruncateTables(tables ...string) error {
	f.t.Helper()

	ctx := context.Background()

	for _, table := range tables {
		query := fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)
		_, err := f.pool.Exec(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to truncate %s: %w", table, err)
		}
	}

	return nil
}

// ResetSequences resets all sequences to start from 1.
func (f *FixturesLoader) ResetSequences() error {
	f.t.Helper()

	ctx := context.Background()

	// Get all sequences
	query := `
		SELECT sequencename 
		FROM pg_sequences 
		WHERE schemaname = 'public'
	`

	rows, err := f.pool.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query sequences: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var seqName string
		if err := rows.Scan(&seqName); err != nil {
			return fmt.Errorf("failed to scan sequence name: %w", err)
		}

		resetQuery := fmt.Sprintf("ALTER SEQUENCE %s RESTART WITH 1", seqName)
		_, err := f.pool.Exec(ctx, resetQuery)
		if err != nil {
			return fmt.Errorf("failed to reset sequence %s: %w", seqName, err)
		}
	}

	return nil
}

// GetTableRowCount returns the number of rows in a table.
func (f *FixturesLoader) GetTableRowCount(table string) (int64, error) {
	f.t.Helper()

	ctx := context.Background()
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)

	var count int64
	err := f.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count rows in %s: %w", table, err)
	}

	return count, nil
}

// TableExists checks if a table exists.
func (f *FixturesLoader) TableExists(table string) (bool, error) {
	f.t.Helper()

	ctx := context.Background()
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = $1
		)
	`

	var exists bool
	err := f.pool.QueryRow(ctx, query, table).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if table exists: %w", err)
	}

	return exists, nil
}
