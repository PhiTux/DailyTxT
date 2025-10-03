package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/phitux/dailytxt/backend/utils"
)

// LoginRequest represents the login request body
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login handles user login
func Login(w http.ResponseWriter, r *http.Request) {
	// Parse the request body
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get users
	users, err := utils.GetUsers()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Check if user exists
	usersList, ok := users["users"].([]any)
	if !ok || len(usersList) == 0 {
		utils.Logger.Printf("Login failed. User '%s' not found", req.Username)
	}

	// Find user
	var userID int
	found := false
	var username string

	for _, u := range usersList {
		user, ok := u.(map[string]any)
		if !ok {
			continue
		}

		if username, ok = user["username"].(string); ok && strings.EqualFold(username, req.Username) {
			found = true
			if id, ok := user["user_id"].(float64); ok {
				userID = int(id)
			}
			break
		}
	}

	if !found {
		// Try to find user in old data
		oldUsers, err := utils.GetOldUsers()
		if err != nil {
			utils.Logger.Printf("Error accessing old users: %v", err)
			http.Error(w, "User/Password combination not found", http.StatusNotFound)
			return
		}

		oldUsersList, ok := oldUsers["users"].([]any)
		if !ok || len(oldUsersList) == 0 {
			utils.Logger.Printf("Login failed. User '%s' not found in new or old data", req.Username)
			http.Error(w, "User/Password combination not found", http.StatusNotFound)
			return
		}

		// Find user in old data
		var oldUser map[string]any
		for _, u := range oldUsersList {
			user, ok := u.(map[string]any)
			if !ok {
				continue
			}

			if username, ok = user["username"].(string); ok && strings.EqualFold(username, req.Username) {
				oldUser = user
				break
			}
		}

		if oldUser == nil {
			utils.Logger.Printf("Login failed. User '%s' not found in new or old data", req.Username)
			http.Error(w, "User/Password combination not found", http.StatusNotFound)
			return
		}

		// Get password
		oldHashedPassword, ok := oldUser["password"].(string)
		if !ok {
			utils.Logger.Printf("Login failed. Password not found for '%s'", req.Username)
			http.Error(w, "User/Password combination not found", http.StatusNotFound)
			return
		}

		// Verify old password
		if !utils.VerifyOldPassword(req.Password, oldHashedPassword) {
			utils.Logger.Printf("Login failed. Old password for user '%s' is incorrect", req.Username)
			http.Error(w, "User/Password combination not found", http.StatusNotFound)
			return
		}

		// Start migration
		utils.Logger.Printf("User '%s' found in old data. Starting migration...", req.Username)

		// Check if there is already a migration in progress for this user
		activeMigrationsMutex.RLock()
		isActive := activeMigrations[username]
		activeMigrationsMutex.RUnlock()

		if isActive {
			utils.Logger.Printf("Migration already in progress for user '%s'. Rejecting second attempt.", req.Username)
			utils.JSONResponse(w, http.StatusConflict, map[string]any{
				"error": "Migration already in progress for this user. Please wait until it completes.",
			})
			return
		}

		// Mark this user as having an active migration
		activeMigrationsMutex.Lock()
		activeMigrations[username] = true
		activeMigrationsMutex.Unlock()

		// Create a channel to report progress
		progressChan := make(chan utils.MigrationProgress, 10)

		// Start migration in a goroutine
		go func() {
			defer close(progressChan)

			// Update progress channel to track migration progress
			go func() {
				for progress := range progressChan {
					migrationProgressMutex.Lock()
					// Convert from utils.MigrationProgress to handlers.MigrationProgress
					migrationProgress[username] = MigrationProgress{
						Phase:          progress.Phase,
						ProcessedItems: progress.ProcessedItems,
						TotalItems:     progress.TotalItems,
						ErrorCount:     progress.ErrorCount,
					}
					migrationProgressMutex.Unlock()
				}
			}()

			utils.Logger.Printf("Starting migration for user '%s'", username)

			err := utils.MigrateUserData(username, req.Password, Register, progressChan)
			if err != nil {
				utils.Logger.Printf("Migration failed for user '%s': %v", username, err)
				// Mark migration as completed even on error
				activeMigrationsMutex.Lock()
				activeMigrations[username] = false
				activeMigrationsMutex.Unlock()
				return
			}

			// Mark migration as completed
			activeMigrationsMutex.Lock()
			activeMigrations[username] = false
			activeMigrationsMutex.Unlock()
		}()

		// Return migration status to client
		utils.JSONResponse(w, http.StatusAccepted, map[string]any{
			"migration_started": true,
			"username":          username,
		})
		return
	}

	derivedKey, availableBackupCodes, err := utils.CheckPasswordForUser(userID, req.Password)
	if err != nil {
		utils.Logger.Printf("Error checking password for user '%s': %v", req.Username, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	} else if derivedKey == "" {
		utils.Logger.Printf("Login failed. Password for user '%s' is incorrect", req.Username)
		http.Error(w, "User/Password combination not found", http.StatusNotFound)
		return
	}

	// Create JWT token
	token, err := utils.GenerateToken(userID, username, derivedKey)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		Expires:  time.Now().Add(time.Duration(utils.Settings.LogoutAfterDays) * 24 * time.Hour),
	})

	// Return success
	utils.JSONResponse(w, http.StatusOK, map[string]any{
		"migration_started":      false,
		"username":               username,
		"available_backup_codes": availableBackupCodes,
	})
}

