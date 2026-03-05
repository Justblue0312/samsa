package db

import (
	"database/sql"
	"embed"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

type Migrate struct {
	DB  *sql.DB
	dsn string
}

func Migrator(pool *pgxpool.Pool) *Migrate {
	m := &Migrate{
		DB: sql.OpenDB(stdlib.GetPoolConnector(pool)),
	}
	goose.SetBaseFS(embedMigrations)

	return m
}

func (m *Migrate) Up() error {
	if err := goose.Up(m.DB, "migrations"); err != nil {
		return err
	}
	return nil
}

func (m *Migrate) Down() error {
	if err := goose.Down(m.DB, "migrations"); err != nil {
		return err
	}
	return nil
}
