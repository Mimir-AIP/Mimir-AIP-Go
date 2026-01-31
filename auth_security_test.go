package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuthManager_HashPassword tests password hashing with bcrypt
func TestAuthManager_HashPassword(t *testing.T) {
	am := utils.NewAuthManager("test-secret", 24*time.Hour, 100)

	// Test basic hashing
	password := "testpassword123"
	hash1, err := am.HashPassword(password)
	require.NoError(t, err, "HashPassword should not error")
	hash2, err := am.HashPassword(password)
	require.NoError(t, err, "HashPassword should not error")

	// With bcrypt, same password produces DIFFERENT hashes (due to salt)
	// but both should verify correctly
	assert.NotEmpty(t, hash1, "Hash should not be empty")
	assert.NotEmpty(t, hash2, "Hash should not be empty")
	assert.NotEqual(t, hash1, hash2, "bcrypt produces different hashes due to salt")

	// Both should verify correctly
	assert.True(t, am.VerifyPassword(password, hash1), "First hash should verify")
	assert.True(t, am.VerifyPassword(password, hash2), "Second hash should verify")

	// Different passwords should produce different hashes (usually)
	hash3, err := am.HashPassword("differentpassword")
	require.NoError(t, err)
	assert.NotEqual(t, hash1, hash3, "Different passwords should produce different hashes")

	// Verify bcrypt format (starts with $2a$)
	assert.True(t, strings.HasPrefix(hash1, "$2"), "bcrypt hash should start with $2")
}

// TestAuthManager_VerifyPassword tests password verification with bcrypt
func TestAuthManager_VerifyPassword(t *testing.T) {
	am := utils.NewAuthManager("test-secret", 24*time.Hour, 100)

	password := "testpassword123"
	hash, err := am.HashPassword(password)
	require.NoError(t, err)

	// Correct password should verify
	assert.True(t, am.VerifyPassword(password, hash), "Correct password should verify")

	// Incorrect password should not verify
	assert.False(t, am.VerifyPassword("wrongpassword", hash), "Incorrect password should not verify")

	// Empty password should not verify
	assert.False(t, am.VerifyPassword("", hash), "Empty password should not verify")

	// Different hash of same password should still verify
	hash2, _ := am.HashPassword(password)
	assert.True(t, am.VerifyPassword(password, hash2), "Different bcrypt hash of same password should verify")
}

// TestAuthManager_CreateUser tests user creation
func TestAuthManager_CreateUser(t *testing.T) {
	am := utils.NewAuthManager("test-secret", 24*time.Hour, 100)

	// Create user
	user, err := am.CreateUser("testuser", "password123", []string{"admin", "user"})
	require.NoError(t, err, "Failed to create user")
	assert.NotEmpty(t, user.ID, "User ID should not be empty")
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, []string{"admin", "user"}, user.Roles)
	assert.True(t, user.Active, "User should be active")
	assert.NotEqual(t, "password123", user.Password, "Password should be hashed")

	// Try to create duplicate user
	_, err = am.CreateUser("testuser", "anotherpassword", []string{"user"})
	assert.Error(t, err, "Should error when creating duplicate user")
	assert.Contains(t, err.Error(), "already exists")
}

// TestAuthManager_AuthenticateUser tests user authentication
func TestAuthManager_AuthenticateUser(t *testing.T) {
	am := utils.NewAuthManager("test-secret", 24*time.Hour, 100)

	// Create user
	_, err := am.CreateUser("testuser", "password123", []string{"user"})
	require.NoError(t, err)

	// Authenticate with correct credentials
	user, err := am.AuthenticateUser("testuser", "password123")
	require.NoError(t, err, "Failed to authenticate user")
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, []string{"user"}, user.Roles)

	// Authenticate with incorrect password
	_, err = am.AuthenticateUser("testuser", "wrongpassword")
	assert.Error(t, err, "Should error with incorrect password")
	assert.Contains(t, err.Error(), "invalid password")

	// Authenticate non-existent user
	_, err = am.AuthenticateUser("nonexistent", "password123")
	assert.Error(t, err, "Should error for non-existent user")
	assert.Contains(t, err.Error(), "not found")

	// Create inactive user and try to authenticate
	inactiveUser, err := am.CreateUser("inactiveuser", "password123", []string{"user"})
	require.NoError(t, err)
	inactiveUser.Active = false

	// Note: Since users are stored in a map by reference, this actually modifies the stored user
	_, err = am.AuthenticateUser("inactiveuser", "password123")
	assert.Error(t, err, "Should error for inactive user")
	assert.Contains(t, err.Error(), "inactive")
}

