package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"asana-youtrack-sync/database"

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
	if _, err := s.db.GetUserByUsername(req.Username); err == nil {
		return nil, ErrUserExists
	}
	if _, err := s.db.GetUserByEmail(req.Email); err == nil {
		return nil, ErrUserExists
	}

	passwordHash := s.hashPassword(req.Password)

	user, err := s.db.CreateUser(req.Username, req.Email, passwordHash)
	if err != nil {
		return nil, err
	}

	return &UserInfo{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
	}, nil
}

// Login authenticates a user and returns a JWT token
func (s *Service) Login(req LoginRequest) (*LoginResponse, error) {
	user, err := s.db.GetUserByUsername(req.Username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !s.verifyPassword(req.Password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	token, expiresAt, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token: token,
		User: UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
		},
		ExpiresAt: expiresAt,
	}, nil
}

// RefreshToken generates a new JWT token for an existing user
func (s *Service) RefreshToken(userID int) (*TokenResponse, error) {
	user, err := s.db.GetUserByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	token, expiresAt, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &TokenResponse{
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

// GetUser retrieves user information by ID
func (s *Service) GetUser(userID int) (*UserInfo, error) {
	user, err := s.db.GetUserByID(userID)
	if err != nil {
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
	user, err := s.db.GetUserByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	if !s.verifyPassword(req.CurrentPassword, user.PasswordHash) {
		return ErrInvalidCredentials
	}

	newPasswordHash := s.hashPassword(req.NewPassword)
	if err = s.db.UpdateUserPassword(userID, newPasswordHash); err != nil {
		return err
	}
	return nil
}

// DeleteUserAccount deletes a user account and all associated data
func (s *Service) DeleteUserAccount(userID int, password string) error {
	user, err := s.db.GetUserByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	if !s.verifyPassword(password, user.PasswordHash) {
		return ErrInvalidCredentials
	}

	return s.db.DeleteUser(userID)
}

// GetUserDataSummary returns a summary of user's data for confirmation
func (s *Service) GetUserDataSummary(userID int) (map[string]interface{}, error) {
	user, err := s.db.GetUserByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	dataSummary, err := s.db.GetUserDataSummary(userID)
	if err != nil {
		return nil, err
	}

	totalRecords := dataSummary["settings"] +
		dataSummary["operations"] +
		dataSummary["ignored_tickets"]
	_ = totalRecords

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
// ValidateTokenUserID validates a JWT and returns only the user ID — used by WebSocket auth
func (s *Service) ValidateTokenUserID(tokenString string) (int, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return 0, err
	}
	return claims.UserID, nil
}

func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
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
	parts := splitString(hashedPassword, "$")
	if len(parts) != 2 {
		return false
	}

	salt, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}

	expectedHash, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	return compareSlices(hash, expectedHash)
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