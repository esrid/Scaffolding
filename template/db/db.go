package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"time"
	"{{projectName}}/utils"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

//go:embed migration/*.sql
var migrationFS embed.FS

func NewDB(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close() // Close the connection on ping failure to avoid resource leak
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func MigrateSchema(db *sql.DB, logger *slog.Logger) error {
	goose.SetBaseFS(migrationFS)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	logger.Info("running database migrations")
	if err := goose.Up(db, "migration"); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	logger.Info("migrations applied successfully")
	return nil
}

func CreateSeed(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	adminEmail := os.Getenv("ADMIN")
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminEmail == "" || adminPassword == "" {
		return fmt.Errorf("missing required environment variables for seeding admin user")
	}

	hashedPassword, err := utils.HashPassword(adminPassword)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}

	query := `
        INSERT INTO users (email, password_hash, role)
        VALUES ($1, $2, 'admin'::role)
        ON CONFLICT (email) DO NOTHING;`
	if _, err := db.ExecContext(ctx, query, adminEmail, hashedPassword); err != nil {
		return fmt.Errorf("failed to seed admin user: %w", err)
	}

	return nil
}
