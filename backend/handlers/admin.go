package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/phitux/dailytxt/backend/utils"
)

type AdminRequest struct {
	AdminPassword string `json:"admin_password"`
}

type AdminUserResponse struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	DiskUsage int64  `json:"disk_usage"`
}

// ValidateAdminPassword validates the admin password
func ValidateAdminPassword(w http.ResponseWriter, r *http.Request) {
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		json.NewEncoder(w).Encode(map[string]bool{
			"valid": false,
		})
		return
	}

	var req struct {
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{
		"valid": req.Password == adminPassword,
	})
}

// validateAdminPasswordInRequest checks if the admin password in request is valid
func validateAdminPasswordInRequest(r *http.Request) bool {
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		return false
	}

	var req AdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return false
	}

	return req.AdminPassword == adminPassword
}

// GetAdminData returns:
// - all users with their disk usage
// - free disk space
// - migration-info
// - app settings (env-vars)
func GetAdminData(w http.ResponseWriter, r *http.Request) {
	if !validateAdminPasswordInRequest(r) {
		http.Error(w, "Invalid admin password", http.StatusUnauthorized)
		return
	}

	// Read users.json
	users, err := utils.GetUsers()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// get users
	usersList, ok := users["users"].([]any)
	if !ok || len(usersList) == 0 {
		utils.Logger.Printf("No users found.")
	}

	// Calculate disk usage for each user
	var adminUsers []AdminUserResponse
	for _, u := range usersList {
		user, ok := u.(map[string]any)
		if !ok {
			continue
		}

		userID, ok := user["user_id"].(float64)
		if !ok {
			continue
		}

		username, _ := user["username"].(string)

		// Calculate disk usage for this user
		diskUsage := calculateUserDiskUsage(int(userID))

		adminUsers = append(adminUsers, AdminUserResponse{
			ID:        int(userID),
			Username:  username,
			DiskUsage: diskUsage,
		})
	}

	// Get free disk space
	freeSpace, err := getFreeDiskSpace()
	if err != nil {
		log.Printf("Error getting free disk space: %v", err)
		freeSpace = 0 // Default to 0 if we can't determine free space
	}

	// Check for old directory and get old users info
	oldDirInfo := getOldDirectoryInfo()

	// Get App Settings (Env-vars)
	appSettings := utils.GetAppSettings()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"users":        adminUsers,
		"free_space":   freeSpace,
		"old_data":     oldDirInfo,
		"app_settings": appSettings,
	})
}

// calculateUserDiskUsage calculates the total disk usage for a user
func calculateUserDiskUsage(userID int) int64 {
	userDataDir := filepath.Join(utils.Settings.DataPath, strconv.Itoa(userID))
	var totalSize int64

	// Calculate size recursively
	err := filepath.Walk(userDataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on errors
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		log.Printf("Error calculating disk usage for user %d: %v", userID, err)
	}

	return totalSize
}

// getFreeDiskSpace calculates the free disk space for the data directory
func getFreeDiskSpace() (int64, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(utils.Settings.DataPath, &stat)
	if err != nil {
		return 0, err
	}

	// Available space = block size * available blocks
	freeSpace := int64(stat.Bavail) * int64(stat.Bsize)
	return freeSpace, nil
}

// getOldDirectoryInfo checks for old directory and returns info about old users and directory size
func getOldDirectoryInfo() map[string]any {
	oldDirPath := filepath.Join(utils.Settings.DataPath, "old")

	// Check if old directory exists
	if _, err := os.Stat(oldDirPath); os.IsNotExist(err) {
		return map[string]any{
			"exists": false,
		}
	}

	// Read users.json from old directory
	usersJsonPath := filepath.Join(oldDirPath, "users.json")
	var oldUsernames []string

	if _, err := os.Stat(usersJsonPath); err == nil {
		// Read and parse users.json
		data, err := os.ReadFile(usersJsonPath)
		if err == nil {
			var usersData map[string]any
			if json.Unmarshal(data, &usersData) == nil {
				if usersList, ok := usersData["users"].([]any); ok {
					for _, u := range usersList {
						if user, ok := u.(map[string]any); ok {
							if username, ok := user["username"].(string); ok {
								oldUsernames = append(oldUsernames, username)
							}
						}
					}
				}
			}
		}
	}

	// Calculate total size of old directory
	var totalSize int64
	filepath.Walk(oldDirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on errors
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	return map[string]any{
		"exists":     true,
		"usernames":  oldUsernames,
		"total_size": totalSize,
	}
}

// DeleteUser deletes a user and all their data
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AdminPassword string `json:"admin_password"`
		UserID        int    `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate admin password
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" || req.AdminPassword != adminPassword {
		http.Error(w, "Invalid admin password", http.StatusUnauthorized)
		return
	}

	// Use the shared delete function from users.go
	if err := deleteUserByID(req.UserID); err != nil {
		log.Printf("Error deleting user %d: %v", req.UserID, err)
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Error deleting user", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{
		"success": true,
	})
}

// DeleteOldData deletes the entire old directory
func DeleteOldData(w http.ResponseWriter, r *http.Request) {
	if !validateAdminPasswordInRequest(r) {
		http.Error(w, "Invalid admin password", http.StatusUnauthorized)
		return
	}

	oldDirPath := filepath.Join(utils.Settings.DataPath, "old")

	// Check if old directory exists
	if _, err := os.Stat(oldDirPath); os.IsNotExist(err) {
		http.Error(w, "Old directory does not exist", http.StatusNotFound)
		return
	}

	// Remove the entire old directory
	if err := os.RemoveAll(oldDirPath); err != nil {
		log.Printf("Error deleting old directory: %v", err)
		http.Error(w, "Error deleting old directory", http.StatusInternalServerError)
		return
	}

	log.Printf("Old directory successfully deleted by admin (id: %d, username: %s)", r.Context().Value(utils.UserIDKey), r.Context().Value(utils.UsernameKey))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{
		"success": true,
	})
}

// OpenRegistrationTemp allows admin to open registration for a limited time window
func OpenRegistrationTemp(w http.ResponseWriter, r *http.Request) {
	// Decode request (admin password + optional seconds)
	var req struct {
		AdminPassword string `json:"admin_password"`
		Seconds       int    `json:"seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" || req.AdminPassword != adminPassword {
		http.Error(w, "Invalid admin password", http.StatusUnauthorized)
		return
	}

	// Default duration 5 minutes; optionally allow custom seconds (max 15 min)
	duration := 5 * 60 // seconds
	if req.Seconds > 0 && req.Seconds <= 15*60 {
		duration = req.Seconds
	}

	utils.SetRegistrationOverride(time.Duration(duration) * time.Second)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success":  true,
		"until":    time.Now().Add(time.Duration(duration) * time.Second).Format(time.RFC3339),
		"duration": duration,
	})
}
