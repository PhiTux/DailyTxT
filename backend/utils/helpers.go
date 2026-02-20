package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Global logger
var Logger *log.Logger

// Application version (separate from AppSettings)
var AppVersion string

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
	DataPath            string   `json:"data_path"`
	Development         bool     `json:"development"`
	SecretToken         string   `json:"secret_token"`
	LogoutAfterDays     int      `json:"logout_after_days"`
	AllowedHosts        []string `json:"allowed_hosts"`
	Indent              int      `json:"indent"`
	AllowRegistration   bool     `json:"allow_registration"`
	BasePath            string   `json:"base_path"`
	ShareCodeTTLMinutes int      `json:"share_code_ttl_minutes"`
	ShareCookieDays     int      `json:"share_cookie_days"`
	SMTPHost            string   `json:"smtp_host"`
	SMTPPort            int      `json:"smtp_port"`
	SMTPUsername        string   `json:"smtp_username"`
	SMTPPassword        string   `json:"smtp_password"`
	SMTPFrom            string   `json:"smtp_from"`
}

// Global settings
var Settings AppSettings

// Registration override state (temporary window to allow registration)
var (
	registrationOverrideUntil time.Time
	registrationOverrideMu    sync.RWMutex
)

// SetRegistrationOverride opens registration for the given duration
func SetRegistrationOverride(d time.Duration) {
	registrationOverrideMu.Lock()
	defer registrationOverrideMu.Unlock()
	registrationOverrideUntil = time.Now().Add(d)
	Logger.Printf("Registration temporarily opened until %s", registrationOverrideUntil.Format(time.RFC3339))
}

func GetRegistrationOverrideUntil() time.Time {
	registrationOverrideMu.RLock()
	defer registrationOverrideMu.RUnlock()
	return registrationOverrideUntil
}

// IsRegistrationAllowed returns whether registration is
// overall allowed or temporarily allowed
func IsRegistrationAllowed() (bool, bool) {
	allowed := Settings.AllowRegistration

	registrationOverrideMu.RLock()
	until := registrationOverrideUntil
	registrationOverrideMu.RUnlock()
	tempAllowed := time.Now().Before(until)

	return allowed, tempAllowed
}

// SetVersion sets the application version
func SetVersion(version string) {
	AppVersion = version
}

// GetVersion returns the current application version
func GetVersion() string {
	return AppVersion
}

// InitSettings loads the application settings
func InitSettings() error {
	// Default settings
	Settings = AppSettings{
		DataPath:            "/data",
		Development:         false,
		SecretToken:         GenerateSecretToken(),
		LogoutAfterDays:     30,
		AllowedHosts:        []string{},
		Indent:              0,
		AllowRegistration:   false,
		BasePath:            "/",
		ShareCodeTTLMinutes: 10,
		ShareCookieDays:     30,
		SMTPPort:            587,
	}

	fmt.Print("\nDetected the following settings:\n================\n")

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

	if basePath := os.Getenv("BASE_PATH"); basePath != "" {
		Settings.BasePath = basePath
	}
	fmt.Printf("Base Path: %s\n", Settings.BasePath)

	if shareCodeTTL := os.Getenv("SHARE_CODE_TTL_MINUTES"); shareCodeTTL != "" {
		var minutes int
		if _, err := fmt.Sscanf(shareCodeTTL, "%d", &minutes); err == nil && minutes > 0 {
			Settings.ShareCodeTTLMinutes = minutes
		}
	}
	fmt.Printf("Share Code TTL Minutes: %d\n", Settings.ShareCodeTTLMinutes)

	if shareCookieDays := os.Getenv("SHARE_COOKIE_DAYS"); shareCookieDays != "" {
		var days int
		if _, err := fmt.Sscanf(shareCookieDays, "%d", &days); err == nil && days > 0 {
			Settings.ShareCookieDays = days
		}
	}
	fmt.Printf("Share Cookie Days: %d\n", Settings.ShareCookieDays)

	if smtpHost := os.Getenv("SMTP_HOST"); smtpHost != "" {
		Settings.SMTPHost = smtpHost
	}
	fmt.Printf("SMTP Host configured: %t\n", Settings.SMTPHost != "")

	if smtpPort := os.Getenv("SMTP_PORT"); smtpPort != "" {
		var port int
		if _, err := fmt.Sscanf(smtpPort, "%d", &port); err == nil && port > 0 {
			Settings.SMTPPort = port
		}
	}
	fmt.Printf("SMTP Port: %d\n", Settings.SMTPPort)

	if smtpUsername := os.Getenv("SMTP_USERNAME"); smtpUsername != "" {
		Settings.SMTPUsername = smtpUsername
	}
	fmt.Printf("SMTP Username configured: %t\n", Settings.SMTPUsername != "")

	if smtpPassword := os.Getenv("SMTP_PASSWORD"); smtpPassword != "" {
		Settings.SMTPPassword = smtpPassword
	}
	fmt.Printf("SMTP Password configured: %t\n", Settings.SMTPPassword != "")

	if smtpFrom := os.Getenv("SMTP_FROM"); smtpFrom != "" {
		Settings.SMTPFrom = smtpFrom
	}
	fmt.Printf("SMTP From: %s\n", Settings.SMTPFrom)

	fmt.Print("================\n\n")

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(Settings.DataPath, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %v", err)
	}

	return nil
}

