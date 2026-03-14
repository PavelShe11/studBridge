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

	"github.com/PavelShe11/studbridge/authMicro/internal/config"
	"github.com/PavelShe11/studbridge/authMicro/internal/entity"
	"github.com/PavelShe11/studbridge/authMicro/internal/infrastructure/inbound/rest/handler"
	"github.com/PavelShe11/studbridge/authMicro/internal/infrastructure/inbound/rest/models"
	"github.com/PavelShe11/studbridge/authMicro/internal/infrastructure/outbound/repository"
	"github.com/PavelShe11/studbridge/authMicro/internal/service"
	"github.com/PavelShe11/studbridge/authMicro/test/helpers"
	"github.com/PavelShe11/studbridge/authMicro/test/mocks"
	"github.com/PavelShe11/studbridge/authMicro/utlis/hash"

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
	mockEmailSender := new(mocks.MockEmailSender)
	mockEmailSender.On("SendVerificationCode", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	realService := service.NewRegistrationService(repo, mockAccountProvider, mockEmailSender, testLogger, cfg)

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

// TestSendRegistrationCode_E2E_HTTPStatuses - table-driven test for HTTP status codes
func TestSendRegistrationCode_E2E_HTTPStatuses(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		setupMock      func(*mocks.MockAccountProvider)
		expectedStatus int
		checkDB        func(*testing.T)
	}{
		{
			name:           "invalid JSON returns 400",
			requestBody:    `{invalid json}`,
			setupMock:      func(m *mocks.MockAccountProvider) {},
			expectedStatus: http.StatusBadRequest,
			checkDB: func(t *testing.T) {
				var count int
				testDB.Get(&count, "SELECT COUNT(*) FROM registration_session")
				assert.Equal(t, 0, count, "No session should be created on invalid JSON")
			},
		},
		{
			name:        "valid request with all fields succeeds",
			requestBody: `{"email":"status@test.com","firstName":"Test","lastName":"User","password":"SecurePass123"}`,
			setupMock: func(m *mocks.MockAccountProvider) {
				m.On("ValidateAccountData", mock.Anything, mock.Anything, "en").Return(nil)
				m.On("GetAccountByEmail", mock.Anything, "status@test.com").Return(nil, nil)
			},
			expectedStatus: http.StatusOK,
			checkDB: func(t *testing.T) {
				var count int
				testDB.Get(&count, "SELECT COUNT(*) FROM registration_session WHERE email = $1", "status@test.com")
				assert.Equal(t, 1, count, "Session should be created for valid request")
			},
		},
		{
			name:        "existing account returns 200 with empty code (security)",
			requestBody: `{"email":"existingacc@test.com","firstName":"Test","lastName":"User","password":"SecurePass123"}`,
			setupMock: func(m *mocks.MockAccountProvider) {
				m.On("ValidateAccountData", mock.Anything, mock.Anything, "en").Return(nil)
				m.On("GetAccountByEmail", mock.Anything, "existingacc@test.com").Return(&entity.Account{AccountId: "123"}, nil)
			},
			expectedStatus: http.StatusOK,
			checkDB: func(t *testing.T) {
				// Session is created with empty code (for security - not revealing email exists)
				var count int
				testDB.Get(&count, "SELECT COUNT(*) FROM registration_session WHERE email = $1", "existingacc@test.com")
				assert.Equal(t, 1, count, "Session should be created (with empty code)")

				var code string
				testDB.Get(&code, "SELECT code FROM registration_session WHERE email = $1", "existingacc@test.com")
				assert.Empty(t, code, "Code should be empty for existing account")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockAccountProvider, e := setupE2E(t)
			tt.setupMock(mockAccountProvider)

			req := httptest.NewRequest(http.MethodPost, "/registration", bytes.NewReader([]byte(tt.requestBody)))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept-Language", "en")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := h.SendRegistrationCode(c)

			if tt.expectedStatus >= 400 {
				// Error cases - check that error was returned or written to response
				if err != nil {
					httpErr, ok := err.(*echo.HTTPError)
					if ok {
						assert.Equal(t, tt.expectedStatus, httpErr.Code)
					}
				} else {
					// Error might be written to response directly
					assert.Equal(t, tt.expectedStatus, rec.Code)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)
			}

			if tt.checkDB != nil {
				tt.checkDB(t)
			}

			mockAccountProvider.AssertExpectations(t)
		})
	}
}

