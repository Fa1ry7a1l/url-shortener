package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "github.com/lib/pq"
)

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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &PostgresStore{db: db}, nil
}

func (p *PostgresStore) Save(ctx context.Context, id string, original string) error {
	const query = `
INSERT INTO short_urls (id, original_url)
VALUES ($1, $2)
`
	_, err := p.db.ExecContext(ctx, query, id, original)
	return err
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
