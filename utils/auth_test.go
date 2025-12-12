package utils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuthManager(t *testing.T) {
	am := NewAuthManager("test-secret", 24*time.Hour, 1000)

	assert.NotNil(t, am)
	assert.Equal(t, "test-secret", am.jwtSecret)
	assert.Equal(t, 24*time.Hour, am.tokenExpiry)
	assert.NotNil(t, am.users)
	assert.NotNil(t, am.apiKeys)
	assert.NotNil(t, am.rateLimiter)
}

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(time.Minute, 10)

	assert.NotNil(t, rl)
	assert.Equal(t, time.Minute, rl.window)
	assert.Equal(t, 10, rl.limit)
	assert.NotNil(t, rl.requests)
}

func TestRateLimiter_IsAllowed(t *testing.T) {
	rl := NewRateLimiter(time.Millisecond*100, 2) // 2 requests per 100ms

	// First two requests should be allowed
	assert.True(t, rl.IsAllowed("client1"))
	assert.True(t, rl.IsAllowed("client1"))

	// Third request should be denied
	assert.False(t, rl.IsAllowed("client1"))

	// Wait for window to reset
	time.Sleep(time.Millisecond * 110)

	// Should be allowed again
	assert.True(t, rl.IsAllowed("client1"))

	// Different client should have separate limits
	assert.True(t, rl.IsAllowed("client2"))
	assert.True(t, rl.IsAllowed("client2"))
	assert.False(t, rl.IsAllowed("client2"))
}

func TestAuthManager_HashPassword(t *testing.T) {
	am := NewAuthManager("test-secret", 24*time.Hour, 1000)

	password := "testpassword"
	hash := am.HashPassword(password)

	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)
	assert.Equal(t, 64, len(hash)) // SHA-256 hex length
}

func TestAuthManager_VerifyPassword(t *testing.T) {
	am := NewAuthManager("test-secret", 24*time.Hour, 1000)

	password := "testpassword"
	hash := am.HashPassword(password)

	// Correct password should verify
	assert.True(t, am.VerifyPassword(password, hash))

	// Incorrect password should not verify
	assert.False(t, am.VerifyPassword("wrongpassword", hash))
}

func TestAuthManager_CreateUser(t *testing.T) {
	am := NewAuthManager("test-secret", 24*time.Hour, 1000)

	user, err := am.CreateUser("testuser", "password123", []string{"user"})

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, am.HashPassword("password123"), user.Password)
	assert.Equal(t, []string{"user"}, user.Roles)
	assert.True(t, user.Active)
	assert.NotEmpty(t, user.ID)
}