func IsRegistrationAllowed(w http.ResponseWriter, r *http.Request) {

	// Check if registration is allowed (consider env and temporary override)
	allowed, tempAllowed := utils.IsRegistrationAllowed()
	utils.JSONResponse(w, http.StatusOK, map[string]any{
		"registration_allowed": allowed,
		"temporary_allowed":    tempAllowed,
		"until":                utils.GetRegistrationOverrideUntil(),
	})
}

// RegisterRequest represents the register request body
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Register handles user registration
// The API endpoint
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if allowed, temporary := utils.IsRegistrationAllowed(); !allowed && !temporary {
		http.Error(w, "Registration is not allowed", http.StatusForbidden)
		return
	}

	// Parse the request body
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := Register(req.Username, req.Password)
	if err != nil {
		http.Error(w, fmt.Sprintf("Internal Server Error: %v", err), http.StatusInternalServerError)
		return
	}

	utils.JSONResponse(w, http.StatusOK, map[string]bool{
		"success": result,
	})
}

// The actual register function (can also be called from migration)
func Register(username string, password string) (bool, error) {
	utils.UsersFileMutex.Lock()
	defer utils.UsersFileMutex.Unlock()

	// Get users
	users, err := utils.GetUsers()
	if err != nil {
		return false, fmt.Errorf("internal Server Error: %d", http.StatusInternalServerError)
	}

	// Check if username already exists
	if len(users) > 0 {
		usersList, ok := users["users"].([]any)
		if !ok {
			utils.Logger.Printf("users.json is not in the correct format. Key 'users' is missing or not a list.")
			return false, fmt.Errorf("users.json is not in the correct format: %d", http.StatusInternalServerError)
		}

		for _, u := range usersList {
			user, ok := u.(map[string]any)
			if !ok {
				continue
			}

			if username_from_file, ok := user["username"].(string); ok && strings.EqualFold(username_from_file, username) {
				utils.Logger.Printf("Registration failed. Username '%s' already exists", username)
				return false, fmt.Errorf("username already exists (error %d)", http.StatusBadRequest)
			}
		}
	}

	// Create new user data
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return false, fmt.Errorf("internal Server Error: %d", http.StatusInternalServerError)
	}

	// Generate a random salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return false, fmt.Errorf("internal Server Error: %d", http.StatusInternalServerError)
	}
	// Convert salt to base64
	saltBase64 := base64.StdEncoding.EncodeToString(salt)

	// Create encryption key
	derivedKey, err := utils.DeriveKeyFromPassword(password, saltBase64)
	if err != nil {
		return false, fmt.Errorf("internal Server Error: %d", http.StatusInternalServerError)
	}

	// Generate a new random encryption key
	encryptionKey := make([]byte, 32)
	if _, err := rand.Read(encryptionKey); err != nil {
		return false, fmt.Errorf("internal Server Error: %d", http.StatusInternalServerError)
	}

	// Encrypt the encryption key with the derived key
	aead, err := utils.CreateAEAD(derivedKey)
	if err != nil {
		return false, fmt.Errorf("internal Server Error: %d", http.StatusInternalServerError)
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return false, fmt.Errorf("internal Server Error: %d", http.StatusInternalServerError)
	}

	encryptedKey := aead.Seal(nonce, nonce, encryptionKey, nil)
	encEncKey := base64.StdEncoding.EncodeToString(encryptedKey)

	// Create or update users
	if len(users) == 0 {
		users = map[string]any{
			"id_counter": 1,
			"users": []map[string]any{
				{
					"user_id":          1,
					"dailytxt_version": 2,
					"username":         username,
					"password":         hashedPassword,
					"salt":             salt,
					"enc_enc_key":      encEncKey,
				},
			},
		}
	} else {
		// Increment ID counter
		idCounter, ok := users["id_counter"].(float64)
		if !ok {
			idCounter = 1
		}
		idCounter++
		users["id_counter"] = idCounter

		// Add new user
		usersList, ok := users["users"].([]any)
		if !ok {
			usersList = []any{}
		}

		usersList = append(usersList, map[string]any{
			"user_id":          int(idCounter),
			"dailytxt_version": 2,
			"username":         username,
			"password":         hashedPassword,
			"salt":             salt,
			"enc_enc_key":      encEncKey,
		})

		users["users"] = usersList
	}

	// Write users to file
	if err := utils.WriteUsers(users); err != nil {
		return false, fmt.Errorf("internal Server Error when trying to write users.json: %d", http.StatusInternalServerError)
	}

	// Return success
	return true, nil
}

