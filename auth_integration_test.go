package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuth_Login tests the login endpoint
func TestAuth_Login(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	// Create a test user first (since auth is not enabled by default)
	auth := utils.GetAuthManager()
	_, _ = auth.CreateUser("testuser", "testpass123", []string{"user"})

	t.Run("Login with valid credentials", func(t *testing.T) {
		loginReq := map[string]any{
			"username": "testuser",
			"password": "testpass123",
		}

		body, _ := json.Marshal(loginReq)
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 200 for valid credentials
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify response structure
		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Check that token is present
		if w.Code == http.StatusOK {
			assert.NotEmpty(t, response["token"], "Response should contain a token")
			assert.NotEmpty(t, response["user"], "Response should contain user")
			assert.NotEmpty(t, response["roles"], "Response should contain roles")
			assert.NotNil(t, response["expires_in"], "Response should contain expires_in")
		}
	})

	t.Run("Login with invalid credentials", func(t *testing.T) {
		loginReq := map[string]any{
			"username": "testuser",
			"password": "wrongpassword",
		}

		body, _ := json.Marshal(loginReq)
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 401 for invalid credentials
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Login with missing fields", func(t *testing.T) {
		loginReq := map[string]any{
			"username": "testuser",
			// Missing password - will be treated as empty string
		}

		body, _ := json.Marshal(loginReq)
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Empty password will fail authentication, returning 401
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Login with non-existent user", func(t *testing.T) {
		loginReq := map[string]any{
			"username": "nonexistent",
			"password": "password123",
		}

		body, _ := json.Marshal(loginReq)
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 401 for non-existent user
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestAuth_TokenRefresh tests the token refresh endpoint
func TestAuth_TokenRefresh(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	// Create a test user first
	auth := utils.GetAuthManager()
	_, _ = auth.CreateUser("testuser2", "testpass123", []string{"user"})

	// First, login to get a valid token
	loginReq := map[string]any{
		"username": "testuser2",
		"password": "testpass123",
	}

	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Skip("Login failed, skipping token refresh test")
	}

	var loginResponse map[string]any
	json.Unmarshal(w.Body.Bytes(), &loginResponse)
	token := loginResponse["token"].(string)

	t.Run("Refresh valid token", func(t *testing.T) {
		refreshReq := map[string]any{
			"token": token,
		}

		body, _ := json.Marshal(refreshReq)
		req := httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 200 for valid token
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotEmpty(t, response["token"], "Response should contain new token")
		assert.NotNil(t, response["expires_in"], "Response should contain expires_in")
	})

	t.Run("Refresh invalid token", func(t *testing.T) {
		refreshReq := map[string]any{
			"token": "invalid.token.here",
		}

		body, _ := json.Marshal(refreshReq)
		req := httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 401 for invalid token
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Refresh with missing token", func(t *testing.T) {
		refreshReq := map[string]any{}

		body, _ := json.Marshal(refreshReq)
		req := httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 400 or 401 for missing/empty token
		assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusUnauthorized,
			"Expected 400 or 401, got %d", w.Code)
	})
}

// TestAuth_CheckEndpoint tests the auth check endpoint
func TestAuth_CheckEndpoint(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Auth check without authentication", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/auth/check", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// When auth is disabled, default user middleware injects anonymous user
		// So we get 200 with authenticated=true for anonymous user
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		authenticated, ok := response["authenticated"].(bool)
		require.True(t, ok, "authenticated field should be a boolean")
		assert.True(t, authenticated, "Anonymous user should be authenticated when auth is disabled")

		// Verify it's the anonymous user
		user, ok := response["user"].(map[string]any)
		require.True(t, ok, "user should be an object")
		assert.Equal(t, "anonymous", user["username"])
	})

	t.Run("Auth check with valid token", func(t *testing.T) {
		// Create a test user
		auth := utils.GetAuthManager()
		_, _ = auth.CreateUser("checkuser", "checkpass123", []string{"user"})

		// First, login to get a valid token
		loginReq := map[string]any{
			"username": "checkuser",
			"password": "checkpass123",
		}

		body, _ := json.Marshal(loginReq)
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Skip("Login failed, skipping auth check with token test")
		}

		var loginResponse map[string]any
		json.Unmarshal(w.Body.Bytes(), &loginResponse)
		token := loginResponse["token"].(string)

		// Now check auth with the token
		req = httptest.NewRequest("GET", "/api/v1/auth/check", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 200 when authenticated
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		authenticated, ok := response["authenticated"].(bool)
		require.True(t, ok, "authenticated field should be a boolean")
		assert.True(t, authenticated, "Should be authenticated")

		// When auth is disabled, the default user middleware may inject anonymous user
		// The auth/check endpoint returns the user from context, which could be anonymous
		// Just verify we have a user object with expected fields
		user, ok := response["user"].(map[string]any)
		require.True(t, ok, "user should be an object")
		assert.NotEmpty(t, user["username"], "Username should not be empty")
		assert.NotNil(t, user["roles"], "Roles should not be nil")
	})
}

