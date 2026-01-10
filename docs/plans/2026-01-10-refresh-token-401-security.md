# Refresh Token 401 Security Design

**Created:** 2026-01-10
**Status:** Approved
**Security Level:** Maximum (OWASP-compliant)

## Overview

Implement maximum information security for `/auth/v1/refresh` endpoint by returning `401 Unauthorized` with empty body for ANY refresh token error, preventing information disclosure to attackers.

## Problem

Currently, the refresh token endpoint returns detailed error messages that leak information:
- `invalidRefreshToken` - reveals JWT signature is invalid
- `refreshTokenExpired` - reveals token existed but expired
- Same error used for "invalid JWT" and "not found in DB"

This allows attackers to:
- Enumerate valid vs invalid tokens
- Distinguish between different failure reasons
- Conduct timing attacks based on different code paths

## Solution

### Security Approach

**Chosen:** Ideal variant - return `401 Unauthorized` for ALL authentication failures
- ✅ No information leakage
- ✅ OWASP-compliant
- ✅ Consistent response regardless of failure reason
- ✅ Detailed logs preserved internally for debugging

### Scope

**Applies to:** `/auth/v1/refresh` endpoint only
**Unchanged:** `/auth/v1/login`, `/auth/v1/registration` (keep current behavior)

### HTTP Status Code Strategy

| Scenario | HTTP Status | Body |
|----------|-------------|------|
| Invalid JSON format | 400 Bad Request | `{"error": "..."}` |
| Missing refreshToken field | 400 Bad Request | `{"error": "refreshToken is required"}` |
| Invalid JWT signature | 401 Unauthorized | (empty) |
| Token not found in DB | 401 Unauthorized | (empty) |
| Token expired | 401 Unauthorized | (empty) |
| Internal server error | 500 Internal Server Error | (empty) |

**Rationale:**
- 400 - obvious request format issues (helps legitimate clients)
- 401 - all token authentication failures (prevents enumeration)
- 500 - infrastructure failures (DB down, etc.)

## Implementation

### Files Changed (3 files)

#### 1. `authMicro/internal/entity/error.go`

**Add new error function after line 54:**

```go
// NewUnauthorizedRefreshTokenError creates error when token is valid but not registered in DB
func NewUnauthorizedRefreshTokenError() *commonEntity.BaseValidationError {
	return &commonEntity.BaseValidationError{
		BaseError: commonEntity.BaseError{Code: "unauthorizedRefreshToken"},
		FieldErrors: []commonEntity.FieldError{{
			NameField: "refreshToken",
			Message:   "unauthorizedRefreshToken",
			Params:    nil,
		}},
	}
}
```

**Error semantics after change:**
- `invalidRefreshToken` - JWT signature invalid or malformed
- `unauthorizedRefreshToken` - JWT valid but not registered in DB (NEW)
- `refreshTokenExpired` - token expired by time

#### 2. `authMicro/internal/service/token.go`

**Change line 107:**

```go
// Before:
if session == nil {
    return nil, entity.NewInvalidRefreshTokenError()
}

// After:
if session == nil {
    return nil, entity.NewUnauthorizedRefreshTokenError()
}
```

**Service error flow after change:**
- Line 94-96: `NewInvalidRefreshTokenError()` - bad JWT signature
- Line 107: `NewUnauthorizedRefreshTokenError()` - token not found in DB
- Line 112: `NewRefreshTokenExpiredError()` - token expired

#### 3. `authMicro/internal/api/rest/handler/refresh_tokens.go`

**Replace lines 36-39:**

```go
// Before:
tokens, err := h.tokenService.RefreshTokens(c.Request().Context(), refreshToken)
if err != nil {
    return err
}

// After:
tokens, err := h.tokenService.RefreshTokens(c.Request().Context(), refreshToken)
if err != nil {
	h.logger.Debug("refresh token failed", "error", err)

	// Internal errors should return 500
	if _, ok := err.(*commonEntity.InternalError); ok {
		return c.NoContent(http.StatusInternalServerError)
	}

	// All token-related errors return 401 without body
	return c.NoContent(http.StatusUnauthorized)
}
```

**Handler behavior:**
- Lines 27-29: Invalid JSON → `400` with body
- Lines 32-34: Empty refreshToken → `400` with body
- Lines 36-47: Any service error → log + return `401`/`500` without body

## Error Flow

```
Client → POST /auth/v1/refresh
         ↓
Handler: Validate request format
         ├─ Invalid JSON? → 400 {"error": "..."}
         ├─ Empty refreshToken? → 400 {"error": "refreshToken is required"}
         └─ OK → pass to Service
                  ↓
Service: RefreshTokens(token)
         ├─ ParseToken failed? → NewInvalidRefreshTokenError()
         ├─ Session == nil? → NewUnauthorizedRefreshTokenError() [NEW]
         ├─ Expired? → NewRefreshTokenExpiredError()
         ├─ CreateTokens failed? → (propagate error)
         └─ DB error? → InternalError
                  ↓
Handler: Process service error
         ├─ InternalError? → Log + 500 (no body)
         └─ Any other error? → Log + 401 (no body)
```

## Security Benefits

✅ **Zero information leakage** - attacker cannot distinguish failure reasons
✅ **Anti-enumeration** - cannot determine if token ever existed
✅ **Timing attack resistant** - single code path for all token errors
✅ **OWASP compliant** - follows best practices for authentication errors
✅ **Audit trail preserved** - detailed errors logged for debugging
✅ **Service testability** - unit tests can still verify specific error types

## Migration Impact

### Breaking Changes

**For legitimate clients:**
- Before: Could distinguish error types from response body
- After: Must treat all 401 responses as "re-authenticate required"

**For monitoring/logging:**
- Before: Error details in response body
- After: Error details only in server logs (Debug level)

### Non-Breaking

- HTTP status codes remain in 4xx/5xx range
- Request format unchanged
- Success response (200) unchanged
- Other endpoints unaffected

## Testing Strategy

### Unit Tests

**Service layer (`token_test.go`):**
- Test `NewUnauthorizedRefreshTokenError()` returned when session == nil
- Verify distinction from `NewInvalidRefreshTokenError()` (bad JWT)

### E2E Tests

**Handler layer (`refresh_tokens_test.go`):**
- All service errors → verify `401` with empty body
- Internal errors → verify `500` with empty body
- Invalid JSON → verify `400` with body
- Empty refreshToken → verify `400` with body

### Manual Testing Scenarios

1. Invalid JSON → `400` with body
2. Empty refreshToken → `400` with body
3. Invalid JWT signature → `401` empty
4. Valid JWT not in DB → `401` empty
5. Expired token → `401` empty
6. DB connection failure → `500` empty

## Future Considerations

### Potential Enhancements

1. **Rate limiting** - add per-IP throttling for failed refresh attempts
2. **Token rotation detection** - detect and block token replay attacks
3. **Anomaly detection** - flag unusual patterns in refresh failures
4. **Metrics** - track 401 rate without exposing specifics

### Consistency Across Endpoints

Consider applying similar approach to:
- `/auth/v1/login` - 401 for any credential failure
- Protected endpoints - 401 for invalid access tokens

## References

- OWASP Authentication Cheat Sheet
- RFC 7235 - HTTP Authentication
- CWE-209 - Information Exposure Through an Error Message
