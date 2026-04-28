package repository

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewDBConnection creates a PostgreSQL connection pool with retry logic
func NewDBConnection() (*pgxpool.Pool, error) {
	ctx := context.Background()

	maxRetries := 5
	for attempt := 1; attempt <= maxRetries; attempt++ {
		pool, err := createPool(ctx)
		if err == nil {
			return pool, nil
		}
		if attempt == maxRetries {
			return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
		}
		delay := time.Duration(attempt*2) * time.Second
		fmt.Printf("Database connection attempt %d/%d failed: %v. Retrying in %v...\n", attempt, maxRetries, err, delay)
		time.Sleep(delay)
	}
	return nil, fmt.Errorf("unable to connect to database")
}

func createPool(ctx context.Context) (*pgxpool.Pool, error) {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "devuser"
	}
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "devuser123"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "retailpos"
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, dbname)

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Connection pool settings
	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}