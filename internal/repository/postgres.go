package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	postgresMigrate "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgerrcode"
	"github.com/lib/pq"
)

const migrationsPath = "file://migrations"

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(dsn string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := runMigrations(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &PostgresStore{db: db}, nil
}

func runMigrations(db *sql.DB) error {
	driver, err := postgresMigrate.WithInstance(db, &postgresMigrate.Config{})
	if err != nil {
		return fmt.Errorf("create migrate driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(migrationsPath, "postgres", driver)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}

func (p *PostgresStore) Save(ctx context.Context, id string, original string) error {
	const query = `
INSERT INTO short_urls (id, original_url)
VALUES ($1, $2)
`
	_, err := p.db.ExecContext(ctx, query, id, original)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && string(pqErr.Code) == pgerrcode.UniqueViolation {
			switch pqErr.Constraint {
			case "short_urls_pkey":
				return ErrIDExists
			case "short_urls_original_url_uq":
				return ErrOriginalURLExist
			default:
				return err
			}
		}
		return err
	}

	return nil
}

func (p *PostgresStore) SaveBatch(ctx context.Context, items []BatchItem) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var (
		valueParts []string
		args       []any
	)

	for i, item := range items {
		n := i*2 + 1
		valueParts = append(valueParts, fmt.Sprintf("($%d, $%d)", n, n+1))
		args = append(args, item.ID, item.Original)
	}

	query := `
INSERT INTO short_urls (id, original_url)
VALUES ` + strings.Join(valueParts, ",")

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && string(pqErr.Code) == pgerrcode.UniqueViolation {
			switch pqErr.Constraint {
			case "short_urls_pkey":
				return ErrIDExists
			case "short_urls_original_url_uq":
				return ErrOriginalURLExist
			default:
				return err
			}
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (p *PostgresStore) Get(ctx context.Context, id string) (string, error) {
	const query = `
SELECT original_url
FROM short_urls
WHERE id = $1
`
	var original string

	err := p.db.QueryRowContext(ctx, query, id).Scan(&original)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}

	return original, nil
}

func (p *PostgresStore) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

func (p *PostgresStore) Close() error {
	return p.db.Close()
}

func (p *PostgresStore) GetByOriginal(ctx context.Context, original string) (string, error) {
	const query = `
SELECT id
FROM short_urls
WHERE original_url = $1
`
	var id string

	err := p.db.QueryRowContext(ctx, query, original).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}

	return id, nil
}