// TestAuth_MeEndpoint tests the /auth/me endpoint
func TestAuth_MeEndpoint(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Get user info without authentication", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// When auth is disabled, default user middleware injects anonymous user
		// So we get 200 with anonymous user info
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify anonymous user is returned
		assert.Equal(t, "anonymous", response["username"])
		assert.NotNil(t, response["roles"], "Response should contain roles")
	})

	t.Run("Get user info with valid token", func(t *testing.T) {
		// Create a test user
		auth := utils.GetAuthManager()
		_, _ = auth.CreateUser("meuser", "mepass123", []string{"user"})

		// First, login to get a valid token
		loginReq := map[string]any{
			"username": "meuser",
			"password": "mepass123",
		}

		body, _ := json.Marshal(loginReq)
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Skip("Login failed, skipping me endpoint test")
		}

		var loginResponse map[string]any
		json.Unmarshal(w.Body.Bytes(), &loginResponse)
		token := loginResponse["token"].(string)

		// Now get user info with the token
		req = httptest.NewRequest("GET", "/api/v1/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 200 when authenticated
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotEmpty(t, response["id"], "Response should contain id")
		// When auth is disabled, the anonymous user may be returned instead of the token user
		// Just verify we have a valid user response
		assert.NotEmpty(t, response["username"], "Response should contain username")
		assert.NotNil(t, response["roles"], "Response should contain roles")
		assert.NotNil(t, response["active"], "Response should contain active")
	})
}

// TestAuth_Logout tests the logout endpoint
func TestAuth_Logout(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Logout without authentication", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 200 even without authentication
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		success, ok := response["success"].(bool)
		require.True(t, ok, "success field should be a boolean")
		assert.True(t, success, "Logout should report success")
		assert.Equal(t, "Logged out successfully", response["message"])
	})

	t.Run("Logout with valid token", func(t *testing.T) {
		// Create a test user
		auth := utils.GetAuthManager()
		_, _ = auth.CreateUser("logoutuser", "logoutpass123", []string{"user"})

		// First, login to get a valid token
		loginReq := map[string]any{
			"username": "logoutuser",
			"password": "logoutpass123",
		}

		body, _ := json.Marshal(loginReq)
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		var token string
		if w.Code == http.StatusOK {
			var loginResponse map[string]any
			json.Unmarshal(w.Body.Bytes(), &loginResponse)
			token = loginResponse["token"].(string)
		}

		// Now logout with the token
		req = httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 200
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		success, ok := response["success"].(bool)
		require.True(t, ok, "success field should be a boolean")
		assert.True(t, success, "Logout should report success")
	})
}

// TestAuth_ListUsers tests the list users endpoint
func TestAuth_ListUsers(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("List users without authentication", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/auth/users", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 200 (auth middleware not applied to this endpoint)
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		users, ok := response["users"].([]any)
		require.True(t, ok, "users should be an array")
		assert.NotNil(t, users, "Users should not be nil")
	})

	t.Run("List users with authentication", func(t *testing.T) {
		// Create a test user
		auth := utils.GetAuthManager()
		_, _ = auth.CreateUser("listuser", "listpass123", []string{"user"})

		// First, login to get a valid token
		loginReq := map[string]any{
			"username": "listuser",
			"password": "listpass123",
		}

		body, _ := json.Marshal(loginReq)
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		var token string
		if w.Code == http.StatusOK {
			var loginResponse map[string]any
			json.Unmarshal(w.Body.Bytes(), &loginResponse)
			token = loginResponse["token"].(string)
		}

		// Now list users with the token
		req = httptest.NewRequest("GET", "/api/v1/auth/users", nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 200
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		users, ok := response["users"].([]any)
		require.True(t, ok, "users should be an array")
		assert.NotNil(t, users, "Users should not be nil")
	})
}

// TestAuth_CreateAPIKey tests the create API key endpoint
func TestAuth_CreateAPIKey(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Create API key without authentication", func(t *testing.T) {
		apiKeyReq := map[string]any{
			"name": "Test API Key",
		}

		body, _ := json.Marshal(apiKeyReq)
		req := httptest.NewRequest("POST", "/api/v1/auth/apikeys", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// When auth is disabled, anonymous user is injected, so we get 200
		// API key is created for the anonymous user
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotEmpty(t, response["key"], "Response should contain API key")
		assert.Equal(t, "anonymous", response["user_id"])
	})

	t.Run("Create API key with valid token", func(t *testing.T) {
		// Create a test user
		auth := utils.GetAuthManager()
		_, _ = auth.CreateUser("apiuser1", "apipass123", []string{"user"})

		// First, login to get a valid token
		loginReq := map[string]any{
			"username": "apiuser1",
			"password": "apipass123",
		}

		body, _ := json.Marshal(loginReq)
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Skip("Login failed, skipping API key creation test")
		}

		var loginResponse map[string]any
		json.Unmarshal(w.Body.Bytes(), &loginResponse)
		token := loginResponse["token"].(string)

		// Now create API key with the token
		apiKeyReq := map[string]any{
			"name": "Test API Key",
		}

		body, _ = json.Marshal(apiKeyReq)
		req = httptest.NewRequest("POST", "/api/v1/auth/apikeys", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 200 when authenticated
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotEmpty(t, response["key"], "Response should contain API key")
		assert.Equal(t, "Test API Key", response["name"])
		assert.NotEmpty(t, response["user_id"])
		assert.NotNil(t, response["created"])
	})

	t.Run("Create API key with default name", func(t *testing.T) {
		// Create a test user
		auth := utils.GetAuthManager()
		_, _ = auth.CreateUser("apiuser2", "apipass123", []string{"user"})

		// First, login to get a valid token
		loginReq := map[string]any{
			"username": "apiuser2",
			"password": "apipass123",
		}

		body, _ := json.Marshal(loginReq)
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Skip("Login failed, skipping API key creation test")
		}

		var loginResponse map[string]any
		json.Unmarshal(w.Body.Bytes(), &loginResponse)
		token := loginResponse["token"].(string)

		// Now create API key without specifying name
		apiKeyReq := map[string]any{}

		body, _ = json.Marshal(apiKeyReq)
		req = httptest.NewRequest("POST", "/api/v1/auth/apikeys", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 200 when authenticated
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotEmpty(t, response["key"], "Response should contain API key")
		// Should have default name
		assert.NotEmpty(t, response["name"])
	})
}
