package middleware

import (
	"net/http"
	"strings"
)

// AuthMiddleware validates the Bearer token for webhook requests
type AuthMiddleware struct {
	authToken string
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(authToken string) *AuthMiddleware {
	return &AuthMiddleware{
		authToken: authToken,
	}
}

// Authenticate validates the Bearer token in the request
func (m *AuthMiddleware) Authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If no auth token is configured, skip authentication
		if m.authToken == "" {
			next(w, r)
			return
		}

		// Get Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized: Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Validate Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Unauthorized: Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		// Validate token
		if parts[1] != m.authToken {
			http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}

		// Token is valid, proceed to handler
		next(w, r)
	}
}
