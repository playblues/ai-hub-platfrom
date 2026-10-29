package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB defines the database interface for summary storage and health checking.
type DB interface {
	InitSchema(ctx context.Context) error
	SaveSummary(ctx context.Context, text, result, language string) (int64, error)
	Ping(ctx context.Context) error
	Close()
}

type postgresDB struct {
	pool *pgxpool.Pool
}

// NewPostgres creates a new PostgreSQL connection pool using pgx/v5.
func NewPostgres(ctx context.Context, dsn string) (DB, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres config: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 2
	config.MaxConnIdleTime = 5 * time.Minute
	config.MaxConnLifetime = 1 * time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres pool: %w", err)
	}

	return &postgresDB{pool: pool}, nil
}

// InitSchema creates the summaries table if it doesn't already exist.
func (p *postgresDB) InitSchema(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS summaries (
		id BIGSERIAL PRIMARY KEY,
		text TEXT NOT NULL,
		result TEXT NOT NULL,
		language VARCHAR(10) NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_summaries_created_at ON summaries(created_at DESC);
	`
	_, err := p.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create summaries schema: %w", err)
	}

	slog.Info("database schema initialized (summaries table ready)")
	return nil
}

// SaveSummary inserts a summary record into the database.
func (p *postgresDB) SaveSummary(ctx context.Context, text, result, language string) (int64, error) {
	query := `
		INSERT INTO summaries (text, result, language, created_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id;
	`
	var id int64
	err := p.pool.QueryRow(ctx, query, text, result, language).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to insert summary into database: %w", err)
	}
	return id, nil
}

// Ping checks if the database connection is alive.
func (p *postgresDB) Ping(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return p.pool.Ping(pingCtx)
}

// Close closes the database connection pool.
func (p *postgresDB) Close() {
	if p.pool != nil {
		p.pool.Close()
	}
}
