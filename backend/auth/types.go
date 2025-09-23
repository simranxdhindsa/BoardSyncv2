package auth

import (
    "time"
    "github.com/golang-jwt/jwt/v5"
)

// LoginRequest represents a login request
type LoginRequest struct {
    Username string `json:"username" validate:"required"`
    Password string `json:"password" validate:"required"`
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
    Username string `json:"username" validate:"required,min=3,max=50"`
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=6"`
}

// LoginResponse represents a login response
type LoginResponse struct {
    Token     string `json:"token"`
    User      UserInfo `json:"user"`
    ExpiresAt time.Time `json:"expires_at"`
}

// UserInfo represents public user information
type UserInfo struct {
    ID       int    `json:"id"`
    Username string `json:"username"`
    Email    string `json:"email"`
}

// Claims represents JWT claims
type Claims struct {
    UserID   int    `json:"user_id"`
    Username string `json:"username"`
    Email    string `json:"email"`
    jwt.RegisteredClaims
}

// TokenResponse represents a token refresh response
type TokenResponse struct {
    Token     string    `json:"token"`
    ExpiresAt time.Time `json:"expires_at"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
    CurrentPassword string `json:"current_password" validate:"required"`
    NewPassword     string `json:"new_password" validate:"required,min=6"`
}

// UserProfileRequest represents a user profile update request
type UserProfileRequest struct {
    Email string `json:"email" validate:"email"`
}
