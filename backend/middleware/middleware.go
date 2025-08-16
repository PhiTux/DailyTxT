package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/phitux/dailytxt/backend/utils"
)

// CORS middleware handles Cross-Origin Resource Sharing
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get origin from request
		origin := r.Header.Get("Origin")

		// Check if origin is in allowed hosts
		allowed := false
		for _, host := range utils.Settings.AllowedHosts {
			if origin == host {
				allowed = true
				break
			}
		}

		// Set CORS headers if origin is allowed
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Content-Disposition")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireAuth middleware checks if user is authenticated
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from cookie
		cookie, err := r.Cookie("token")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			utils.Logger.Printf("Unauthorized access attempt, no cookie found: %s %s", r.Method, r.URL.Path)
			return
		}

		// Validate JWT token
		claims, err := utils.ValidateToken(cookie.Value)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			utils.Logger.Printf("Unauthorized access attempt, invalid token: %s %s", r.Method, r.URL.Path)
			return
		}

		// Add user info to request context
		ctx := context.WithValue(r.Context(), utils.UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, utils.UsernameKey, claims.Username)
		ctx = context.WithValue(ctx, utils.DerivedKeyKey, claims.DerivedKey)

		// Continue with the next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Logger middleware logs all requests
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If not in development mode, skip detailed logging
		if !utils.Settings.Development {
			next.ServeHTTP(w, r)
			return
		}

		// Skip logging for static files
		if strings.HasPrefix(r.URL.Path, "/static/") {
			next.ServeHTTP(w, r)
			return
		}

		// Log request
		startTime := time.Now()

		// Create a response writer wrapper to capture the status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call the next handler with our custom response writer
		next.ServeHTTP(rw, r)

		// Log response
		duration := time.Since(startTime)
		utils.Logger.Printf("%s %s - Status: %d - Duration: %v", r.Method, r.URL.Path, rw.statusCode, duration)
	})
}

// responseWriter is a wrapper for http.ResponseWriter that captures the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code and delegates to the underlying ResponseWriter
func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}
