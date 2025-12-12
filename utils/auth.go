package utils

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// AuthClaims represents JWT claims
type AuthClaims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// User represents a system user
type User struct {
	ID       string
	Username string
	Password string // hashed
	Roles    []string
	Active   bool
}

// AuthManager handles authentication and authorization
type AuthManager struct {
	jwtSecret   string
	users       map[string]*User
	apiKeys     map[string]*APIKey
	tokenExpiry time.Duration
	rateLimiter *RateLimiter
}

// APIKey represents an API key
type APIKey struct {
	Key     string
	UserID  string
	Name    string
	Active  bool
	Created time.Time
}

// RateLimiter implements rate limiting
type RateLimiter struct {
	requests map[string][]time.Time
	window   time.Duration
	limit    int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(window time.Duration, limit int) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		window:   window,
		limit:    limit,
	}
}

// IsAllowed checks if a request is allowed
func (rl *RateLimiter) IsAllowed(key string) bool {
	now := time.Now()

	// Clean old requests
	if requests, exists := rl.requests[key]; exists {
		var validRequests []time.Time
		for _, reqTime := range requests {
			if now.Sub(reqTime) < rl.window {
				validRequests = append(validRequests, reqTime)
			}
		}
		rl.requests[key] = validRequests
	}

	// Check if under limit
	if len(rl.requests[key]) >= rl.limit {
		return false
	}

	// Add new request
	rl.requests[key] = append(rl.requests[key], now)
	return true
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(jwtSecret string, tokenExpiry time.Duration, rateLimit int) *AuthManager {
	return &AuthManager{
		jwtSecret:   jwtSecret,
		users:       make(map[string]*User),
		apiKeys:     make(map[string]*APIKey),
		tokenExpiry: tokenExpiry,
		rateLimiter: NewRateLimiter(time.Minute, rateLimit),
	}
}

// HashPassword hashes a password using SHA-256
func (am *AuthManager) HashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// VerifyPassword verifies a password against a hash
func (am *AuthManager) VerifyPassword(password, hash string) bool {
	return am.HashPassword(password) == hash
}

// CreateUser creates a new user
func (am *AuthManager) CreateUser(username, password string, roles []string) (*User, error) {
	if _, exists := am.users[username]; exists {
		return nil, fmt.Errorf("user already exists")
	}

	user := &User{
		ID:       generateID(),
		Username: username,
		Password: am.HashPassword(password),
		Roles:    roles,
		Active:   true,
	}

	am.users[username] = user
	return user, nil
}

// AuthenticateUser authenticates a user with username and password
func (am *AuthManager) AuthenticateUser(username, password string) (*User, error) {
	user, exists := am.users[username]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	if !user.Active {
		return nil, fmt.Errorf("user is inactive")
	}

	if !am.VerifyPassword(password, user.Password) {
		return nil, fmt.Errorf("invalid password")
	}

	return user, nil
}

// GenerateJWT generates a JWT token for a user
func (am *AuthManager) GenerateJWT(user *User) (string, error) {
	claims := AuthClaims{
		UserID:   user.ID,
		Username: user.Username,
		Roles:    user.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(am.tokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "mimir-aip",
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(am.jwtSecret))
}

// ValidateJWT validates a JWT token
func (am *AuthManager) ValidateJWT(tokenString string) (*AuthClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(am.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*AuthClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// CreateAPIKey creates a new API key for a user
func (am *AuthManager) CreateAPIKey(userID, name string) (*APIKey, error) {
	key := generateAPIKey()

	apiKey := &APIKey{
		Key:     key,
		UserID:  userID,
		Name:    name,
		Active:  true,
		Created: time.Now(),
	}

	am.apiKeys[key] = apiKey
	return apiKey, nil
}

// ValidateAPIKey validates an API key
func (am *AuthManager) ValidateAPIKey(key string) (*APIKey, error) {
	apiKey, exists := am.apiKeys[key]
	if !exists {
		return nil, fmt.Errorf("invalid API key")
	}

	if !apiKey.Active {
		return nil, fmt.Errorf("API key is inactive")
	}

	return apiKey, nil
}

// CheckPermission checks if a user has a specific permission
func (am *AuthManager) CheckPermission(user *User, permission string) bool {
	for _, role := range user.Roles {
		if role == "admin" {
			return true
		}
		if role == permission {
			return true
		}
	}
	return false
}

// AuthMiddleware creates authentication middleware
func (am *AuthManager) AuthMiddleware(requiredRoles []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check rate limit first
			clientIP := getClientIP(r)
			if !am.rateLimiter.IsAllowed(clientIP) {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			var user *User

			// Check for JWT token first
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				claims, jwtErr := am.ValidateJWT(token)
				if jwtErr == nil {
					// Find user by ID
					for _, u := range am.users {
						if u.ID == claims.UserID {
							user = u
							break
						}
					}
				}
			}

			// Check for API key if no JWT user found
			if user == nil {
				apiKey := r.Header.Get("X-API-Key")
				if apiKey != "" {
					key, err := am.ValidateAPIKey(apiKey)
					if err == nil {
						// Find user by ID
						for _, u := range am.users {
							if u.ID == key.UserID {
								user = u
								break
							}
						}
					}
				}
			}

			// If still no user, check for basic auth
			if user == nil {
				username, password, ok := r.BasicAuth()
				if ok {
					user, _ = am.AuthenticateUser(username, password)
				}
			}

			if user == nil {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			// Check permissions
			if len(requiredRoles) > 0 {
				hasPermission := false
				for _, requiredRole := range requiredRoles {
					if am.CheckPermission(user, requiredRole) {
						hasPermission = true
						break
					}
				}
				if !hasPermission {
					http.Error(w, "Insufficient permissions", http.StatusForbidden)
					return
				}
			}

			// Add user to context
			ctx := context.WithValue(r.Context(), "user", user)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeadersMiddleware adds security headers
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		// CSP relaxed to allow Next.js inline scripts and styles
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		next.ServeHTTP(w, r)
	})
}

// InputValidationMiddleware validates input data
func InputValidationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate content type for POST/PUT requests
		if r.Method == "POST" || r.Method == "PUT" {
			contentType := r.Header.Get("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
				return
			}

			// Check content length
			if r.ContentLength > 1024*1024 { // 1MB limit
				http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
				return
			}
		}

		// Validate query parameters
		for _, values := range r.URL.Query() {
			for _, value := range values {
				if len(value) > 1000 { // Parameter too long
					http.Error(w, "Query parameter too long", http.StatusBadRequest)
					return
				}
				// Check for potentially dangerous characters
				if strings.Contains(value, "<") || strings.Contains(value, ">") {
					http.Error(w, "Invalid characters in query parameter", http.StatusBadRequest)
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// generateID generates a random ID
func generateID() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// generateAPIKey generates a random API key
func generateAPIKey() string {
	bytes := make([]byte, 32)
	_, _ = rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

// getClientIP gets the client IP address
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP if multiple
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if strings.Contains(ip, ":") {
		ip, _, _ = strings.Cut(ip, ":")
	}
	return ip
}

// GetUserFromContext gets the user from request context
func GetUserFromContext(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value("user").(*User)
	return user, ok
}

// GetUsers returns all users (for admin endpoints)
func (am *AuthManager) GetUsers() map[string]*User {
	return am.users
}

// GetTokenExpiry returns the token expiry duration
func (am *AuthManager) GetTokenExpiry() time.Duration {
	return am.tokenExpiry
}

// Global auth manager instance
var globalAuthManager *AuthManager
var authOnce sync.Once

// GetAuthManager returns the global auth manager instance
func GetAuthManager() *AuthManager {
	authOnce.Do(func() {
		globalAuthManager = NewAuthManager("change-me-in-production", 24*time.Hour, 1000)
	})
	return globalAuthManager
}

// InitAuthManager initializes the global auth manager with configuration
func InitAuthManager(config SecurityConfig) error {
	authManager := GetAuthManager()

	// Update JWT secret
	if config.JWTSecret != "" {
		authManager.jwtSecret = config.JWTSecret
	}

	// Update token expiry
	if config.TokenExpiry > 0 {
		authManager.tokenExpiry = time.Duration(config.TokenExpiry) * time.Hour
	}

	// Update rate limit
	if config.RateLimit > 0 {
		authManager.rateLimiter = NewRateLimiter(time.Minute, config.RateLimit)
	}

	// Create default admin user if auth is enabled
	if config.EnableAuth {
		_, err := authManager.CreateUser("admin", "admin123", []string{"admin"})
		if err != nil && !strings.Contains(err.Error(), "user already exists") {
			return fmt.Errorf("failed to create default admin user: %w", err)
		}
	}

	return nil
}
