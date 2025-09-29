package auth

import (
	"errors"
	"fmt"
	"time"

	"asana-youtrack-sync/database"
	"crypto/rand"
	"encoding/base64"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
)

var (
	ErrUserExists         = errors.New("user already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// Service handles authentication operations
type Service struct {
	db        *database.DB
	jwtSecret []byte
}

// NewService creates a new authentication service
func NewService(db *database.DB, jwtSecret string) *Service {
	return &Service{
		db:        db,
		jwtSecret: []byte(jwtSecret),
	}
}

// Register creates a new user account
func (s *Service) Register(req RegisterRequest) (*UserInfo, error) {
	fmt.Printf("DEBUG: Attempting to register user: %s\n", req.Username)

	// Check if user already exists
	if _, err := s.db.GetUserByUsername(req.Username); err == nil {
		fmt.Printf("DEBUG: User %s already exists by username\n", req.Username)
		return nil, ErrUserExists
	}
	if _, err := s.db.GetUserByEmail(req.Email); err == nil {
		fmt.Printf("DEBUG: User %s already exists by email\n", req.Email)
		return nil, ErrUserExists
	}

	// Hash password
	passwordHash := s.hashPassword(req.Password)
	fmt.Printf("DEBUG: Password hashed for user: %s\n", req.Username)

	// Create user
	user, err := s.db.CreateUser(req.Username, req.Email, passwordHash)
	if err != nil {
		fmt.Printf("DEBUG: Failed to create user in database: %v\n", err)
		return nil, err
	}

	fmt.Printf("DEBUG: User %s registered successfully with ID: %d\n", user.Username, user.ID)

	return &UserInfo{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
	}, nil
}

// Login authenticates a user and returns a JWT token
func (s *Service) Login(req LoginRequest) (*LoginResponse, error) {
	fmt.Printf("DEBUG: Attempting to login user: %s\n", req.Username)

	// Get user by username
	user, err := s.db.GetUserByUsername(req.Username)
	if err != nil {
		fmt.Printf("DEBUG: User not found: %s\n", req.Username)
		return nil, ErrInvalidCredentials
	}

	fmt.Printf("DEBUG: Found user: %s (ID: %d)\n", user.Username, user.ID)

	// Verify password
	if !s.verifyPassword(req.Password, user.PasswordHash) {
		fmt.Printf("DEBUG: Password verification failed for user: %s\n", req.Username)
		return nil, ErrInvalidCredentials
	}

	fmt.Printf("DEBUG: Password verified successfully for user: %s\n", req.Username)

	// Generate JWT token
	token, expiresAt, err := s.generateToken(user)
	if err != nil {
		fmt.Printf("DEBUG: Failed to generate token for user %s: %v\n", req.Username, err)
		return nil, err
	}

	fmt.Printf("DEBUG: Token generated successfully for user: %s (length: %d)\n", req.Username, len(token))

	response := &LoginResponse{
		Token: token,
		User: UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
		},
		ExpiresAt: expiresAt,
	}

	fmt.Printf("DEBUG: Login response prepared for user: %s\n", req.Username)
	return response, nil
}

// RefreshToken generates a new JWT token for an existing user
func (s *Service) RefreshToken(userID int) (*TokenResponse, error) {
	fmt.Printf("DEBUG: Refreshing token for user ID: %d\n", userID)

	user, err := s.db.GetUserByID(userID)
	if err != nil {
		fmt.Printf("DEBUG: User not found for token refresh: %d\n", userID)
		return nil, ErrUserNotFound
	}

	token, expiresAt, err := s.generateToken(user)
	if err != nil {
		fmt.Printf("DEBUG: Failed to generate refresh token: %v\n", err)
		return nil, err
	}

	fmt.Printf("DEBUG: Token refreshed successfully for user: %s\n", user.Username)

	return &TokenResponse{
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

// GetUser retrieves user information by ID
func (s *Service) GetUser(userID int) (*UserInfo, error) {
	fmt.Printf("DEBUG: Getting user info for ID: %d\n", userID)

	user, err := s.db.GetUserByID(userID)
	if err != nil {
		fmt.Printf("DEBUG: User not found: %d\n", userID)
		return nil, ErrUserNotFound
	}

	return &UserInfo{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
	}, nil
}

// ChangePassword changes a user's password
func (s *Service) ChangePassword(userID int, req ChangePasswordRequest) error {
	fmt.Printf("DEBUG: Changing password for user ID: %d\n", userID)

	user, err := s.db.GetUserByID(userID)
	if err != nil {
		fmt.Printf("DEBUG: User not found for password change: %d\n", userID)
		return ErrUserNotFound
	}

	// Verify current password
	if !s.verifyPassword(req.CurrentPassword, user.PasswordHash) {
		fmt.Printf("DEBUG: Current password verification failed for user: %s\n", user.Username)
		return ErrInvalidCredentials
	}

	// Hash new password
	newPasswordHash := s.hashPassword(req.NewPassword)

	// Update password in database
	err = s.db.UpdateUserPassword(userID, newPasswordHash)
	if err != nil {
		fmt.Printf("DEBUG: Failed to update password in database: %v\n", err)
		return err
	}

	fmt.Printf("DEBUG: Password changed successfully for user: %s\n", user.Username)
	return nil
}

// DeleteUserAccount deletes a user account and all associated data
func (s *Service) DeleteUserAccount(userID int, password string) error {
	fmt.Printf("AUTH: Attempting to delete account for user ID: %d\n", userID)

	// Get user to verify password
	user, err := s.db.GetUserByID(userID)
	if err != nil {
		fmt.Printf("AUTH: User not found: %d\n", userID)
		return ErrUserNotFound
	}

	// Verify password before deletion
	if !s.verifyPassword(password, user.PasswordHash) {
		fmt.Printf("AUTH: Password verification failed for user: %s\n", user.Username)
		return ErrInvalidCredentials
	}

	fmt.Printf("AUTH: Password verified, proceeding with account deletion for user: %s\n", user.Username)

	// Delete user and all associated data
	err = s.db.DeleteUser(userID)
	if err != nil {
		fmt.Printf("AUTH: Failed to delete user from database: %v\n", err)
		return err
	}

	fmt.Printf("AUTH: Account and all associated data deleted successfully for user: %s (ID: %d)\n", user.Username, userID)
	return nil
}

// GetUserDataSummary returns a summary of user's data for confirmation
func (s *Service) GetUserDataSummary(userID int) (map[string]interface{}, error) {
	fmt.Printf("AUTH: Getting data summary for user ID: %d\n", userID)

	user, err := s.db.GetUserByID(userID)
	if err != nil {
		fmt.Printf("AUTH: User not found: %d\n", userID)
		return nil, ErrUserNotFound
	}

	dataSummary, err := s.db.GetUserDataSummary(userID)
	if err != nil {
		fmt.Printf("AUTH: Failed to get data summary: %v\n", err)
		return nil, err
	}

	totalRecords := dataSummary["settings"] + 
		dataSummary["operations"] + 
		dataSummary["ignored_tickets"]

	fmt.Printf("AUTH: Data summary retrieved for user: %s (Total records: %d)\n", user.Username, totalRecords)

	return map[string]interface{}{
		"user": map[string]interface{}{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"created_at": user.CreatedAt.Format(time.RFC3339),
		},
		"data_summary": dataSummary,
		"total_records": totalRecords,
		"warning": "⚠️ This action is IRREVERSIBLE. All your data including settings, sync history, and ignored tickets will be permanently deleted.",
	}, nil
}

// ValidateToken validates a JWT token and returns claims
func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	fmt.Printf("DEBUG: Validating token (length: %d)\n", len(tokenString))

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		fmt.Printf("DEBUG: Token parsing failed: %v\n", err)
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		fmt.Printf("DEBUG: Token validated successfully for user: %s (ID: %d)\n", claims.Username, claims.UserID)
		return claims, nil
	}

	fmt.Printf("DEBUG: Token validation failed: invalid claims\n")
	return nil, errors.New("invalid token")
}

// generateToken creates a new JWT token for a user
func (s *Service) generateToken(user *database.User) (string, time.Time, error) {
	expiresAt := time.Now().Add(24 * time.Hour) // Token valid for 24 hours

	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.Username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// hashPassword hashes a password using Argon2
func (s *Service) hashPassword(password string) string {
	salt := make([]byte, 16)
	rand.Read(salt)

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	// Encode salt and hash to base64
	saltEncoded := base64.StdEncoding.EncodeToString(salt)
	hashEncoded := base64.StdEncoding.EncodeToString(hash)

	return saltEncoded + "$" + hashEncoded
}

// verifyPassword verifies a password against its hash
func (s *Service) verifyPassword(password, hashedPassword string) bool {
	fmt.Printf("DEBUG: Verifying password (hash length: %d)\n", len(hashedPassword))

	parts := splitString(hashedPassword, "$")
	if len(parts) != 2 {
		fmt.Printf("DEBUG: Invalid hash format, expected 2 parts, got %d\n", len(parts))
		return false
	}

	salt, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		fmt.Printf("DEBUG: Failed to decode salt: %v\n", err)
		return false
	}

	expectedHash, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		fmt.Printf("DEBUG: Failed to decode hash: %v\n", err)
		return false
	}

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	result := compareSlices(hash, expectedHash)
	fmt.Printf("DEBUG: Password verification result: %t\n", result)
	return result
}

// Helper functions
func splitString(s, sep string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			parts = append(parts, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func compareSlices(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}