// Logout handles user logout
func Logout(w http.ResponseWriter, r *http.Request) {
	// Delete token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})

	// Return success
	utils.JSONResponse(w, http.StatusOK, map[string]bool{
		"success": true,
	})
}

// CheckLogin checks if user is logged in
func CheckLogin(w http.ResponseWriter, r *http.Request) {
	// Get token from cookie
	cookie, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Validate JWT token
	claims, err := utils.ValidateToken(cookie.Value)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Return user info
	utils.JSONResponse(w, http.StatusOK, map[string]any{
		"user_id":  claims.UserID,
		"username": claims.Username,
	})
}

func GetDefaultSettings() map[string]any {
	// Default settings
	return map[string]any{
		"autoloadImagesByDefault":    false,
		"setAutoloadImagesPerDevice": true,
		"useALookBack":               true,
		"aLookBackYears":             []int{1, 5, 10},
		"useBrowserTimezone":         true,
		"timezone":                   "UTC",
		"useBrowserLanguage":         true,
		"language":                   "en",
		"darkModeAutoDetect":         true,
		"useDarkMode":                false,
		"background":                 "gradient",
		"monochromeBackgroundColor":  "#ececec",
		"checkForUpdates":            true,
		"includeTestVersions":        false,
		"requirePasswordOnPageLoad":  false,
	}
}

