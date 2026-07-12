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

// PostgresStore persists short URLs in PostgreSQL.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore connects to PostgreSQL, checks connectivity, and applies migrations.
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

// Save stores an ID and original URL without a user binding.
func (p *PostgresStore) Save(ctx context.Context, id string, original string) error {
	return p.SaveForUser(ctx, id, original, "")
}

// SaveForUser stores an ID and original URL for a user.
func (p *PostgresStore) SaveForUser(ctx context.Context, id string, original string, userID string) error {
	const query = `
INSERT INTO short_urls (id, original_url, user_id, is_deleted)
VALUES ($1, $2, $3, FALSE)
`
	_, err := p.db.ExecContext(ctx, query, id, original, userID)
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

// SaveBatch stores multiple URLs in one transaction.
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
		n := i*3 + 1
		valueParts = append(valueParts, fmt.Sprintf("($%d, $%d, $%d, FALSE)", n, n+1, n+2))
		args = append(args, item.ID, item.Original, item.UserID)
	}

	query := `
INSERT INTO short_urls (id, original_url, user_id, is_deleted)
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

// Get returns the original URL for id.
func (p *PostgresStore) Get(ctx context.Context, id string) (string, error) {
	const query = `
SELECT original_url, is_deleted
FROM short_urls
WHERE id = $1
`
	var (
		original  string
		isDeleted bool
	)

	err := p.db.QueryRowContext(ctx, query, id).Scan(&original, &isDeleted)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if isDeleted {
		return "", ErrDeleted
	}

	return original, nil
}

// DeleteBatchByUser marks user-owned IDs as deleted.
func (p *PostgresStore) DeleteBatchByUser(ctx context.Context, userID string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	const query = `
UPDATE short_urls
SET is_deleted = TRUE
WHERE user_id = $1
  AND id = ANY($2)
`

	_, err := p.db.ExecContext(ctx, query, userID, pq.Array(ids))
	return err
}

// Ping checks that PostgreSQL is reachable.
func (p *PostgresStore) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

// Close closes the underlying database connection pool.
func (p *PostgresStore) Close() error {
	return p.db.Close()
}

// GetByOriginal returns the ID for an already stored original URL.
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

// GetByUser returns non-deleted URLs owned by userID.
func (p *PostgresStore) GetByUser(ctx context.Context, userID string) ([]UserURL, error) {
	const query = `
SELECT id, original_url
FROM short_urls
WHERE user_id = $1
  AND is_deleted = FALSE
`

	rows, err := p.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]UserURL, 0)
	for rows.Next() {
		var item UserURL
		if err := rows.Scan(&item.ID, &item.Original); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// Stats returns service-wide counters.
func (p *PostgresStore) Stats(ctx context.Context) (Stats, error) {
	const query = `
SELECT COUNT(*), COUNT(DISTINCT NULLIF(user_id, ''))
FROM short_urls
`

	var urls, users int64
	if err := p.db.QueryRowContext(ctx, query).Scan(&urls, &users); err != nil {
		return Stats{}, err
	}

	return Stats{
		URLs:  int(urls),
		Users: int(users),
	}, nil
}
