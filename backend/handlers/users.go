package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
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
		/* http.Error(w, "User/Password combination not found", http.StatusNotFound)
		return */
	}

	// Find user
	var userID int
	var hashedPassword string
	var salt string
	found := false

	for _, u := range usersList {
		user, ok := u.(map[string]any)
		if !ok {
			continue
		}

		if username, ok := user["username"].(string); ok && username == req.Username {
			found = true
			if id, ok := user["user_id"].(float64); ok {
				userID = int(id)
			}
			if pwd, ok := user["password"].(string); ok {
				hashedPassword = pwd
			}
			if s, ok := user["salt"].(string); ok {
				salt = s
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

		oldUsersList, ok := oldUsers["users"].([]interface{})
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

			if username, ok := user["username"].(string); ok && username == req.Username {
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
		isActive := activeMigrations[req.Username]
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
		activeMigrations[req.Username] = true
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
					migrationProgress[req.Username] = MigrationProgress{
						Phase:          progress.Phase,
						CurrentItem:    progress.CurrentItem,
						ProcessedItems: progress.ProcessedItems,
						TotalItems:     progress.TotalItems,
					}
					migrationProgressMutex.Unlock()
				}
			}()

			err := utils.MigrateUserData(req.Username, req.Password, Register, progressChan)
			if err != nil {
				utils.Logger.Printf("Migration failed for user '%s': %v", req.Username, err)
				// Mark migration as completed even on error
				activeMigrationsMutex.Lock()
				activeMigrations[req.Username] = false
				activeMigrationsMutex.Unlock()
				return
			}

			utils.Logger.Printf("Migration completed for user '%s'", req.Username)

			// Mark migration as completed
			activeMigrationsMutex.Lock()
			activeMigrations[req.Username] = false
			activeMigrationsMutex.Unlock()
		}()

		// Return migration status to client
		utils.JSONResponse(w, http.StatusAccepted, map[string]any{
			"migration_started": true,
			"username":          req.Username,
		})
		return
	}

	// Verify password
	if !utils.VerifyPassword(req.Password, hashedPassword, salt) {
		utils.Logger.Printf("Login failed. Password for user '%s' is incorrect", req.Username)
		http.Error(w, "User/Password combination not found", http.StatusNotFound)
		return
	}

	// Get intermediate key
	derivedKey, err := utils.DeriveKeyFromPassword(req.Password, salt)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	derivedKeyBase64 := base64.StdEncoding.EncodeToString(derivedKey)

	// Create JWT token
	token, err := utils.GenerateToken(userID, req.Username, derivedKeyBase64)
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
		"migration_started": false,
		"username":          req.Username,
	})
}

func IsRegistrationAllowed(w http.ResponseWriter, r *http.Request) {
	// Check if registration is allowed
	utils.JSONResponse(w, http.StatusOK, map[string]bool{
		"registration_allowed": utils.Settings.AllowRegistration,
	})
}

// RegisterRequest represents the register request body
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Register handles user registration
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
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

func Register(username string, password string) (bool, error) {
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

			if username_from_file, ok := user["username"].(string); ok && username_from_file == username {
				utils.Logger.Printf("Registration failed. Username '%s' already exists", username)
				return false, fmt.Errorf("username already exists: %d", http.StatusBadRequest)
			}
		}
	}

	// Create new user data
	hashedPassword, salt, err := utils.HashPassword(password)
	if err != nil {
		return false, fmt.Errorf("internal Server Error: %d", http.StatusInternalServerError)
	}

	// Create encryption key
	derivedKey, err := utils.DeriveKeyFromPassword(password, salt)
	if err != nil {
		return false, fmt.Errorf("internal Server Error: %d", http.StatusInternalServerError)
	}

	// Generate a new random encryption key
	encryptionKey := make([]byte, 32)
	if _, err := utils.RandRead(encryptionKey); err != nil {
		return false, fmt.Errorf("internal Server Error: %d", http.StatusInternalServerError)
	}

	// Encrypt the encryption key with the derived key
	aead, err := utils.CreateAEAD(derivedKey)
	if err != nil {
		return false, fmt.Errorf("internal Server Error: %d", http.StatusInternalServerError)
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := utils.RandRead(nonce); err != nil {
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
	defaultSettings := map[string]any{
		"autoloadImagesByDefault":    false,
		"setAutoloadImagesPerDevice": true,
		"useOnThisDay":               true,
		"onThisDayYears":             []int{1, 5, 10},
		"useBrowserTimezone":         true,
		"timezone":                   "UTC",
	}

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
		currentSettings = map[string]any{
			"autoloadImagesByDefault":    false,
			"setAutoloadImagesPerDevice": true,
			"useOnThisDay":               true,
			"onThisDayYears":             []int{1, 5, 10},
			"useBrowserTimezone":         true,
			"timezone":                   "UTC",
		}
	}

	// Update settings
	for key, value := range newSettings {
		currentSettings[key] = value
	}

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
	CurrentItem    string `json:"current_item"`    // Current item being migrated
	ProcessedItems int    `json:"processed_items"` // Number of items processed
	TotalItems     int    `json:"total_items"`     // Total number of items to process
}

// migrationProgress keeps track of migration progress for all users
var migrationProgress = make(map[string]MigrationProgress)
var migrationProgressMutex sync.Mutex
var activeMigrations = make(map[string]bool)
var activeMigrationsMutex sync.RWMutex

// CheckMigrationProgress checks the progress of a user migration
func CheckMigrationProgress(w http.ResponseWriter, r *http.Request) {
	// Get username from query parameters
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	// Get progress
	migrationProgressMutex.Lock()
	progress, exists := migrationProgress[username]
	migrationProgressMutex.Unlock()

	if !exists {
		utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
			"progress": 0,
			"status":   "not_started",
		})
		return
	}

	// Return progress
	status := "in_progress"
	if progress.TotalItems > 0 && progress.ProcessedItems >= progress.TotalItems {
		status = "completed"
	}

	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"progress": progress,
		"status":   status,
	})
}

// GetMigrationProgress returns the migration progress for a user
func GetMigrationProgress(w http.ResponseWriter, r *http.Request) {
	// Parse the request body
	var req struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get migration progress
	migrationProgressMutex.Lock()
	progress, exists := migrationProgress[req.Username]
	migrationProgressMutex.Unlock()

	// Check if migration is actually active
	activeMigrationsMutex.RLock()
	isActive := activeMigrations[req.Username]
	activeMigrationsMutex.RUnlock()

	if !exists {
		utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
			"migration_in_progress": false,
			"status":                "not_started",
		})
		return
	}

	// Check if migration is completed
	migrationCompleted := progress.Phase == "completed" || (progress.ProcessedItems >= progress.TotalItems && progress.TotalItems > 0)

	// Return progress
	status := "in_progress"
	if migrationCompleted {
		status = "completed"
	} else if !isActive {
		// If migration is not active but not completed, it might have failed
		status = "failed"
	}

	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"migration_in_progress": isActive && !migrationCompleted,
		"progress":              progress,
		"status":                status,
	})
}