// GetUserSettings retrieves user settings
func GetUserSettings(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value(utils.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get derived key from context
	derivedKey, ok := r.Context().Value(utils.DerivedKeyKey).(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user settings
	encryptedSettings, err := utils.GetUserSettings(userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving user settings: %v", err), http.StatusInternalServerError)
		return
	}

	// Default settings
	defaultSettings := GetDefaultSettings()

	// If no settings found, return defaults
	if len(encryptedSettings) == 0 {
		utils.JSONResponse(w, http.StatusOK, defaultSettings)
		return
	}

	// Decrypt settings
	encKey, err := utils.GetEncryptionKey(userID, derivedKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting encryption key: %v", err), http.StatusInternalServerError)
		return
	}

	decryptedSettings, err := utils.DecryptText(encryptedSettings, encKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error decrypting settings: %v", err), http.StatusInternalServerError)
		return
	}

	// Parse JSON
	var settings map[string]any
	if err := json.Unmarshal([]byte(decryptedSettings), &settings); err != nil {
		http.Error(w, fmt.Sprintf("Error parsing settings: %v", err), http.StatusInternalServerError)
		return
	}

	// Apply defaults for missing keys
	for key, value := range defaultSettings {
		if _, exists := settings[key]; !exists {
			settings[key] = value
		}
	}

	// Return settings
	utils.JSONResponse(w, http.StatusOK, settings)
}

// SaveUserSettings saves user settings
func SaveUserSettings(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value(utils.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get derived key from context
	derivedKey, ok := r.Context().Value(utils.DerivedKeyKey).(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var newSettings map[string]any
	if err := json.NewDecoder(r.Body).Decode(&newSettings); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get existing settings
	encryptedSettings, err := utils.GetUserSettings(userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving user settings: %v", err), http.StatusInternalServerError)
		return
	}

	// Get encryption key
	encKey, err := utils.GetEncryptionKey(userID, derivedKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting encryption key: %v", err), http.StatusInternalServerError)
		return
	}

	// Current settings
	var currentSettings map[string]any

	// If settings exist, decrypt them
	if len(encryptedSettings) > 0 {
		decryptedSettings, err := utils.DecryptText(encryptedSettings, encKey)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error decrypting settings: %v", err), http.StatusInternalServerError)
			return
		}

		// Parse JSON
		if err := json.Unmarshal([]byte(decryptedSettings), &currentSettings); err != nil {
			http.Error(w, fmt.Sprintf("Error parsing settings: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// If no settings or empty, use defaults
	if len(currentSettings) == 0 {
		currentSettings = GetDefaultSettings()
	}

	// Update settings
	maps.Copy(currentSettings, newSettings)

	// Encrypt settings
	settingsJSON, err := json.Marshal(currentSettings)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error encoding settings: %v", err), http.StatusInternalServerError)
		return
	}

	encryptedNewSettings, err := utils.EncryptText(string(settingsJSON), encKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error encrypting settings: %v", err), http.StatusInternalServerError)
		return
	}

	// Write settings
	if err := utils.WriteUserSettings(userID, encryptedNewSettings); err != nil {
		http.Error(w, fmt.Sprintf("Error writing settings: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success
	utils.JSONResponse(w, http.StatusOK, map[string]bool{
		"success": true,
	})
}

// MigrationProgress stores the progress of user data migration
type MigrationProgress struct {
	Phase          string `json:"phase"`           // Current migration phase
	ProcessedItems int    `json:"processed_items"` // Number of items processed
	TotalItems     int    `json:"total_items"`     // Total number of items to process
	ErrorCount     int    `json:"error_count"`     // Number of errors encountered during migration
}

// migrationProgress keeps track of migration progress for all users
var migrationProgress = make(map[string]MigrationProgress)
var migrationProgressMutex sync.Mutex
var activeMigrations = make(map[string]bool)
var activeMigrationsMutex sync.RWMutex

// GetMigrationProgress returns the migration progress for a user
func GetMigrationProgress(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		utils.Logger.Printf("username: %s", username)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get migration progress
	migrationProgressMutex.Lock()
	progress, exists := migrationProgress[username]
	migrationProgressMutex.Unlock()

	// Check if migration is actually active
	activeMigrationsMutex.RLock()
	isActive := activeMigrations[username]
	activeMigrationsMutex.RUnlock()

	if !exists {
		utils.JSONResponse(w, http.StatusOK, map[string]any{
			"migration_in_progress": false,
			"progress":              map[string]string{"phase": "not_started"},
		})
		return
	}

	// Check if migration is completed
	migrationCompleted := progress.Phase == "completed"

	utils.JSONResponse(w, http.StatusOK, map[string]any{
		"migration_in_progress": isActive && !migrationCompleted,
		"progress":              progress,
	})
}

// ChangePasswordRequest represents the change password request body
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// ChangePassword changes the user's password
func ChangePassword(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value(utils.UserIDKey).(int)
	if !ok {
		utils.JSONResponse(w, http.StatusUnauthorized, map[string]any{
			"success": false,
			"message": "User not authenticated",
		})
		return
	}

	// Get derived key from context
	_, ok = r.Context().Value(utils.DerivedKeyKey).(string)
	if !ok {
		utils.JSONResponse(w, http.StatusUnauthorized, map[string]any{
			"success": false,
			"message": "User not authenticated",
		})
		return
	}

	// Parse request body
	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSONResponse(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"message": "Invalid request body",
		})
		return
	}

	derivedKey, availableBackupCodes, err := utils.CheckPasswordForUser(userID, req.OldPassword)
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"message": fmt.Sprintf("Error checking old password: %v", err),
		})
		return
	} else if len(derivedKey) == 0 {
		utils.JSONResponse(w, http.StatusOK, map[string]any{
			"success":                false,
			"message":                "Old password is incorrect",
			"password_incorrect":     true,
			"available_backup_codes": availableBackupCodes,
		})
		return
	}

	utils.UsersFileMutex.Lock()
	defer utils.UsersFileMutex.Unlock()

	// Get user data
	users, err := utils.GetUsers()
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"message": fmt.Sprintf("Error retrieving users: %v", err),
		})
		return
	}

	usersList, ok := users["users"].([]any)
	if !ok {
		utils.JSONResponse(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"message": "Users data is not in the correct format",
		})
		return
	}

	var user map[string]any
	for _, u := range usersList {
		uMap, ok := u.(map[string]any)
		if !ok {
			continue
		}
		if id, ok := uMap["user_id"].(float64); ok && int(id) == userID {
			user = uMap
			break
		}
	}

	if user == nil {
		utils.JSONResponse(w, http.StatusNotFound, map[string]any{
			"success": false,
			"message": "User not found",
		})
		return
	}

	newHashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"message": fmt.Sprintf("Error hashing new password: %v", err),
		})
		return
	}

	// Update user data
	user["password"] = newHashedPassword

	// Decrypt the existing encryption key
	// Get encryption key
	encKey, err := utils.GetEncryptionKey(userID, derivedKey)
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"message": fmt.Sprintf("Error getting encryption key: %v", err),
		})
		return
	}
	encKeyBytes, err := base64.URLEncoding.DecodeString(encKey)
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"message": fmt.Sprintf("Error decoding encryption key: %v", err),
		})
		return
	}

	// Re-Encrypt the encryption key with the new password
	// Generate a random salt
	salt := make([]byte, 16)
	_, err = rand.Read(salt)
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"message": fmt.Sprintf("Error generating salt: %v", err),
		})
		return
	}

	saltBase64 := base64.StdEncoding.EncodeToString(salt)
	newDerivedKey, err := utils.DeriveKeyFromPassword(req.NewPassword, saltBase64)
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"message": fmt.Sprintf("Error deriving new key from password: %v", err),
		})
		return
	}

	// Encrypt the encryption key with the new derived key
	aead, err := utils.CreateAEAD(newDerivedKey)
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"message": fmt.Sprintf("Error creating AEAD: %v", err),
		})
		return
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"message": fmt.Sprintf("Error generating nonce: %v", err),
		})
		return
	}
	encryptedKey := aead.Seal(nonce, nonce, encKeyBytes, nil)
	encEncKey := base64.StdEncoding.EncodeToString(encryptedKey)

	// Update user data with new salt and encrypted key
	user["salt"] = saltBase64
	user["enc_enc_key"] = encEncKey

	// Remove backup codes if they exist
	user["backup_codes"] = []any{}

	// Update users data
	for i, u := range usersList {
		if uMap, ok := u.(map[string]any); ok && uMap["user_id"] == userID {
			usersList[i] = user
			break
		}
	}
	users["users"] = usersList
	// Write updated users data to file
	if err := utils.WriteUsers(users); err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"message": fmt.Sprintf("Error writing users data: %v", err),
		})
		return
	}

	// create new JWT token with updated derived key
	token, err := utils.GenerateToken(userID, user["username"].(string), base64.StdEncoding.EncodeToString(newDerivedKey))
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"message": fmt.Sprintf("Error generating new token: %v", err),
		})
		return
	}

	// Set new token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		Expires:  time.Now().Add(time.Duration(utils.Settings.LogoutAfterDays) * 24 * time.Hour),
	})

	// Return success and return a new cookie
	utils.JSONResponse(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Password changed successfully",
	})
}

