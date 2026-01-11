//go:build integration
// +build integration

package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
	"unicode"

	"github.com/PavelShe11/studbridge/authMicro/internal/entity"
	"github.com/PavelShe11/studbridge/authMicro/internal/port"
	trmsql "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	testDB        *sqlx.DB
	testContainer testcontainers.Container
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Setup PostgreSQL container
	req := testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:16-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "test",
				"POSTGRES_PASSWORD": "test",
				"POSTGRES_DB":       "test_db",
			},
			WaitingFor: wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	}

	container, err := testcontainers.GenericContainer(ctx, req)
	if err != nil {
		panic(fmt.Sprintf("failed to start container: %s", err))
	}

	testContainer = container

	// Get connection string
	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "5432")
	connStr := fmt.Sprintf("postgres://test:test@%s:%s/test_db?sslmode=disable", host, port.Port())

	// Connect to database
	testDB, err = sqlx.Connect("postgres", connStr)
	if err != nil {
		container.Terminate(ctx)
		panic(fmt.Sprintf("failed to connect to database: %s", err))
	}

	// Configure automatic camelCase <-> snake_case mapping
	testDB.Mapper = reflectx.NewMapperFunc("db", func(s string) string {
		return toSnakeCase(s)
	})

	// Run migrations
	runMigrations(testDB)

	// Run tests
	code := m.Run()

	// Cleanup
	testDB.Close()
	container.Terminate(ctx)

	os.Exit(code)
}

func runMigrations(db *sqlx.DB) {
	migration := `
	CREATE TABLE IF NOT EXISTS registration_session (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		code TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		code_expires TIMESTAMP NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_registration_session_email ON registration_session(email);
	CREATE INDEX IF NOT EXISTS idx_registration_session_code_expires ON registration_session(code_expires);
	`
	_, err := db.Exec(migration)
	if err != nil {
		panic(fmt.Sprintf("failed to run migrations: %s", err))
	}
}

// setupTest creates a fresh repository with clean database
func setupTest(t *testing.T) port.RegistrationSessionRepository {
	_, err := testDB.Exec("TRUNCATE TABLE registration_session")
	require.NoError(t, err)

	return NewRegistrationSessionRepository(testDB, trmsql.DefaultCtxGetter)
}

// TestSave_NewSession_Success - insert new session
func TestSave_NewSession_Success(t *testing.T) {
	t.Parallel()

	repo := setupTest(t)
	ctx := context.Background()

	session := &entity.RegistrationSession{
		Email:       "test@test.com",
		Code:        "hashed_123456",
		CodeExpires: time.Now().Add(2 * time.Minute),
	}

	err := repo.Save(ctx, session)

	assert.NoError(t, err)
	assert.NotEmpty(t, session.Id)
	assert.NotZero(t, session.CreatedAt)
}

// TestSave_ExistingSession_UpdatesRecord - UPSERT updates existing
func TestSave_ExistingSession_UpdatesRecord(t *testing.T) {
	t.Parallel()

	repo := setupTest(t)
	ctx := context.Background()

	session := &entity.RegistrationSession{
		Email:       "update@test.com",
		Code:        "initial_code",
		CodeExpires: time.Now().Add(2 * time.Minute),
	}
	repo.Save(ctx, session)

	session.Code = "updated_code"
	session.CodeExpires = time.Now().Add(5 * time.Minute)

	err := repo.Save(ctx, session)

	assert.NoError(t, err)

	found, _ := repo.FindByEmail(ctx, "update@test.com")
	assert.Equal(t, "updated_code", found.Code)
}

// TestFindByEmail_SessionExists_ReturnsSession
func TestFindByEmail_SessionExists_ReturnsSession(t *testing.T) {
	t.Parallel()

	repo := setupTest(t)
	ctx := context.Background()

	session := &entity.RegistrationSession{
		Email:       "find@test.com",
		Code:        "123456",
		CodeExpires: time.Now().Add(2 * time.Minute),
	}
	repo.Save(ctx, session)

	found, err := repo.FindByEmail(ctx, "find@test.com")

	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "find@test.com", found.Email)
	assert.Equal(t, "123456", found.Code)
	assert.NotEmpty(t, found.Id)
}

// TestFindByEmail_NotFound_ReturnsNil
func TestFindByEmail_NotFound_ReturnsNil(t *testing.T) {
	t.Parallel()

	repo := setupTest(t)
	ctx := context.Background()

	found, err := repo.FindByEmail(ctx, "notfound@test.com")

	assert.NoError(t, err)
	assert.Nil(t, found)
}

// toSnakeCase converts PascalCase to snake_case
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