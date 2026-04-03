package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/lucciano/prds/pkg/config"
	"github.com/redis/go-redis/v9"
)

// DB wraps sql.DB with app-specific helpers.
type DB struct {
	*sql.DB
}

// Connect opens a PostgreSQL connection pool.
func Connect(cfg *config.Config) (*DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPass, cfg.DBName,
	)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	// Connection pool tuning
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &DB{db}, nil
}

// Migrate runs all pending database migrations.
func Migrate(dbURL string) error {
	m, err := migrate.New("file://migrations", dbURL)
	if err != nil {
		return fmt.Errorf("migrate.New: %w", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate.Up: %w", err)
	}
	return nil
}

// ConnectRedis returns a connected Redis client.
func ConnectRedis(cfg *config.Config) (*redis.Client, error) {
	opts, err := redis.ParseURL("redis://" + cfg.RedisURL)
	if err != nil {
		// Fallback for bare host:port
		opts = &redis.Options{Addr: cfg.RedisURL}
	}
	client := redis.NewClient(opts)
	return client, nil
}