func TestAuthManager_CreateUser_Duplicate(t *testing.T) {
	am := NewAuthManager("test-secret", 24*time.Hour, 1000)

	// Create first user
	_, err := am.CreateUser("testuser", "password123", []string{"user"})
	require.NoError(t, err)

	// Try to create duplicate user
	_, err = am.CreateUser("testuser", "differentpassword", []string{"admin"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user already exists")
}

func TestAuthManager_AuthenticateUser(t *testing.T) {
	am := NewAuthManager("test-secret", 24*time.Hour, 1000)

	// Create a user
	_, err := am.CreateUser("testuser", "password123", []string{"user"})
	require.NoError(t, err)

	// Test correct authentication
	user, err := am.AuthenticateUser("testuser", "password123")
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "testuser", user.Username)

	// Test incorrect password
	_, err = am.AuthenticateUser("testuser", "wrongpassword")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid password")

	// Test non-existent user
	_, err = am.AuthenticateUser("nonexistent", "password123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestAuthManager_AuthenticateUser_Inactive(t *testing.T) {
	am := NewAuthManager("test-secret", 24*time.Hour, 1000)

	// Create a user
	user, err := am.CreateUser("testuser", "password123", []string{"user"})
	require.NoError(t, err)

	// Deactivate the user
	user.Active = false

	// Try to authenticate inactive user
	_, err = am.AuthenticateUser("testuser", "password123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user is inactive")
}

func TestAuthManager_GenerateJWT(t *testing.T) {
	am := NewAuthManager("test-secret", 24*time.Hour, 1000)

	user := &User{
		ID:       "user123",
		Username: "testuser",
		Roles:    []string{"user", "admin"},
		Active:   true,
	}

	token, err := am.GenerateJWT(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify token structure
	parts := strings.Split(token, ".")
	assert.Len(t, parts, 3)
}

func TestAuthManager_ValidateJWT(t *testing.T) {
	am := NewAuthManager("test-secret", 24*time.Hour, 1000)

	user := &User{
		ID:       "user123",
		Username: "testuser",
		Roles:    []string{"user", "admin"},
		Active:   true,
	}

	// Generate token
	token, err := am.GenerateJWT(user)
	require.NoError(t, err)

	// Validate token
	claims, err := am.ValidateJWT(token)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.Username, claims.Username)
	assert.Equal(t, user.Roles, claims.Roles)
	assert.Equal(t, "mimir-aip", claims.Issuer)
	assert.Equal(t, user.ID, claims.Subject)
}

func TestAuthManager_ValidateJWT_Invalid(t *testing.T) {
	am := NewAuthManager("test-secret", 24*time.Hour, 1000)

	// Test invalid token
	_, err := am.ValidateJWT("invalid.token.here")
	assert.Error(t, err)

	// Test token with wrong secret
	am2 := NewAuthManager("different-secret", 24*time.Hour, 1000)
	user := &User{
		ID:       "user123",
		Username: "testuser",
		Roles:    []string{"user"},
		Active:   true,
	}

	token, err := am.GenerateJWT(user)
	require.NoError(t, err)

	// Try to validate with different secret
	_, err = am2.ValidateJWT(token)
	assert.Error(t, err)
}

func TestAuthManager_ValidateJWT_Expired(t *testing.T) {
	am := NewAuthManager("test-secret", time.Millisecond, 1000) // Very short expiry

	user := &User{
		ID:       "user123",
		Username: "testuser",
		Roles:    []string{"user"},
		Active:   true,
	}

	token, err := am.GenerateJWT(user)
	require.NoError(t, err)

	// Wait for token to expire
	time.Sleep(time.Millisecond * 10)

	// Try to validate expired token
	_, err = am.ValidateJWT(token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token is expired")
}

func TestAuthManager_CreateAPIKey(t *testing.T) {
	am := NewAuthManager("test-secret", 24*time.Hour, 1000)

	apiKey, err := am.CreateAPIKey("user123", "test-key")
	assert.NoError(t, err)
	assert.NotNil(t, apiKey)
	assert.Equal(t, "user123", apiKey.UserID)
	assert.Equal(t, "test-key", apiKey.Name)
	assert.True(t, apiKey.Active)
	assert.NotEmpty(t, apiKey.Key)
	assert.NotEmpty(t, apiKey.Created)
}

func TestAuthManager_ValidateAPIKey(t *testing.T) {
	am := NewAuthManager("test-secret", 24*time.Hour, 1000)

	// Create API key
	apiKey, err := am.CreateAPIKey("user123", "test-key")
	require.NoError(t, err)

	// Validate valid key
	validKey, err := am.ValidateAPIKey(apiKey.Key)
	assert.NoError(t, err)
	assert.Equal(t, apiKey, validKey)

	// Validate invalid key
	_, err = am.ValidateAPIKey("invalid-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid API key")
}

func TestAuthManager_ValidateAPIKey_Inactive(t *testing.T) {
	am := NewAuthManager("test-secret", 24*time.Hour, 1000)

	// Create API key
	apiKey, err := am.CreateAPIKey("user123", "test-key")
	require.NoError(t, err)

	// Deactivate the key
	apiKey.Active = false

	// Try to validate inactive key
	_, err = am.ValidateAPIKey(apiKey.Key)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API key is inactive")
}

func TestAuthManager_CheckPermission(t *testing.T) {
	am := NewAuthManager("test-secret", 24*time.Hour, 1000)

	tests := []struct {
		name       string
		userRoles  []string
		permission string
		expected   bool
	}{
		{
			name:       "admin has all permissions",
			userRoles:  []string{"admin"},
			permission: "any-permission",
			expected:   true,
		},
		{
			name:       "user has specific permission",
			userRoles:  []string{"user", "read"},
			permission: "read",
			expected:   true,
		},
		{
			name:       "user lacks permission",
			userRoles:  []string{"user"},
			permission: "admin",
			expected:   false,
		},
		{
			name:       "empty roles",
			userRoles:  []string{},
			permission: "admin",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{
				ID:       "user123",
				Username: "testuser",
				Roles:    tt.userRoles,
				Active:   true,
			}

			result := am.CheckPermission(user, tt.permission)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuthManager_AuthMiddleware(t *testing.T) {
	am := NewAuthManager("test-secret", 24*time.Hour, 1000)

	// Create a user
	user, err := am.CreateUser("testuser", "password123", []string{"user"})
	require.NoError(t, err)

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Wrap with middleware
	wrappedHandler := am.AuthMiddleware([]string{})(handler)

	t.Run("valid JWT token", func(t *testing.T) {
		token, err := am.GenerateJWT(user)
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "success", w.Body.String())
	})

	t.Run("valid API key", func(t *testing.T) {
		apiKey, err := am.CreateAPIKey(user.ID, "test-key")
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-API-Key", apiKey.Key)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "success", w.Body.String())
	})

	t.Run("valid basic auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.SetBasicAuth("testuser", "password123")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "success", w.Body.String())
	})

	t.Run("no authentication", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid JWT token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer invalid.token")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestAuthManager_AuthMiddleware_RequiredRoles(t *testing.T) {
	am := NewAuthManager("test-secret", 24*time.Hour, 1000)

	// Create users with different roles
	adminUser, err := am.CreateUser("admin", "password123", []string{"admin"})
	require.NoError(t, err)

	regularUser, err := am.CreateUser("user", "password123", []string{"user"})
	require.NoError(t, err)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Require admin role
	wrappedHandler := am.AuthMiddleware([]string{"admin"})(handler)

	t.Run("admin user has access", func(t *testing.T) {
		token, err := am.GenerateJWT(adminUser)
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("regular user denied", func(t *testing.T) {
		token, err := am.GenerateJWT(regularUser)
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestAuthManager_RateLimiting(t *testing.T) {
	am := NewAuthManager("test-secret", 24*time.Hour, 2) // 2 requests per minute

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := am.AuthMiddleware([]string{})(handler)

	// First two requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code) // No auth, but not rate limited
	}

	// Third request should be rate limited
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := SecurityHeadersMiddleware(handler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "max-age=31536000; includeSubDomains", w.Header().Get("Strict-Transport-Security"))
	assert.Equal(t, "default-src 'self'", w.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
}

func TestInputValidationMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := InputValidationMiddleware(handler)

	t.Run("valid GET request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?param=value", nil)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("valid POST with JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", strings.NewReader(`{"test": "data"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("POST without JSON content type", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", strings.NewReader("data"))
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("POST with too large content", func(t *testing.T) {
		largeData := strings.Repeat("x", 1024*1024+1) // > 1MB
		req := httptest.NewRequest("POST", "/", strings.NewReader(largeData))
		req.Header.Set("Content-Type", "application/json")
		req.ContentLength = int64(len(largeData))
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	})

	t.Run("query parameter too long", func(t *testing.T) {
		longParam := strings.Repeat("x", 1001)
		req := httptest.NewRequest("GET", "/?param="+longParam, nil)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("query parameter with dangerous characters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?param=<script>", nil)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestGetUserFromContext(t *testing.T) {
	user := &User{
		ID:       "user123",
		Username: "testuser",
		Roles:    []string{"user"},
		Active:   true,
	}

	ctx := context.WithValue(context.Background(), "user", user)

	retrievedUser, ok := GetUserFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, user, retrievedUser)

	// Test with no user in context
	_, ok = GetUserFromContext(context.Background())
	assert.False(t, ok)
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expectedIP string
	}{
		{
			name: "X-Forwarded-For header",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.1, 10.0.0.1",
			},
			remoteAddr: "127.0.0.1:8080",
			expectedIP: "192.168.1.1",
		},
		{
			name: "X-Real-IP header",
			headers: map[string]string{
				"X-Real-IP": "192.168.1.2",
			},
			remoteAddr: "127.0.0.1:8080",
			expectedIP: "192.168.1.2",
		},
		{
			name:       "RemoteAddr only",
			headers:    map[string]string{},
			remoteAddr: "127.0.0.1:8080",
			expectedIP: "127.0.0.1",
		},
		{
			name:       "IPv4 RemoteAddr with port",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.100:8080",
			expectedIP: "192.168.1.100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			ip := getClientIP(req)
			assert.Equal(t, tt.expectedIP, ip)
		})
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Equal(t, 32, len(id1)) // 16 bytes = 32 hex chars
	assert.Equal(t, 32, len(id2))
}

func TestGenerateAPIKey(t *testing.T) {
	key1 := generateAPIKey()
	key2 := generateAPIKey()

	assert.NotEmpty(t, key1)
	assert.NotEmpty(t, key2)
	assert.NotEqual(t, key1, key2)
	// Base64 URL encoding of 32 bytes should be around 43 chars
	assert.Greater(t, len(key1), 40)
	assert.Greater(t, len(key2), 40)
}

func TestAuthManager_ConcurrentReads(t *testing.T) {
	am := NewAuthManager("test-secret", 24*time.Hour, 1000)

	// Create a user first
	user, err := am.CreateUser("testuser", "password123", []string{"user"})
	require.NoError(t, err)

	var wg sync.WaitGroup
	numGoroutines := 10

	// Concurrent user reads (safe operation)
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			users := am.GetUsers()
			assert.Contains(t, users, "testuser")
		}()
	}

	// Concurrent authentication (safe operation)
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = am.AuthenticateUser("testuser", "password123")
		}()
	}

	wg.Wait()

	// Verify user still exists
	users := am.GetUsers()
	assert.Contains(t, users, "testuser")

	// Test concurrent permission checks
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = am.CheckPermission(user, "user")
		}()
	}

	wg.Wait()
}

func TestGetAuthManager_Singleton(t *testing.T) {
	am1 := GetAuthManager()
	am2 := GetAuthManager()

	assert.Same(t, am1, am2)
}

func TestInitAuthManager(t *testing.T) {
	config := SecurityConfig{
		EnableAuth:  true,
		JWTSecret:   "new-secret",
		TokenExpiry: 48,
		RateLimit:   500,
	}

	err := InitAuthManager(config)
	assert.NoError(t, err)

	am := GetAuthManager()
	assert.Equal(t, "new-secret", am.jwtSecret)
	assert.Equal(t, 48*time.Hour, am.tokenExpiry)
	assert.Equal(t, 500, am.rateLimiter.limit)

	// Check that default admin user was created
	users := am.GetUsers()
	assert.Contains(t, users, "admin")
}