// TestAuthManager_GenerateJWT tests JWT token generation
func TestAuthManager_GenerateJWT(t *testing.T) {
	am := utils.NewAuthManager("test-secret-key-for-jwt-signing", 24*time.Hour, 100)

	user := &utils.User{
		ID:       "user-001",
		Username: "testuser",
		Roles:    []string{"admin", "user"},
	}

	// Generate token
	token, err := am.GenerateJWT(user)
	require.NoError(t, err, "Failed to generate JWT")
	assert.NotEmpty(t, token, "Token should not be empty")

	// Verify token structure (should have 3 parts separated by dots)
	parts := strings.Split(token, ".")
	assert.Len(t, parts, 3, "JWT should have 3 parts")
}

// TestAuthManager_ValidateJWT tests JWT token validation
func TestAuthManager_ValidateJWT(t *testing.T) {
	secret := "test-secret-key-for-jwt-signing"
	am := utils.NewAuthManager(secret, 24*time.Hour, 100)

	user := &utils.User{
		ID:       "user-001",
		Username: "testuser",
		Roles:    []string{"admin"},
	}

	// Generate and validate token
	token, err := am.GenerateJWT(user)
	require.NoError(t, err)

	claims, err := am.ValidateJWT(token)
	require.NoError(t, err, "Failed to validate JWT")
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.Username, claims.Username)
	assert.Equal(t, user.Roles, claims.Roles)
	assert.Equal(t, "mimir-aip", claims.Issuer)
	assert.Equal(t, user.ID, claims.Subject)

	// Validate invalid token
	_, err = am.ValidateJWT("invalid.token.here")
	assert.Error(t, err, "Should error for invalid token")

	// Validate token with wrong secret
	wrongAM := utils.NewAuthManager("wrong-secret", 24*time.Hour, 100)
	_, err = wrongAM.ValidateJWT(token)
	assert.Error(t, err, "Should error when validating with wrong secret")
}

// TestAuthManager_ValidateJWT_Expiration tests token expiration
func TestAuthManager_ValidateJWT_Expiration(t *testing.T) {
	secret := "test-secret-key-for-jwt-signing"
	// Create auth manager with very short expiry
	am := utils.NewAuthManager(secret, 1*time.Millisecond, 100)

	user := &utils.User{
		ID:       "user-001",
		Username: "testuser",
		Roles:    []string{"user"},
	}

	// Generate token
	token, err := am.GenerateJWT(user)
	require.NoError(t, err)

	// Wait for token to expire
	time.Sleep(50 * time.Millisecond)

	// Try to validate expired token
	_, err = am.ValidateJWT(token)
	assert.Error(t, err, "Should error for expired token")
}

// TestAuthManager_CreateAPIKey tests API key creation
func TestAuthManager_CreateAPIKey(t *testing.T) {
	am := utils.NewAuthManager("test-secret", 24*time.Hour, 100)

	// Create API key
	apiKey, err := am.CreateAPIKey("user-001", "Test Key")
	require.NoError(t, err, "Failed to create API key")
	assert.NotEmpty(t, apiKey.Key, "API key should not be empty")
	assert.Equal(t, "user-001", apiKey.UserID)
	assert.Equal(t, "Test Key", apiKey.Name)
	assert.True(t, apiKey.Active, "API key should be active")
	assert.False(t, apiKey.Created.IsZero(), "Created time should be set")

	// Verify key format (base64 URL encoding)
	assert.NotContains(t, apiKey.Key, "+", "Key should not contain + (base64 URL safe)")
	assert.NotContains(t, apiKey.Key, "/", "Key should not contain / (base64 URL safe)")
}

