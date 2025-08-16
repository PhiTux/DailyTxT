package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
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
	DataPath          string   `json:"data_path"`
	Development       bool     `json:"development"`
	SecretToken       string   `json:"secret_token"`
	LogoutAfterDays   int      `json:"logout_after_days"`
	AllowedHosts      []string `json:"allowed_hosts"`
	Indent            int      `json:"indent"`
	AllowRegistration bool     `json:"allow_registration"`
}

// Global settings
var Settings AppSettings

// InitSettings loads the application settings
func InitSettings() error {
	// Default settings
	Settings = AppSettings{
		DataPath:          "/data",
		Development:       false,
		SecretToken:       GenerateSecretToken(),
		LogoutAfterDays:   30,
		AllowedHosts:      []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		Indent:            0,
		AllowRegistration: false,
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

	if allowedHosts := os.Getenv("ALLOWED_HOSTS"); allowedHosts != "" {
		// Split allowedHosts by comma and trim spaces
		hosts := strings.Split(allowedHosts, ",")
		for i, host := range hosts {
			hosts[i] = strings.TrimSpace(host)
		}
		Settings.AllowedHosts = hosts
	}
	fmt.Printf("Allowed Hosts: %v\n", Settings.AllowedHosts)

	if indent := os.Getenv("INDENT"); indent != "" {
		// Parse indent to int
		var ind int
		if _, err := fmt.Sscanf(indent, "%d", &ind); err == nil {
			Settings.Indent = ind
		}
	}
	fmt.Printf("Indent: %d\n", Settings.Indent)

	if allowRegistration := os.Getenv("ALLOW_REGISTRATION"); allowRegistration != "" {
		// Parse allowRegistration to bool
		if allowRegistration == "true" {
			Settings.AllowRegistration = true
		} else {
			Settings.AllowRegistration = false
		}
	}
	fmt.Printf("Allow Registration: %t\n", Settings.AllowRegistration)

	fmt.Print("================\n\n")

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(Settings.DataPath, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %v", err)
	}

	return nil
}

func GetUserIDByUsername(username string) (int, error) {
	// Get users
	users, err := GetUsers()
	if err != nil {
		return 0, fmt.Errorf("failed to get users: %v", err)
	}

	// Find user by username
	for _, user := range users {
		user, ok := user.(map[string]any)
		if !ok {
			continue // Skip if user is not a map
		}
		if user["username"] == username {
			if id, ok := user["id"].(int); ok {
				return id, nil
			}
			return 0, fmt.Errorf("user ID not found for username: %s", username)
		}
	}

	return 0, fmt.Errorf("user not found: %s", username)
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

// Move data to directory "old", if users.json is from dailytxt version 1
func HandleOldData(logger *log.Logger) {
	// Check if users.json exists
	usersFile := Settings.DataPath + "/users.json"
	if _, err := os.Stat(usersFile); os.IsNotExist(err) {
		logger.Println("No users.json found, skipping old data check.")
		return
	}

	// Read the file
	data, err := os.ReadFile(usersFile)
	if err != nil {
		logger.Printf("Error reading users.json: %v", err)
		return
	}

	// Check if the file is from dailytxt version 1
	var usersData map[string]interface{}
	if err := json.Unmarshal(data, &usersData); err != nil {
		logger.Printf("Error parsing users.json: %v", err)
		return
	}

	// Check if users array exists
	usersArray, ok := usersData["users"].([]interface{})
	if !ok || len(usersArray) == 0 {
		logger.Println("No users found in users.json, skipping migration.")
		return
	}

	// Check if any user is missing the dailytxt_version=2 field
	needsMigration := false
	for _, userInterface := range usersArray {
		user, ok := userInterface.(map[string]interface{})
		if !ok {
			continue
		}

		// Check if the version field exists and is 2
		version, exists := user["dailytxt_version"]
		if !exists || version != float64(2) {
			needsMigration = true
			logger.Printf("Found user without dailytxt_version=2: %s", user["username"])
			break
		}
	}

	// If no migration is needed, return
	if !needsMigration {
		logger.Println("All users have dailytxt_version=2, no migration needed.")
		return
	}

	// Create "old" directory
	oldDir := Settings.DataPath + "/old"
	if err := os.MkdirAll(oldDir, 0755); err != nil {
		logger.Printf("Error creating old directory: %v", err)
		return
	}

	// Move all files from data to old
	logger.Println("Moving all data to old directory...")

	// List all files and directories in the data path
	entries, err := os.ReadDir(Settings.DataPath)
	if err != nil {
		logger.Printf("Error reading data directory: %v", err)
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		// Skip the "old" directory itself
		if name == "old" {
			continue
		}

		srcPath := Settings.DataPath + "/" + name
		destPath := oldDir + "/" + name

		// Check if it's a directory or file
		info, err := os.Stat(srcPath)
		if err != nil {
			logger.Printf("Error getting info for %s: %v", srcPath, err)
			continue
		}

		if info.IsDir() {
			// For directories, copy recursively
			if err := CopyDir(srcPath, destPath, logger); err != nil {
				logger.Printf("Error copying directory %s to %s: %v", srcPath, destPath, err)
			} else {
				// Remove the original directory after successful copy
				if err := os.RemoveAll(srcPath); err != nil {
					logger.Printf("Error removing original directory %s: %v", srcPath, err)
				}
			}
		} else {
			// For files, copy directly
			if err := CopyFile(srcPath, destPath, logger); err != nil {
				logger.Printf("Error copying file %s to %s: %v", srcPath, destPath, err)
			} else {
				// Remove the original file after successful copy
				if err := os.Remove(srcPath); err != nil {
					logger.Printf("Error removing original file %s: %v", srcPath, err)
				}
			}
		}
	}

	logger.Println("All old data has been moved to " + oldDir + ". When logging in to old account, the migration will be started.\n")
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string, logger *log.Logger) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Copy the content
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Sync the file to ensure it's written to disk
	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination file: %w", err)
	}

	// Get the source file permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to get source file info: %w", err)
	}

	// Set the same permissions for the destination file
	if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set destination file permissions: %w", err)
	}

	logger.Printf("Copied file from %s to %s", src, dst)
	return nil
}

