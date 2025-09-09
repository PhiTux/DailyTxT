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

// GetAllUsers returns all users with their disk usage
func GetAllUsers(w http.ResponseWriter, r *http.Request) {
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"users":      adminUsers,
		"free_space": freeSpace,
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