// TestAuthManager_ValidateAPIKey tests API key validation
func TestAuthManager_ValidateAPIKey(t *testing.T) {
	am := utils.NewAuthManager("test-secret", 24*time.Hour, 100)

	// Create API key
	apiKey, err := am.CreateAPIKey("user-001", "Test Key")
	require.NoError(t, err)

	// Validate correct key
	validatedKey, err := am.ValidateAPIKey(apiKey.Key)
	require.NoError(t, err, "Failed to validate API key")
	assert.Equal(t, apiKey.UserID, validatedKey.UserID)
	assert.Equal(t, apiKey.Name, validatedKey.Name)

	// Validate invalid key
	_, err = am.ValidateAPIKey("invalid-key-12345")
	assert.Error(t, err, "Should error for invalid API key")
	assert.Contains(t, err.Error(), "invalid API key")
}

// TestAuthManager_CheckPermission tests permission checking
func TestAuthManager_CheckPermission(t *testing.T) {
	am := utils.NewAuthManager("test-secret", 24*time.Hour, 100)

	// Create users with different roles
	adminUser, _ := am.CreateUser("admin", "password", []string{"admin"})
	regularUser, _ := am.CreateUser("user", "password", []string{"user"})
	multiRoleUser, _ := am.CreateUser("multi", "password", []string{"user", "editor"})

	// Admin should have all permissions
	assert.True(t, am.CheckPermission(adminUser, "admin"), "Admin should have admin permission")
	assert.True(t, am.CheckPermission(adminUser, "user"), "Admin should have user permission")
	assert.True(t, am.CheckPermission(adminUser, "editor"), "Admin should have editor permission")

	// Regular user should only have user permission
	assert.False(t, am.CheckPermission(regularUser, "admin"), "User should not have admin permission")
	assert.True(t, am.CheckPermission(regularUser, "user"), "User should have user permission")
	assert.False(t, am.CheckPermission(regularUser, "editor"), "User should not have editor permission")

	// Multi-role user should have multiple permissions
	assert.False(t, am.CheckPermission(multiRoleUser, "admin"), "Multi-role user should not have admin permission")
	assert.True(t, am.CheckPermission(multiRoleUser, "user"), "Multi-role user should have user permission")
	assert.True(t, am.CheckPermission(multiRoleUser, "editor"), "Multi-role user should have editor permission")
}

// TestRateLimiter_IsAllowed tests rate limiting
func TestRateLimiter_IsAllowed(t *testing.T) {
	// Create rate limiter: 3 requests per 100ms window
	rl := utils.NewRateLimiter(100*time.Millisecond, 3)

	key := "test-client"

	// First 3 requests should be allowed
	assert.True(t, rl.IsAllowed(key), "First request should be allowed")
	assert.True(t, rl.IsAllowed(key), "Second request should be allowed")
	assert.True(t, rl.IsAllowed(key), "Third request should be allowed")

	// Fourth request should be denied
	assert.False(t, rl.IsAllowed(key), "Fourth request should be denied")

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// After window expires, requests should be allowed again
	assert.True(t, rl.IsAllowed(key), "Request after window expiry should be allowed")
}

// TestRateLimiter_MultipleClients tests rate limiting for multiple clients
func TestRateLimiter_MultipleClients(t *testing.T) {
	rl := utils.NewRateLimiter(1*time.Second, 2)

	// Each client has its own limit
	assert.True(t, rl.IsAllowed("client-1"), "Client 1 first request")
	assert.True(t, rl.IsAllowed("client-1"), "Client 1 second request")
	assert.False(t, rl.IsAllowed("client-1"), "Client 1 third request - denied")

	// Client 2 should still have full limit
	assert.True(t, rl.IsAllowed("client-2"), "Client 2 first request")
	assert.True(t, rl.IsAllowed("client-2"), "Client 2 second request")
	assert.False(t, rl.IsAllowed("client-2"), "Client 2 third request - denied")
}

