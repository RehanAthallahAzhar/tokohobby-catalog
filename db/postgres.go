package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/models"
	_ "github.com/lib/pq" // PostgreSQL driver
)

func Connect(ctx context.Context, credential *models.Credential) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Jakarta",
		credential.Host,
		credential.Username,
		credential.Password,
		credential.DatabaseName,
		credential.Port,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	// Optional: test connection
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(100)
	db.SetConnMaxLifetime(time.Hour)

	log.Println("Database connection established successfully (sqlc).")
	return db, nil
}
