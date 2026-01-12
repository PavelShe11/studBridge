//go:build e2e
// +build e2e

package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
	"unicode"

	"github.com/PavelShe11/studbridge/authMicro/internal/api/rest/handler"
	"github.com/PavelShe11/studbridge/authMicro/internal/api/rest/models"
	"github.com/PavelShe11/studbridge/authMicro/internal/config"
	"github.com/PavelShe11/studbridge/authMicro/internal/infrastructure/adapter/repository"
	"github.com/PavelShe11/studbridge/authMicro/internal/service"
	"github.com/PavelShe11/studbridge/authMicro/test/helpers"
	"github.com/PavelShe11/studbridge/authMicro/test/mocks"

	trmsql "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

// toSnakeCase converts CamelCase to snake_case
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, unicode.ToLower(r))
		} else {
			result = append(result, r)
		}
	}
	return string(result)
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

// setupE2E creates full E2E stack with real components
func setupE2E(t *testing.T) (*handler.Register, *mocks.MockAccountProvider, *echo.Echo) {
	// Clean database before each test
	_, err := testDB.Exec("TRUNCATE TABLE registration_session")
	require.NoError(t, err)

	// Real repository
	repo := repository.NewRegistrationSessionRepository(testDB, trmsql.DefaultCtxGetter)

	// Mock only external gRPC (AccountProvider is external service)
	mockAccountProvider := new(mocks.MockAccountProvider)

	// Real service with real repository
	testLogger := helpers.NewNoopLogger()
	cfg := config.CodeGenConfig{
		CodePattern:   "[0-9]{6}",
		CodeMaxLength: 6,
		CodeTTL:       2 * time.Minute,
	}
	realService := service.NewRegistrationService(repo, mockAccountProvider, testLogger, cfg)

	// Real handler (translator not used in tested methods, pass nil)
	h := handler.NewRegisterHandler(testLogger, realService, nil)

	e := echo.New()

	return h, mockAccountProvider, e
}

// TestSendRegistrationCode_E2E_ValidRequest - Full E2E test with real database
func TestSendRegistrationCode_E2E_ValidRequest(t *testing.T) {
	h, mockAccountProvider, e := setupE2E(t)

	requestBody := map[string]any{
		"email":     "test@test.com",
		"firstName": "John",
		"lastName":  "Doe",
		"password":  "SecurePass123",
	}

	// Mock only external gRPC calls (not in our control)
	mockAccountProvider.On("ValidateAccountData", mock.Anything, requestBody, "en").Return(nil)
	mockAccountProvider.On("GetAccountByEmail", mock.Anything, "test@test.com").Return(nil, nil)

	// Make HTTP request
	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/registration", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute handler (goes through real service -> real repository -> real DB)
	err := h.SendRegistrationCode(c)

	// Assert HTTP response
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response models.RegistrationResponse
	json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NotEmpty(t, response.CodeExpires)
	assert.Equal(t, "[0-9]{6}", response.CodePattern)

	// Verify session was ACTUALLY saved in real database
	var count int
	err = testDB.Get(&count, "SELECT COUNT(*) FROM registration_session WHERE email = $1", "test@test.com")
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "Session should be saved in database")

	// Verify code is hashed (not empty)
	var code string
	err = testDB.Get(&code, "SELECT code FROM registration_session WHERE email = $1", "test@test.com")
	assert.NoError(t, err)
	assert.NotEmpty(t, code, "Code should be hashed and stored")

	mockAccountProvider.AssertExpectations(t)
}

