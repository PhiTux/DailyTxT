package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
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
		http.Error(w, "User/Password combination not found", http.StatusNotFound)
		return
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
		utils.Logger.Printf("Login failed. User '%s' not found", req.Username)
		http.Error(w, "User/Password combination not found", http.StatusNotFound)
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
	utils.JSONResponse(w, http.StatusOK, map[string]string{
		"username": req.Username,
	})
}

// RegisterRequest represents the register request body
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Register handles user registration
func Register(w http.ResponseWriter, r *http.Request) {
	// Parse the request body
	var req RegisterRequest
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

	// Check if username already exists
	if len(users) > 0 {
		usersList, ok := users["users"].([]any)
		if !ok {
			utils.Logger.Printf("users.json is not in the correct format. Key 'users' is missing or not a list.")
			http.Error(w, "users.json is not in the correct format", http.StatusInternalServerError)
			return
		}

		for _, u := range usersList {
			user, ok := u.(map[string]any)
			if !ok {
				continue
			}

			if username, ok := user["username"].(string); ok && username == req.Username {
				utils.Logger.Printf("Registration failed. Username '%s' already exists", req.Username)
				http.Error(w, "Username already exists", http.StatusBadRequest)
				return
			}
		}
	}

	// Create new user data
	hashedPassword, salt, err := utils.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Create encryption key
	derivedKey, err := utils.DeriveKeyFromPassword(req.Password, salt)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Generate a new random encryption key
	encryptionKey := make([]byte, 32)
	if _, err := utils.RandRead(encryptionKey); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Encrypt the encryption key with the derived key
	aead, err := utils.CreateAEAD(derivedKey)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := utils.RandRead(nonce); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
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
					"username":         req.Username,
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
			"username":         req.Username,
			"password":         hashedPassword,
			"salt":             salt,
			"enc_enc_key":      encEncKey,
		})

		users["users"] = usersList
	}

	// Write users to file
	if err := utils.WriteUsers(users); err != nil {
		http.Error(w, "Internal Server Error when trying to write users.json", http.StatusInternalServerError)
		return
	}

	// Return success
	utils.JSONResponse(w, http.StatusOK, map[string]bool{
		"success": true,
	})
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