// TestAuthMiddleware_JWTAuth tests JWT authentication middleware
func TestAuthMiddleware_JWTAuth(t *testing.T) {
	secret := "test-secret-key-for-jwt-signing"
	am := utils.NewAuthManager(secret, 24*time.Hour, 100)

	// Create user
	user, _ := am.CreateUser("testuser", "password", []string{"admin"})

	// Generate JWT token
	token, _ := am.GenerateJWT(user)

	// Create middleware
	middleware := am.AuthMiddleware([]string{"admin"})

	// Test with valid JWT
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check user is in context
		ctxUser, ok := utils.GetUserFromContext(r.Context())
		assert.True(t, ok, "User should be in context")
		assert.Equal(t, user.Username, ctxUser.Username)
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code, "Request with valid JWT should succeed")

	// Test with invalid JWT
	req2 := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req2.Header.Set("Authorization", "Bearer invalid-token")
	rr2 := httptest.NewRecorder()

	handler.ServeHTTP(rr2, req2)
	assert.Equal(t, http.StatusUnauthorized, rr2.Code, "Request with invalid JWT should be unauthorized")

	// Test without JWT
	req3 := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rr3 := httptest.NewRecorder()

	handler.ServeHTTP(rr3, req3)
	assert.Equal(t, http.StatusUnauthorized, rr3.Code, "Request without JWT should be unauthorized")
}

// TestAuthMiddleware_APIKeyAuth tests API key authentication middleware
func TestAuthMiddleware_APIKeyAuth(t *testing.T) {
	am := utils.NewAuthManager("test-secret", 24*time.Hour, 100)

	// Create user and API key
	user, _ := am.CreateUser("apiuser", "password", []string{"user"})
	apiKey, _ := am.CreateAPIKey(user.ID, "Test API Key")

	middleware := am.AuthMiddleware([]string{"user"})

	// Test with valid API key
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("X-API-Key", apiKey.Key)
	rr := httptest.NewRecorder()

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxUser, ok := utils.GetUserFromContext(r.Context())
		assert.True(t, ok, "User should be in context")
		assert.Equal(t, user.Username, ctxUser.Username)
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code, "Request with valid API key should succeed")

	// Test with invalid API key
	req2 := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req2.Header.Set("X-API-Key", "invalid-key")
	rr2 := httptest.NewRecorder()

	handler.ServeHTTP(rr2, req2)
	assert.Equal(t, http.StatusUnauthorized, rr2.Code, "Request with invalid API key should be unauthorized")
}

// TestAuthMiddleware_BasicAuth tests basic authentication middleware
func TestAuthMiddleware_BasicAuth(t *testing.T) {
	am := utils.NewAuthManager("test-secret", 24*time.Hour, 100)

	// Create user
	user, _ := am.CreateUser("basicuser", "password123", []string{"user"})

	middleware := am.AuthMiddleware([]string{"user"})

	// Test with valid basic auth
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.SetBasicAuth(user.Username, "password123")
	rr := httptest.NewRecorder()

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxUser, ok := utils.GetUserFromContext(r.Context())
		assert.True(t, ok, "User should be in context")
		assert.Equal(t, user.Username, ctxUser.Username)
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code, "Request with valid basic auth should succeed")

	// Test with invalid basic auth
	req2 := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req2.SetBasicAuth(user.Username, "wrongpassword")
	rr2 := httptest.NewRecorder()

	handler.ServeHTTP(rr2, req2)
	assert.Equal(t, http.StatusUnauthorized, rr2.Code, "Request with invalid basic auth should be unauthorized")
}

// TestAuthMiddleware_RateLimit tests rate limiting in middleware
func TestAuthMiddleware_RateLimit(t *testing.T) {
	// Create auth manager with strict rate limit
	am := utils.NewAuthManager("test-secret", 24*time.Hour, 2)

	middleware := am.AuthMiddleware([]string{})

	clientIP := "192.168.1.1"

	// First two requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.RemoteAddr = clientIP + ":12345"
		rr := httptest.NewRecorder()

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code, "Request without auth should be unauthorized (but not rate limited)")
	}

	// Third request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.RemoteAddr = clientIP + ":12345"
	rr := httptest.NewRecorder()

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code, "Third request should be rate limited")
}

// TestSecurityHeadersMiddleware tests security headers middleware
func TestSecurityHeadersMiddleware(t *testing.T) {
	middleware := utils.SecurityHeadersMiddleware

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rr := httptest.NewRecorder()

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)

	// Check security headers
	assert.Equal(t, "nosniff", rr.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rr.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", rr.Header().Get("X-XSS-Protection"))
	assert.Contains(t, rr.Header().Get("Strict-Transport-Security"), "max-age=31536000")
	assert.Contains(t, rr.Header().Get("Content-Security-Policy"), "default-src 'self'")
	assert.Equal(t, "strict-origin-when-cross-origin", rr.Header().Get("Referrer-Policy"))
}

