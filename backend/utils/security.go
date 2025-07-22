package utils

import (
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
	"math/big"
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

func getArgon2Configuration() *Argon2Configuration {
	return &Argon2Configuration{
		TimeCost:   5,
		MemoryCost: 64 * 1024,
		Threads:    uint8(runtime.NumCPU()),
		KeyLength:  32,
	}
}

// HashPassword hashes a password using Argon2
func HashPassword(password string) (string, error) {
	config := getArgon2Configuration()

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

	// Derive key (don't use config from above, as a variable amount of thread will lead to different results)
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

// CheckPasswordForUser checks if the provided password matches the user's password OR on of his backup codes.
// Returns the derivedKey, if successfully validating password, otherwise empty string
// Return the amount of backup codes available for the user (-1 if password does not match or if backup code was NOT used).
func CheckPasswordForUser(userID int, password string) (string, int, error) {
	// Get users
	users, err := GetUsers()
	if err != nil {
		return "", -1, fmt.Errorf("error retrieving users: %v", err)
	}

	// Find user
	usersList, ok := users["users"].([]any)
	if !ok {
		return "", -1, fmt.Errorf("users.json is not in the correct format")
	}

	for _, u := range usersList {
		user, ok := u.(map[string]any)
		if !ok {
			continue
		}

		if id, ok := user["user_id"].(float64); ok && int(id) == userID {
			passwordHash, ok := user["password"].(string)
			if !ok {
				return "", -1, fmt.Errorf("user data is not in the correct format")
			}

			if VerifyPassword(password, passwordHash) {
				// Calculate derived key
				derKey, err := DeriveKeyFromPassword(password, user["salt"].(string))
				if err != nil {
					return "", -1, fmt.Errorf("error deriving key from password: %v", err)
				}

				return base64.StdEncoding.EncodeToString(derKey), -1, nil
			}

			// Check backup codes
			backupCodes, ok := user["backup_codes"].([]any)
			if !ok {
				return "", -1, nil
			}

			for i, code := range backupCodes {
				codeStr, ok := code.(map[string]any)["password"].(string)
				if !ok {
					Logger.Printf("Invalid backup code format for user %d: %v", userID, code)
					continue // Skip invalid codes
				}

				if !VerifyPassword(password, codeStr) {
					continue
				}

				// Password matched the code! Remove backup code
				backupCodes = append(backupCodes[:i], backupCodes[i+1:]...)

				// Update user data
				user["backup_codes"] = backupCodes
				if err := WriteUsers(users); err != nil {
					return "", -1, fmt.Errorf("error saving updated user data: %v", err)
				}

				// Calculate derived key
				tempKey, err := DeriveKeyFromPassword(password, code.(map[string]any)["salt"].(string))
				if err != nil {
					return "", -1, fmt.Errorf("error deriving key from password: %v", err)
				}

				derKey, err := DecryptText(code.(map[string]any)["enc_derived_key"].(string), base64.URLEncoding.EncodeToString(tempKey))
				if err != nil {
					return "", -1, fmt.Errorf("error decrypting derived key: %v", err)
				}

				return derKey, len(backupCodes), nil
			}

			return "", -1, nil
		}
	}

	return "", -1, nil
}

func CreatePasswordString() string {
	var chars string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789/+-_*!?#$%&(){}[]=@~"
	password := make([]byte, 10)

	for i := range password {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			panic(fmt.Sprintf("Failed to generate random index: %v", err))
		}
		password[i] = chars[n.Int64()]
	}

	return string(password)
}

// GenerateBackupCodes generates 6 backup codes for a user
// With those backup-codes, the derived key gets encrypted
func GenerateBackupCodes(derived_key string) ([]string, []map[string]any, error) {
	backupCodes := make([]string, 6)
	codeData := make([]map[string]any, 6)
	for i := range 6 {
		// Initialize the map for this index
		codeData[i] = make(map[string]any)

		// Generate a random backup code (=password (!= uuid))
		code := CreatePasswordString()

		// create hash
		hash, err := HashPassword(code)
		if err != nil {
			return nil, nil, fmt.Errorf("error hashing backup code: %v", err)
		}

		// Generate a random salt
		salt := make([]byte, 16)
		if _, err := rand.Read(salt); err != nil {
			return nil, nil, fmt.Errorf("error generating salt: %v", err)
		}
		// Convert salt to base64
		saltBase64 := base64.StdEncoding.EncodeToString(salt)

		// Create derived encryption key to later encrypt the original derived key
		intermediateKey, err := DeriveKeyFromPassword(code, saltBase64)
		if err != nil {
			return nil, nil, fmt.Errorf("error deriving key from password: %v", err)
		}

		// Encrypt the derived key with the intermediate key from the backup code
		encDerivedKey, err := EncryptText(derived_key, base64.URLEncoding.EncodeToString(intermediateKey))
		if err != nil {
			return nil, nil, fmt.Errorf("error encrypting derived key: %v", err)
		}

		backupCodes[i] = code
		codeData[i]["password"] = hash
		codeData[i]["salt"] = saltBase64
		codeData[i]["enc_derived_key"] = encDerivedKey
	}

	return backupCodes, codeData, nil
}
