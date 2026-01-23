package handlers

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"

	"github.com/phitux/dailytxt/backend/utils"
)

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
		"firstDayOfWeek":             "monday",
		"showChangelogOnUpdate":      true,
		"writeDateFormat":            "2-digit",
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