// TestRegistrationConfirmEmail_E2E_Success - Full E2E confirmation test
func TestRegistrationConfirmEmail_E2E_Success(t *testing.T) {
	h, mockAccountProvider, e := setupE2E(t)

	// Insert a session directly with a known code
	knownCode := "123456"
	hashedCode := hash.MustHashCode(knownCode)
	codeExpires := time.Now().Add(5 * time.Minute)

	_, err := testDB.Exec(`
		INSERT INTO registration_session (email, code, code_expires)
		VALUES ($1, $2, $3)
	`, "confirm@test.com", hashedCode, codeExpires)
	require.NoError(t, err)

	requestBody := map[string]any{
		"email":     "confirm@test.com",
		"firstName": "Test",
		"lastName":  "User",
		"password":  "SecurePass123",
		"code":      knownCode,
	}

	// Mock external gRPC calls
	mockAccountProvider.On("ValidateAccountData", mock.Anything, requestBody, "en").Return(nil)
	mockAccountProvider.On("CreateAccount", mock.Anything, requestBody, "en").Return(nil)

	// Make HTTP request
	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/registration/confirmEmail", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute handler
	err = h.RegistrationConfirmEmail(c)

	// Assert HTTP response
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify session was DELETED from database (cleanup after successful registration)
	var count int
	err = testDB.Get(&count, "SELECT COUNT(*) FROM registration_session WHERE email = $1", "confirm@test.com")
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "Session should be deleted after successful confirmation")

	mockAccountProvider.AssertExpectations(t)
}

// TestRegistrationConfirmEmail_E2E_InvalidCode - Wrong code returns error
func TestRegistrationConfirmEmail_E2E_InvalidCode(t *testing.T) {
	h, mockAccountProvider, e := setupE2E(t)

	// Insert a session with a known code
	knownCode := "123456"
	hashedCode := hash.MustHashCode(knownCode)
	codeExpires := time.Now().Add(5 * time.Minute)

	_, err := testDB.Exec(`
		INSERT INTO registration_session (email, code, code_expires)
		VALUES ($1, $2, $3)
	`, "wrongcode@test.com", hashedCode, codeExpires)
	require.NoError(t, err)

	requestBody := map[string]any{
		"email":     "wrongcode@test.com",
		"firstName": "Test",
		"lastName":  "User",
		"password":  "SecurePass123",
		"code":      "000000", // Wrong code!
	}

	// Mock external gRPC calls (ValidateAccountData is called first)
	mockAccountProvider.On("ValidateAccountData", mock.Anything, requestBody, "en").Return(nil)

	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/registration/confirmEmail", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = h.RegistrationConfirmEmail(c)

	// Should return error or error status
	if err != nil {
		httpErr, ok := err.(*echo.HTTPError)
		if ok {
			assert.Equal(t, http.StatusBadRequest, httpErr.Code)
		}
	} else {
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	}

	// Verify session was NOT deleted (confirmation failed)
	var count int
	err = testDB.Get(&count, "SELECT COUNT(*) FROM registration_session WHERE email = $1", "wrongcode@test.com")
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "Session should NOT be deleted after failed confirmation")

	mockAccountProvider.AssertExpectations(t)
}

// TestRegistrationConfirmEmail_E2E_ExpiredCode - Expired code returns error
func TestRegistrationConfirmEmail_E2E_ExpiredCode(t *testing.T) {
	h, mockAccountProvider, e := setupE2E(t)

	// Insert a session with EXPIRED code (use UTC for consistent comparison)
	knownCode := "123456"
	hashedCode := hash.MustHashCode(knownCode)
	codeExpires := time.Now().UTC().Add(-10 * time.Minute) // Expired (well in the past)

	_, err := testDB.Exec(`
		INSERT INTO registration_session (email, code, code_expires)
		VALUES ($1, $2, $3)
	`, "expired@test.com", hashedCode, codeExpires)
	require.NoError(t, err)

	requestBody := map[string]any{
		"email":     "expired@test.com",
		"firstName": "Test",
		"lastName":  "User",
		"password":  "SecurePass123",
		"code":      knownCode, // Correct code but expired
	}

	mockAccountProvider.On("ValidateAccountData", mock.Anything, requestBody, "en").Return(nil)

	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/registration/confirmEmail", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = h.RegistrationConfirmEmail(c)

	// Should return error or error status
	if err != nil {
		httpErr, ok := err.(*echo.HTTPError)
		if ok {
			assert.Equal(t, http.StatusBadRequest, httpErr.Code)
		}
	} else {
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	}

	mockAccountProvider.AssertExpectations(t)
}

// TestRegistrationConfirmEmail_E2E_SessionNotFound - No session returns error
func TestRegistrationConfirmEmail_E2E_SessionNotFound(t *testing.T) {
	h, mockAccountProvider, e := setupE2E(t)

	// No session inserted - email doesn't exist
	requestBody := map[string]any{
		"email":     "nosession@test.com",
		"firstName": "Test",
		"lastName":  "User",
		"password":  "SecurePass123",
		"code":      "123456",
	}

	mockAccountProvider.On("ValidateAccountData", mock.Anything, requestBody, "en").Return(nil)

	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/registration/confirmEmail", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.RegistrationConfirmEmail(c)

	// Should return error or error status
	if err != nil {
		httpErr, ok := err.(*echo.HTTPError)
		if ok {
			assert.Equal(t, http.StatusBadRequest, httpErr.Code)
		}
	} else {
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	}

	mockAccountProvider.AssertExpectations(t)
}
