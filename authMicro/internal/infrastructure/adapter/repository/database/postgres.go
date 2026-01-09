package database

import (
	"errors"
	"fmt"
	"time"
	"unicode"

	"github.com/PavelShe11/studbridge/authMicro/internal/config"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func NewPostgresDB(cfg config.DBConfig) (*sqlx.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.DBName, cfg.SSLMode)

	db, err := sqlx.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	// Configure automatic camelCase <-> snake_case mapping
	db.Mapper = reflectx.NewMapperFunc("db", func(s string) string {
		return toSnakeCase(s)
	})

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func InitSchema(db *sqlx.DB) error {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to get instance driver: %w", err)
	}
	m, err := migrate.NewWithDatabaseInstance("file:///migrations", "pgx", driver)
	if err != nil {
		return fmt.Errorf("failed to init migrations: %w", err)
	}
	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	return nil
}

func toSnakeCase(s string) string {
	if s == "" {
		return s
	}

	var result []rune
	runes := []rune(s)

	for i := 0; i < len(runes); i++ {
		if i > 0 && unicode.IsUpper(runes[i]) {
			if unicode.IsLower(runes[i-1]) ||
				(i+1 < len(runes) && unicode.IsLower(runes[i+1])) {
				result = append(result, '_')
			}
		}
		result = append(result, unicode.ToLower(runes[i]))
	}

	return string(result)
}
