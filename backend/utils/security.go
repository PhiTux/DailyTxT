package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
)

// Claims represents the JWT claims
type Claims struct {
	UserID     int    `json:"user_id"`
	Username   string `json:"name"`
	DerivedKey string `json:"derived_key"`
	jwt.RegisteredClaims
}

// GenerateToken creates a new JWT token
func GenerateToken(userID int, username, derivedKey string) (string, error) {
	// Create expiration time
	expirationTime := time.Now().Add(time.Duration(Settings.LogoutAfterDays) * 24 * time.Hour)

	// Create claims
	claims := &Claims{
		UserID:     userID,
		Username:   username,
		DerivedKey: derivedKey,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret key
	tokenString, err := token.SignedString([]byte(Settings.SecretToken))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func ValidateToken(tokenString string) (*Claims, error) {
	// Parse token
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(Settings.SecretToken), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// HashPassword hashes a password using Argon2
func HashPassword(password string) (string, string, error) {
	// Generate a random salt
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", "", err
	}

	// Hash password
	hash := argon2.IDKey([]byte(password), salt, 2, 64*1024, 4, 32)

	// Encode salt and hash to base64
	saltBase64 := base64.StdEncoding.EncodeToString(salt)
	hashBase64 := base64.StdEncoding.EncodeToString(hash)

	return hashBase64, saltBase64, nil
}

// VerifyPassword verifies if a password matches a hash
func VerifyPassword(password, hashBase64, saltBase64 string) bool {
	// Decode salt and hash
	salt, err := base64.StdEncoding.DecodeString(saltBase64)
	if err != nil {
		return false
	}

	_, err = base64.StdEncoding.DecodeString(hashBase64)
	if err != nil {
		return false
	}

	// Hash the provided password with the same salt
	hash := argon2.IDKey([]byte(password), salt, 2, 64*1024, 4, 32)

	// Compare hashes
	return base64.StdEncoding.EncodeToString(hash) == hashBase64
}

// DeriveKeyFromPassword derives a key from a password and salt
func DeriveKeyFromPassword(password, saltBase64 string) ([]byte, error) {
	// Decode salt
	salt, err := base64.StdEncoding.DecodeString(saltBase64)
	if err != nil {
		return nil, err
	}

	// Derive key
	key := argon2.IDKey([]byte(password), salt, 2, 64*1024, 4, 32)
	return key, nil
}

// GenerateToken generates a secure random token
func GenerateSecretToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("Failed to generate secret token: %v", err))
	}
	return base64.URLEncoding.EncodeToString(b)
}