// TestInputValidationMiddleware_ContentType tests content type validation
func TestInputValidationMiddleware_ContentType(t *testing.T) {
	middleware := utils.InputValidationMiddleware

	// Test valid JSON content type
	req := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(`{"key": "value"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code, "Valid JSON content type should be allowed")

	// Test invalid content type
	req2 := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader("plain text"))
	req2.Header.Set("Content-Type", "text/plain")
	rr2 := httptest.NewRecorder()

	handler.ServeHTTP(rr2, req2)
	assert.Equal(t, http.StatusBadRequest, rr2.Code, "Invalid content type should be rejected")
}

// TestInputValidationMiddleware_QueryParams tests query parameter validation
func TestInputValidationMiddleware_QueryParams(t *testing.T) {
	middleware := utils.InputValidationMiddleware

	// Test valid query params
	req := httptest.NewRequest(http.MethodGet, "/api/test?key=value", nil)
	rr := httptest.NewRecorder()

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code, "Valid query params should be allowed")

	// Test dangerous query params
	req2 := httptest.NewRequest(http.MethodGet, "/api/test?key=<script>", nil)
	rr2 := httptest.NewRecorder()

	handler.ServeHTTP(rr2, req2)
	assert.Equal(t, http.StatusBadRequest, rr2.Code, "Dangerous query params should be rejected")
}

// TestInitAuthManager tests auth manager initialization
func TestInitAuthManager(t *testing.T) {
	// Test with auth enabled
	config := utils.SecurityConfig{
		JWTSecret:   "custom-secret-key",
		TokenExpiry: 12,
		RateLimit:   500,
		EnableAuth:  true,
	}

	err := utils.InitAuthManager(config)
	require.NoError(t, err, "Failed to initialize auth manager")

	am := utils.GetAuthManager()
	assert.Equal(t, 12*time.Hour, am.GetTokenExpiry(), "Token expiry should be set correctly")

	// Verify default admin user was created
	users := am.GetUsers()
	adminUser, exists := users["admin"]
	assert.True(t, exists, "Default admin user should exist")
	assert.Equal(t, []string{"admin"}, adminUser.Roles, "Admin user should have admin role")

	// Try to authenticate as admin
	authUser, err := am.AuthenticateUser("admin", "admin123")
	require.NoError(t, err, "Should authenticate default admin")
	assert.Equal(t, "admin", authUser.Username)
}

// TestAuthClaims_JWTClaims tests JWT claims structure
func TestAuthClaims_JWTClaims(t *testing.T) {
	claims := utils.AuthClaims{
		UserID:   "user-123",
		Username: "testuser",
		Roles:    []string{"admin", "user"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "mimir-aip",
			Subject:   "user-123",
		},
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("test-secret"))
	require.NoError(t, err, "Failed to sign token")

	// Parse and verify claims
	parsedToken, err := jwt.ParseWithClaims(tokenString, &utils.AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})
	require.NoError(t, err, "Failed to parse token")

	parsedClaims, ok := parsedToken.Claims.(*utils.AuthClaims)
	require.True(t, ok, "Should be able to cast to AuthClaims")
	assert.Equal(t, claims.UserID, parsedClaims.UserID)
	assert.Equal(t, claims.Username, parsedClaims.Username)
	assert.Equal(t, claims.Roles, parsedClaims.Roles)
	assert.Equal(t, claims.Issuer, parsedClaims.Issuer)
	assert.Equal(t, claims.Subject, parsedClaims.Subject)
}

// TestGetUserFromContext tests getting user from context
func TestGetUserFromContext(t *testing.T) {
	// Create user
	user := &utils.User{
		ID:       "user-001",
		Username: "testuser",
		Roles:    []string{"user"},
	}

	// Create context with user
	ctx := context.WithValue(context.Background(), "user", user)

	// Get user from context
	retrievedUser, ok := utils.GetUserFromContext(ctx)
	assert.True(t, ok, "Should find user in context")
	assert.Equal(t, user.Username, retrievedUser.Username)

	// Try to get user from empty context
	_, ok = utils.GetUserFromContext(context.Background())
	assert.False(t, ok, "Should not find user in empty context")
}
