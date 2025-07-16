package utils

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
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

// Generate a UUID v7 and base64-encode it (url-safe)
func GenerateUUID() (string, error) {
	// Generate a UUID v7
	uuidObj, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("failed to generate UUID v7: %v", err)
	}

	// convert UUID to binary
	uuidBytes, err := uuidObj.MarshalBinary()
	if err != nil {
		return "", fmt.Errorf("failed to marshal UUID to binary: %v", err)
	}

	// Kodiere die UUID als Base64URL ohne Padding
	// für die Kompatibilität mit existierenden UUIDs wie "JVsDBnHCZbqoAjSKUPLgGn"
	encodedUUID := base64.RawURLEncoding.EncodeToString(uuidBytes)

	return encodedUUID, nil
}

// CreateAEAD creates an AEAD cipher for encryption/decryption
func CreateAEAD(key []byte) (cipher.AEAD, error) {
	return chacha20poly1305.New(key)
}

// EncryptText encrypts text using the provided key
func EncryptText(text, key string) (string, error) {
	// Decode key
	keyBytes, err := base64.URLEncoding.DecodeString(key)
	if err != nil {
		return "", fmt.Errorf("error decoding key: %v", err)
	}

	// Create AEAD cipher
	aead, err := chacha20poly1305.New(keyBytes)
	if err != nil {
		return "", fmt.Errorf("error creating cipher: %v", err)
	}

	// Create nonce
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("error creating nonce: %v", err)
	}

	// Encrypt text
	ciphertext := aead.Seal(nonce, nonce, []byte(text), nil)
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// DecryptText decrypts text using the provided key
func DecryptText(ciphertext, key string) (string, error) {
	// Decode key and ciphertext
	keyBytes, err := base64.URLEncoding.DecodeString(key)
	if err != nil {
		return "", fmt.Errorf("error decoding key: %v", err)
	}

	ciphertextBytes, err := base64.URLEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("error decoding ciphertext: %v", err)
	}

	// Create AEAD cipher
	aead, err := chacha20poly1305.New(keyBytes)
	if err != nil {
		return "", fmt.Errorf("error creating cipher: %v", err)
	}

	// Extract nonce from ciphertext
	if len(ciphertextBytes) < aead.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertextBytes := ciphertextBytes[:aead.NonceSize()], ciphertextBytes[aead.NonceSize():]

	// Decrypt text
	plaintext, err := aead.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("error decrypting ciphertext: %v", err)
	}

	return string(plaintext), nil
}

// EncryptFile encrypts a file using the provided key
func EncryptFile(data []byte, key string) ([]byte, error) {
	// Decode key
	keyBytes, err := base64.URLEncoding.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("error decoding key: %v", err)
	}

	// Create AEAD cipher
	aead, err := chacha20poly1305.New(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("error creating cipher: %v", err)
	}

	// Create nonce
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("error creating nonce: %v", err)
	}

	// Encrypt file
	ciphertext := aead.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// DecryptFile decrypts a file using the provided key
func DecryptFile(ciphertext []byte, key string) ([]byte, error) {
	// Decode key
	keyBytes, err := base64.URLEncoding.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("error decoding key: %v", err)
	}

	// Create AEAD cipher
	aead, err := chacha20poly1305.New(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("error creating cipher: %v", err)
	}

	// Extract nonce from ciphertext
	if len(ciphertext) < aead.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:aead.NonceSize()], ciphertext[aead.NonceSize():]

	// Decrypt file
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("error decrypting ciphertext: %v", err)
	}

	return plaintext, nil
}

// GetEncryptionKey retrieves the encryption key for a specific user
func GetEncryptionKey(userID int, derivedKey string) (string, error) {
	// Get users
	users, err := GetUsers()
	if err != nil {
		return "", fmt.Errorf("error retrieving users: %v", err)
	}

	// Find user
	usersList, ok := users["users"].([]any)
	if !ok {
		return "", fmt.Errorf("users.json is not in the correct format")
	}

	for _, u := range usersList {
		user, ok := u.(map[string]any)
		if !ok {
			continue
		}

		if id, ok := user["user_id"].(float64); ok && int(id) == userID {
			encEncKey, ok := user["enc_enc_key"].(string)
			if !ok {
				return "", fmt.Errorf("user data is not in the correct format")
			}

			// Decode derived key
			derivedKeyBytes, err := base64.StdEncoding.DecodeString(derivedKey)
			if err != nil {
				return "", fmt.Errorf("error decoding derived key: %v", err)
			}

			// Create Fernet cipher
			aead, err := CreateAEAD(derivedKeyBytes)
			if err != nil {
				return "", fmt.Errorf("error creating cipher: %v", err)
			}

			// Decode encrypted key
			encEncKeyBytes, err := base64.StdEncoding.DecodeString(encEncKey)
			if err != nil {
				return "", fmt.Errorf("error decoding encrypted key: %v", err)
			}

			// Extract nonce from encrypted key
			if len(encEncKeyBytes) < aead.NonceSize() {
				return "", fmt.Errorf("encrypted key too short")
			}
			nonce, encKeyBytes := encEncKeyBytes[:aead.NonceSize()], encEncKeyBytes[aead.NonceSize():]

			// Decrypt key
			keyBytes, err := aead.Open(nil, nonce, encKeyBytes, nil)
			if err != nil {
				return "", fmt.Errorf("error decrypting key: %v", err)
			}

			// Return base64-encoded key
			return base64.URLEncoding.EncodeToString(keyBytes), nil
		}
	}

	return "", fmt.Errorf("user not found")
}