// TestSendRegistrationCode_E2E_ExistingSession - Test session update
func TestSendRegistrationCode_E2E_ExistingSession(t *testing.T) {
	h, mockAccountProvider, e := setupE2E(t)

	requestBody := map[string]any{
		"email":     "existing@test.com",
		"firstName": "Jane",
		"lastName":  "Smith",
		"password":  "SecurePass456",
	}

	// Mock external gRPC calls for both requests
	mockAccountProvider.On("ValidateAccountData", mock.Anything, requestBody, "en").Return(nil).Times(2)
	mockAccountProvider.On("GetAccountByEmail", mock.Anything, "existing@test.com").Return(nil, nil).Times(2)

	// First request - creates session
	jsonBody, _ := json.Marshal(requestBody)
	req1 := httptest.NewRequest(http.MethodPost, "/registration", bytes.NewReader(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Accept-Language", "en")
	rec1 := httptest.NewRecorder()
	c1 := e.NewContext(req1, rec1)
	h.SendRegistrationCode(c1)

	// Get first code
	var firstCode string
	testDB.Get(&firstCode, "SELECT code FROM registration_session WHERE email = $1", "existing@test.com")

	// Second request - updates session with expired code
	time.Sleep(100 * time.Millisecond) // Ensure different timestamp
	jsonBody2, _ := json.Marshal(requestBody)
	req2 := httptest.NewRequest(http.MethodPost, "/registration", bytes.NewReader(jsonBody2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Accept-Language", "en")
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	h.SendRegistrationCode(c2)

	// Verify only ONE session exists (upserted, not duplicated)
	var count int
	err := testDB.Get(&count, "SELECT COUNT(*) FROM registration_session WHERE email = $1", "existing@test.com")
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "Should have exactly one session (upserted)")

	// Verify code was NOT updated (same code reused if not expired)
	var secondCode string
	testDB.Get(&secondCode, "SELECT code FROM registration_session WHERE email = $1", "existing@test.com")
	assert.Equal(t, firstCode, secondCode, "Code should remain the same if not expired")

	mockAccountProvider.AssertExpectations(t)
}

// TestSendRegistrationCode_E2E_DatabasePersistence - Verify data survives across requests
func TestSendRegistrationCode_E2E_DatabasePersistence(t *testing.T) {
	h, mockAccountProvider, e := setupE2E(t)

	// Create multiple sessions
	emails := []string{"user1@test.com", "user2@test.com", "user3@test.com"}
	for _, email := range emails {
		requestBody := map[string]any{
			"email":     email,
			"firstName": "Test",
			"lastName":  "User",
			"password":  "SecurePass789",
		}

		mockAccountProvider.On("ValidateAccountData", mock.Anything, requestBody, "en").Return(nil).Once()
		mockAccountProvider.On("GetAccountByEmail", mock.Anything, email).Return(nil, nil).Once()

		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http.MethodPost, "/registration", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept-Language", "en")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		h.SendRegistrationCode(c)
	}

	// Verify all sessions persist in database
	var count int
	err := testDB.Get(&count, "SELECT COUNT(*) FROM registration_session")
	assert.NoError(t, err)
	assert.Equal(t, 3, count, "All 3 sessions should persist in database")

	// Verify each session has required fields
	type Session struct {
		Id          string    `db:"id"`
		Code        string    `db:"code"`
		Email       string    `db:"email"`
		CodeExpires time.Time `db:"code_expires"`
		CreatedAt   time.Time `db:"created_at"`
	}
	var sessions []Session
	err = testDB.Select(&sessions, "SELECT * FROM registration_session ORDER BY email")
	assert.NoError(t, err)
	assert.Len(t, sessions, 3)

	for i, session := range sessions {
		assert.NotEmpty(t, session.Id, "Session %d should have ID", i)
		assert.NotEmpty(t, session.Code, "Session %d should have hashed code", i)
		assert.Equal(t, emails[i], session.Email, "Session %d should have correct email", i)
		assert.True(t, session.CodeExpires.After(time.Now()), "Session %d code should not be expired", i)
		assert.True(t, session.CreatedAt.Before(time.Now().Add(time.Second)), "Session %d should have valid created_at", i)
	}

	mockAccountProvider.AssertExpectations(t)
}