// CopyDir copies a directory recursively from src to dst
func CopyDir(src, dst string, logger *log.Logger) error {
	// Get source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to get source directory info: %w", err)
	}

	// Create destination directory with the same permissions
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Read source directory entries
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := src + "/" + entry.Name()
		dstPath := dst + "/" + entry.Name()

		// Get entry info
		entryInfo, err := os.Stat(srcPath)
		if err != nil {
			logger.Printf("Error getting info for %s: %v", srcPath, err)
			continue
		}

		// Copy directory or file
		if entryInfo.IsDir() {
			if err := CopyDir(srcPath, dstPath, logger); err != nil {
				logger.Printf("Error copying directory %s to %s: %v", srcPath, dstPath, err)
			}
		} else {
			if err := CopyFile(srcPath, dstPath, logger); err != nil {
				logger.Printf("Error copying file %s to %s: %v", srcPath, dstPath, err)
			}
		}
	}

	logger.Printf("Copied directory from %s to %s", src, dst)
	return nil
}

func GetUsernameByID(userID int) string {
	// Get users
	users, err := GetUsers()
	if err != nil {
		fmt.Printf("failed to get users: %v", err)
		return ""
	}

	// Find user by ID
	for _, userInterface := range users["users"].([]any) {
		user, ok := userInterface.(map[string]any)
		if !ok {
			continue // Skip if user is not a map
		}
		id := int(user["user_id"].(float64))
		if id == userID {
			if username, ok := user["username"].(string); ok {
				return username
			}
			fmt.Printf("username not found for user ID: %d\n", userID)
			return ""
		}
	}

	fmt.Printf("user not found with ID: %d\n", userID)
	return ""
}