type DeleteAccountRequest struct {
	Password string `json:"password"`
}

// deleteUserByID deletes a user by their user ID from the system
// This is a shared function used by both DeleteAccount and admin DeleteUser
func deleteUserByID(userID int) error {
	utils.UsersFileMutex.Lock()
	defer utils.UsersFileMutex.Unlock()

	// Get User data
	users, err := utils.GetUsers()
	if err != nil {
		return fmt.Errorf("error retrieving users: %v", err)
	}
	usersList, ok := users["users"].([]any)
	if !ok {
		return fmt.Errorf("users data is not in the correct format")
	}

	// Remove user from users list
	var newUsersList []any
	userFound := false
	for _, u := range usersList {
		uMap, ok := u.(map[string]any)
		if !ok {
			continue
		}
		// Keep all users, except the one with the same user_id
		if id, ok := uMap["user_id"].(float64); !ok || int(id) != userID {
			utils.Logger.Printf("Keeping user with ID %f (%d)", id, userID)
			newUsersList = append(newUsersList, u)
		} else {
			userFound = true
		}
	}

	if !userFound {
		return fmt.Errorf("user with ID %d not found", userID)
	}

	users["users"] = newUsersList

	if err := utils.WriteUsers(users); err != nil {
		return fmt.Errorf("error writing users data: %v", err)
	}

	// Delete directory of the user with all his data
	if err := utils.DeleteUserData(userID); err != nil {
		return fmt.Errorf("error deleting user data of ID %d: %v", userID, err)
	}

	return nil
}