func GetAppSettings() AppSettings {
	var tempSettings AppSettings

	data, _ := json.Marshal(Settings)
	json.Unmarshal(data, &tempSettings)

	// dont't show secret - remove it!
	tempSettings.SecretToken = ""
	return tempSettings
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

// Docker Hub API structures
type DockerHubTag struct {
	Name string `json:"name"`
}

type DockerHubTagsResponse struct {
	Results []DockerHubTag `json:"results"`
}

// Version cache
var (
	lastVersionCheck     time.Time
	cachedLatestVersion  string
	cachedLatestWithTest string
	versionCacheDuration = time.Hour
)

// parseVersion parses a semver string and returns major, minor, patch as integers
// Returns -1, -1, -1 if parsing fails
func parseVersion(version string) (int, int, int) {
	// Remove 'v' prefix if present
	version = strings.TrimPrefix(version, "v")

	// Split by '-' to separate version from pre-release identifiers
	parts := strings.Split(version, "-")
	if len(parts) == 0 {
		return -1, -1, -1
	}

	// Parse the main version part (e.g., "2.3.1")
	versionPart := parts[0]
	versionNumbers := strings.Split(versionPart, ".")

	if len(versionNumbers) != 3 {
		return -1, -1, -1
	}

	major, err1 := strconv.Atoi(versionNumbers[0])
	minor, err2 := strconv.Atoi(versionNumbers[1])
	patch, err3 := strconv.Atoi(versionNumbers[2])

	if err1 != nil || err2 != nil || err3 != nil {
		return -1, -1, -1
	}

	return major, minor, patch
}

// compareVersions compares two version strings
// Returns 1 if v1 > v2, -1 if v1 < v2, 0 if equal
func compareVersions(v1, v2 string) int {
	maj1, min1, pat1 := parseVersion(v1)
	maj2, min2, pat2 := parseVersion(v2)

	// If either version is invalid, treat it as lower
	if maj1 == -1 {
		if maj2 == -1 {
			return 0
		}
		return -1
	}
	if maj2 == -1 {
		return 1
	}

	// Compare major version
	if maj1 != maj2 {
		if maj1 > maj2 {
			return 1
		}
		return -1
	}

	// Compare minor version
	if min1 != min2 {
		if min1 > min2 {
			return 1
		}
		return -1
	}

	// Compare patch version
	if pat1 != pat2 {
		if pat1 > pat2 {
			return 1
		}
		return -1
	}

	return 0
}

// isStableVersion checks if a version is stable (no pre-release identifiers like "testing")
func isStableVersion(version string) bool {
	// Remove 'v' prefix if present
	version = strings.TrimPrefix(version, "v")

	// Convert to lowercase for case-insensitive search
	lowerVersion := strings.ToLower(version)

	// Check if the version contains "test" (part of "testing", "test", etc.)
	if strings.Contains(lowerVersion, "test") {
		return false
	}

	return true
}

// GetLatestVersion fetches the latest version information from Docker Hub
// Returns (latest_stable_version, latest_version_including_testing)
func GetLatestVersion() (string, string) {
	// Check if we have cached data that's still fresh
	if time.Since(lastVersionCheck) < versionCacheDuration && cachedLatestVersion != "" {
		return cachedLatestVersion, cachedLatestWithTest
	}

	// Fetch tags from Docker Hub
	resp, err := http.Get("https://hub.docker.com/v2/repositories/phitux/dailytxt/tags")
	if err != nil {
		Logger.Printf("Error fetching Docker Hub tags: %v", err)
		// Return cached values if available, otherwise empty
		return cachedLatestVersion, cachedLatestWithTest
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		Logger.Printf("Docker Hub API returned status %d", resp.StatusCode)
		return cachedLatestVersion, cachedLatestWithTest
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		Logger.Printf("Error reading Docker Hub response: %v", err)
		return cachedLatestVersion, cachedLatestWithTest
	}

	var tagsResponse DockerHubTagsResponse
	if err := json.Unmarshal(body, &tagsResponse); err != nil {
		Logger.Printf("Error parsing Docker Hub response: %v", err)
		return cachedLatestVersion, cachedLatestWithTest
	}

	var latestStable, latestOverall string

	// Process all tags
	for _, tag := range tagsResponse.Results {
		tagName := tag.Name

		// Skip non-version tags like "latest"
		if !strings.Contains(tagName, ".") {
			continue
		}

		// Check if this is a valid semver-like version
		maj, min, pat := parseVersion(tagName)
		if maj == -1 || min == -1 || pat == -1 {
			continue
		}

		// Update latest overall version
		if latestOverall == "" || compareVersions(tagName, latestOverall) > 0 {
			latestOverall = tagName
		}

		// Update latest stable version (only if it's stable)
		if isStableVersion(tagName) {
			if latestStable == "" || compareVersions(tagName, latestStable) > 0 {
				latestStable = tagName
			}
		}
	}

	// Update cache
	lastVersionCheck = time.Now()
	cachedLatestVersion = latestStable
	cachedLatestWithTest = latestOverall

	return latestStable, latestOverall
}

// GetVersionInfo returns the current application version (public endpoint, no auth required)
func GetVersionInfo(w http.ResponseWriter, r *http.Request) {
	latest_stable, latest_overall := GetLatestVersion()

	JSONResponse(w, http.StatusOK, map[string]string{
		"current_version":        GetVersion(),
		"latest_stable_version":  latest_stable,
		"latest_overall_version": latest_overall,
	})
}
