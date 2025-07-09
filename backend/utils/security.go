package utils

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
)

// Global logger
var Logger *log.Logger

func init() {
	// Initialize logger
	Logger = log.New(os.Stdout, "dailytxt: ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
}

// ContextKey is a type for context keys
type ContextKey string

// Context keys
const (
	UserIDKey     ContextKey = "userID"
	UsernameKey   ContextKey = "username"
	DerivedKeyKey ContextKey = "derivedKey"
)

// Settings holds the application settings
type AppSettings struct {
	DataPath        string   `json:"data_path"`
	Development     bool     `json:"development"`
	SecretToken     string   `json:"secret_token"`
	LogoutAfterDays int      `json:"logout_after_days"`
	AllowedHosts    []string `json:"allowed_hosts"`
	Indent          int      `json:"indent"`
}

// Global settings
var Settings AppSettings

// InitSettings loads the application settings
func InitSettings() error {
	// Default settings
	Settings = AppSettings{
		DataPath:        "/data",
		Development:     false,
		SecretToken:     generateSecretToken(),
		LogoutAfterDays: 30,
		AllowedHosts:    []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		Indent:          0,
	}

	fmt.Print("\nDetected following settings:\n================\n")

	// Override with environment variables if available
	if dataPath := os.Getenv("DATA_PATH"); dataPath != "" {
		Settings.DataPath = dataPath
	}
	fmt.Printf("Data Path: %s\n", Settings.DataPath)

	if os.Getenv("DEVELOPMENT") == "true" {
		Settings.Development = true
	}
	fmt.Printf("Development Mode: %t\n", Settings.Development)

	if secretToken := os.Getenv("SECRET_TOKEN"); secretToken != "" {
		Settings.SecretToken = secretToken
	}
	fmt.Printf("Secret Token: %s\n", Settings.SecretToken)

	if logoutDays := os.Getenv("LOGOUT_AFTER_DAYS"); logoutDays != "" {
		// Parse logoutDays to int
		var days int
		if _, err := fmt.Sscanf(logoutDays, "%d", &days); err == nil {
			Settings.LogoutAfterDays = days
		}
	}
	fmt.Printf("Logout After Days: %d\n", Settings.LogoutAfterDays)

	if indent := os.Getenv("INDENT"); indent != "" {
		// Parse indent to int
		var ind int
		if _, err := fmt.Sscanf(indent, "%d", &ind); err == nil {
			Settings.Indent = ind
		}
	}
	fmt.Printf("Indent: %d\n================\n\n", Settings.Indent)

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(Settings.DataPath, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %v", err)
	}

	return nil
}

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

// GenerateSecretToken generates a secure random token
func generateSecretToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		Logger.Fatalf("Failed to generate secret token: %v", err)
	}
	return base64.URLEncoding.EncodeToString(b)
}

// JSONResponse sends a JSON response with the given status code and data
func JSONResponse(w http.ResponseWriter, statusCode int, data any) {
	// Set content type
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	// Encode data to JSON
	var encoder *json.Encoder
	if Settings.Development && Settings.Indent > 0 {
		encoder = json.NewEncoder(w)
		encoder.SetIndent("", fmt.Sprintf("%*s", Settings.Indent, ""))
	} else {
		encoder = json.NewEncoder(w)
	}

	if err := encoder.Encode(data); err != nil {
		Logger.Printf("Error encoding JSON response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
