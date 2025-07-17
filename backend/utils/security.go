package utils

import (
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
	"runtime"
	"strings"
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

type Argon2Configuration struct {
	HashRaw    []byte
	Salt       []byte
	TimeCost   uint32
	MemoryCost uint32
	Threads    uint8
	KeyLength  uint32
}

// HashPassword hashes a password using Argon2
func HashPassword(password string) (string, error) {
	config := &Argon2Configuration{
		TimeCost:   5,
		MemoryCost: 64 * 1024,
		Threads:    uint8(runtime.NumCPU()),
		KeyLength:  32,
	}

	// Generate a random salt
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}
	config.Salt = salt

	// Hash password
	config.HashRaw = argon2.IDKey([]byte(password), salt, config.TimeCost, config.MemoryCost, config.Threads, config.KeyLength)

	// Generate standardized hash format
	encodedHash := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		config.MemoryCost,
		config.TimeCost,
		config.Threads,
		base64.RawStdEncoding.EncodeToString(config.Salt),
		base64.RawStdEncoding.EncodeToString(config.HashRaw),
	)

	return encodedHash, nil
}

func parseArgon2Hash(encodedHash string) (*Argon2Configuration, error) {
	components := strings.Split(encodedHash, "$")
	if len(components) != 6 {
		return nil, fmt.Errorf("invalid hash format structure")
	}

	// Validate algorithm identifier
	if !strings.HasPrefix(components[1], "argon2id") {
		return nil, fmt.Errorf("unsupported algorithm variant")
	}

	// Extract version information
	var version int
	fmt.Sscanf(components[2], "v=%d", &version)

	// Parse configuration parameters
	config := &Argon2Configuration{}
	fmt.Sscanf(components[3], "m=%d,t=%d,p=%d",
		&config.MemoryCost, &config.TimeCost, &config.Threads)

	// Decode salt component
	salt, err := base64.RawStdEncoding.DecodeString(components[4])
	if err != nil {
		return nil, fmt.Errorf("salt decoding failed: %w", err)
	}
	config.Salt = salt

	// Decode hash component
	hash, err := base64.RawStdEncoding.DecodeString(components[5])
	if err != nil {
		return nil, fmt.Errorf("hash decoding failed: %w", err)
	}
	config.HashRaw = hash
	config.KeyLength = uint32(len(hash))

	return config, nil
}

// VerifyPassword verifies if a password matches a hash
func VerifyPassword(password, hashBase64 string) bool {
	// Parse stored hash parameters
	config, err := parseArgon2Hash(hashBase64)
	if err != nil {
		return false
	}

	// Generate hash using identical parameters
	computedHash := argon2.IDKey(
		[]byte(password),
		config.Salt,
		config.TimeCost,
		config.MemoryCost,
		config.Threads,
		config.KeyLength,
	)

	// Perform constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare(config.HashRaw, computedHash) == 1
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