func DeleteAccount(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value(utils.UserIDKey).(int)
	if !ok {
		utils.JSONResponse(w, http.StatusUnauthorized, map[string]any{
			"success": false,
			"message": "User not authenticated",
		})
		return
	}

	// Parse request body
	var req DeleteAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSONResponse(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"message": "Invalid request body",
		})
		return
	}

	derived_key, _, err := utils.CheckPasswordForUser(userID, req.Password)
	if err != nil || len(derived_key) == 0 {
		utils.JSONResponse(w, http.StatusOK, map[string]any{
			"success":            false,
			"message":            "Error checking password",
			"password_incorrect": true,
		})
		return
	}

	// Use the shared delete function
	if err := deleteUserByID(userID); err != nil {
		utils.JSONResponse(w, http.StatusOK, map[string]any{
			"success": false,
			"message": err.Error(),
		})
		utils.Logger.Printf("Error deleting user account ID %d: %v", userID, err)
		return
	}

	utils.JSONResponse(w, http.StatusOK, map[string]any{
		"success": true,
	})
}

type CreateBackupCodesRequest struct {
	Password string `json:"password"`
}

// CreateBackupCodes creates 6 random backup codes for the user
// Those are storing the encrypted derived key from the original password!
func CreateBackupCodes(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value(utils.UserIDKey).(int)
	if !ok {
		utils.JSONResponse(w, http.StatusUnauthorized, map[string]any{
			"success": false,
			"message": "User not authenticated",
		})
		return
	}

	// Parse request body
	var req CreateBackupCodesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSONResponse(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"message": "Invalid request body",
		})
		return
	}

	// Check if password is correct
	derivedKey, backup_codes, err := utils.CheckPasswordForUser(userID, req.Password)
	if err != nil || len(derivedKey) == 0 {
		utils.Logger.Printf("Error checking password for user %d: %v", userID, err)

		utils.JSONResponse(w, http.StatusOK, map[string]any{
			"success": false,
			"message": "Error checking password",
		})
		return
	}

	// otherwise, we have the correct password

	// Generate backup codes
	codes, codeData, err := utils.GenerateBackupCodes(derivedKey)
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"message": fmt.Sprintf("Error generating backup codes: %v", err),
		})
		return
	}

	// Save backup codes to file
	if err := utils.SaveBackupCodes(userID, codeData); err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"message": fmt.Sprintf("Error saving backup codes: %v", err),
		})
		return
	}

	available_backup_codes := len(codes)
	if backup_codes == -1 {
		available_backup_codes = -1
	}

	utils.JSONResponse(w, http.StatusOK, map[string]any{
		"success":                true,
		"backup_codes":           codes,
		"available_backup_codes": available_backup_codes,
	})
}

