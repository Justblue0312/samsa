package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justblue/samsa/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func New(ctx context.Context, c *config.Config) (*pgxpool.Pool, error) {
	databaseURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Postgres.User,
		c.Postgres.Pwd,
		c.Postgres.Host,
		c.Postgres.Port,
		c.Postgres.Database,
		c.Postgres.SslMode,
	)

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	config.MaxConnIdleTime = c.Postgres.MaxConnIdleTime
	config.MaxConns = c.Postgres.MaxConns
	config.MaxConnLifetime = c.Postgres.MaxConnLifetime

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	return pool, nil
}

func NewTestDatabase(ctx context.Context, c *config.Config) (*pgxpool.Pool, error) {
	databaseURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Postgres.User,
		c.Postgres.Pwd,
		c.Postgres.Host,
		c.Postgres.Port,
		c.Postgres.TestDatabase,
		c.Postgres.SslMode,
	)

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	config.MaxConnIdleTime = c.Postgres.MaxConnIdleTime
	config.MaxConns = c.Postgres.MaxConns
	config.MaxConnLifetime = c.Postgres.MaxConnLifetime

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	return pool, nil
}

// queryTracer implements pgx.QueryTracer for logging SQL queries in development.
type queryTracer struct{}

type queryContextKey struct{}

type queryData struct {
	sql       string
	args      []any
	startTime time.Time
}

// TraceQueryStart logs the start of a query.
func (t *queryTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, queryContextKey{}, &queryData{
		sql:       data.SQL,
		args:      data.Args,
		startTime: time.Now(),
	})
}

// TraceQueryEnd logs the end of a query with duration.
func (t *queryTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	qd, ok := ctx.Value(queryContextKey{}).(*queryData)
	if !ok {
		return
	}

	duration := time.Since(qd.startTime)

	// Log differently based on whether it succeeded or failed
	if data.Err != nil {
		slog.Error("database query failed",
			slog.String("sql", truncateSQL(qd.sql, 200)),
			slog.Duration("duration", duration),
			slog.Any("error", data.Err),
		)
	} else {
		// Only log slow queries (>100ms) at Info level, others at Debug
		level := slog.LevelDebug
		if duration > 100*time.Millisecond {
			level = slog.LevelWarn
		}

		slog.Log(ctx, level, "database query",
			slog.String("sql", truncateSQL(qd.sql, 200)),
			slog.Duration("duration", duration),
			slog.Int64("rows_affected", data.CommandTag.RowsAffected()),
		)
	}
}

// truncateSQL truncates SQL for logging to avoid huge log entries.
func truncateSQL(sql string, maxLen int) string {
	if len(sql) <= maxLen {
		return sql
	}
	return sql[:maxLen] + "..."
}
