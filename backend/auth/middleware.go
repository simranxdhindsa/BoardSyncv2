package auth

import (
    "context"
    "net/http"
    "strings"
)

type contextKey string

const UserContextKey contextKey = "user"

// Middleware creates JWT authentication middleware
func (s *Service) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Get token from Authorization header
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            http.Error(w, "Missing authorization header", http.StatusUnauthorized)
            return
        }

        // Check if header starts with "Bearer "
        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        if tokenString == authHeader {
            http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
            return
        }

        // Validate token
        claims, err := s.ValidateToken(tokenString)
        if err != nil {
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }

        // Add user to request context
        ctx := context.WithValue(r.Context(), UserContextKey, claims)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// OptionalMiddleware creates optional JWT authentication middleware
// This middleware will set user context if token is present and valid, but won't reject requests without tokens
func (s *Service) OptionalMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Get token from Authorization header
        authHeader := r.Header.Get("Authorization")
        if authHeader != "" {
            tokenString := strings.TrimPrefix(authHeader, "Bearer ")
            if tokenString != authHeader {
                // Validate token
                if claims, err := s.ValidateToken(tokenString); err == nil {
                    // Add user to request context
                    ctx := context.WithValue(r.Context(), UserContextKey, claims)
                    r = r.WithContext(ctx)
                }
            }
        }

        next.ServeHTTP(w, r)
    })
}

// GetUserFromContext extracts user claims from request context
func GetUserFromContext(r *http.Request) (*Claims, bool) {
    user, ok := r.Context().Value(UserContextKey).(*Claims)
    return user, ok
}

// RequireAuth is a convenience function to check if user is authenticated
func RequireAuth(w http.ResponseWriter, r *http.Request) (*Claims, bool) {
    user, ok := GetUserFromContext(r)
    if !ok {
        http.Error(w, "Authentication required", http.StatusUnauthorized)
        return nil, false
    }
    return user, true
}