// ChangeUsername handles changing a user's username
func ChangeUsername(w http.ResponseWriter, r *http.Request) {
	// Get user info from context
	userID, ok := r.Context().Value(utils.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request
	var req struct {
		NewUsername string `json:"new_username"`
		Password    string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	req.NewUsername = strings.TrimSpace(req.NewUsername)
	if req.NewUsername == "" {
		utils.JSONResponse(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"message": "Username cannot be empty",
		})
		return
	}

	// check password
	derivedKey, availableBackupCodes, err := utils.CheckPasswordForUser(userID, req.Password)
	if err != nil || len(derivedKey) == 0 {
		utils.JSONResponse(w, http.StatusOK, map[string]any{
			"success":            false,
			"password_incorrect": true,
		})
		return
	}

	// Get users
	users, err := utils.GetUsers()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	usersList, ok := users["users"].([]any)
	if !ok {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Check if new username is already taken (case-insensitive)
	for _, u := range usersList {
		user, ok := u.(map[string]any)
		if !ok {
			continue
		}

		existingUsername, ok := user["username"].(string)
		if !ok {
			continue
		}

		// Skip current user
		if int(user["user_id"].(float64)) == userID {
			continue
		}

		// Case-insensitive comparison
		if strings.EqualFold(existingUsername, req.NewUsername) {
			utils.JSONResponse(w, http.StatusOK, map[string]any{
				"success":        false,
				"username_taken": true,
			})
			return
		}
	}

	// Update username
	for _, u := range usersList {
		user, ok := u.(map[string]any)
		if !ok {
			continue
		}

		if int(user["user_id"].(float64)) == userID {
			user["username"] = req.NewUsername
			//usersList[currentUserIndex] = user
			users["users"] = usersList
			break
		}
	}

	// Save users file
	if err := utils.WriteUsers(users); err != nil {
		utils.Logger.Printf("Error saving users after username change: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	utils.Logger.Printf("Username changed for user ID %d to '%s'", userID, req.NewUsername)

	utils.JSONResponse(w, http.StatusOK, map[string]any{
		"success":                true,
		"available_backup_codes": availableBackupCodes,
	})
}

// ValidatePasswordRequest represents the validate password request body
type ValidatePasswordRequest struct {
	Password string `json:"password"`
}

// ValidatePassword validates the user's password for re-authentication
func ValidatePassword(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value(utils.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse the request body
	var req ValidatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate password using the same method as login
	derived_key, available_backup_codes, _ := utils.CheckPasswordForUser(userID, req.Password)

	utils.JSONResponse(w, http.StatusOK, map[string]any{
		"valid":                  derived_key != "",
		"available_backup_codes": available_backup_codes,
	})
}
