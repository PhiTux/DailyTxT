package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Mutexes für Dateizugriffe
var (
	activeMigrationsMutex sync.RWMutex // Für die Map der aktiven Migrationen
	oldUsersFileMutex     sync.RWMutex // Für old/users.json
	templatesMutex        sync.RWMutex // Für templates.json
	tagsMutex             sync.RWMutex // Für tags.json
	logsMutex             sync.RWMutex // Für Logs
	filesMutex            sync.RWMutex // Für Dateien im files-Verzeichnis
)

// Map zur Verfolgung aktiver Migrationen (username -> bool)
var activeMigrations = make(map[string]bool)

// IsUserMigrating prüft, ob für einen Benutzer bereits eine Migration läuft
func IsUserMigrating(username string) bool {
	activeMigrationsMutex.RLock()
	defer activeMigrationsMutex.RUnlock()
	return activeMigrations[username]
}

// SetUserMigrating markiert einen Benutzer als migrierend oder nicht migrierend
func SetUserMigrating(username string, migrating bool) {
	activeMigrationsMutex.Lock()
	defer activeMigrationsMutex.Unlock()
	if migrating {
		activeMigrations[username] = true
	} else {
		delete(activeMigrations, username)
	}
}

// Ferent implementation based on Python's cryptography.fernet
// Reference: https://github.com/fernet/spec/blob/master/Spec.md

const (
	fernetVersion byte  = 0x80
	maxClockSkew  int64 = 60 // seconds
)

// FernetDecrypt decrypts a Fernet token using the given key
func FernetDecrypt(token string, key []byte) (string, error) {
	// Decode token
	tokenBytes, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return "", fmt.Errorf("invalid token encoding: %v", err)
	}

	// Check token length
	if len(tokenBytes) < 1+8+16+1+32 {
		return "", fmt.Errorf("token too short")
	}

	// Check version
	if tokenBytes[0] != fernetVersion {
		return "", fmt.Errorf("invalid token version")
	}

	// Extract parts
	// timestamp := tokenBytes[1:9]
	iv := tokenBytes[9:25]
	ciphertext := tokenBytes[25 : len(tokenBytes)-32]
	// hmacValue := tokenBytes[len(tokenBytes)-32:]

	// Generate encryption key from the master key
	// signingKey := key[:16] // Unused for now, will be needed if HMAC verification is enabled
	encryptionKey := key[16:32]

	// Verify HMAC (optional for migration, commented out for now)
	// TODO: Uncomment if signature verification is needed
	/*
		h := hmac.New(sha256.New, signingKey)
		h.Write(tokenBytes[:len(tokenBytes)-32])
		calculatedHMAC := h.Sum(nil)
		if subtle.ConstantTimeCompare(calculatedHMAC, hmacValue) != 1 {
			return "", fmt.Errorf("invalid token signature")
		}
	*/

	// Create cipher
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", fmt.Errorf("error creating cipher: %v", err)
	}

	// Decrypt using AES-128-CBC (Fernet uses CBC, not CTR)
	plaintext := make([]byte, len(ciphertext))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(plaintext, ciphertext)

	// Remove PKCS7 padding
	plaintext = pkcs7Unpad(plaintext)

	// Return plaintext as string (it's already a base64-encoded string)
	return string(plaintext), nil
}

// pkcs7Unpad removes PKCS#7 padding
func pkcs7Unpad(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	padding := int(data[len(data)-1])
	if padding > len(data) {
		return data // Invalid padding
	}

	// Check if padding is valid
	for i := len(data) - padding; i < len(data); i++ {
		if int(data[i]) != padding {
			return data // Invalid padding
		}
	}

	return data[:len(data)-padding]
}

// GetOldUsers retrieves the users from the old users.json file
func GetOldUsers() (map[string]any, error) {
	oldUsersFileMutex.RLock()
	defer oldUsersFileMutex.RUnlock()

	// Try to open the old users.json file
	filePath := filepath.Join(Settings.DataPath, "old", "users.json")
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			Logger.Printf("old/users.json - File not found")
			return map[string]any{}, nil
		}
		Logger.Printf("Error opening old/users.json: %v", err)
		return nil, fmt.Errorf("internal server error when trying to open old/users.json")
	}
	defer file.Close()

	// Read the file content
	var content map[string]any
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&content); err != nil {
		Logger.Printf("Error decoding old/users.json: %v", err)
		return nil, fmt.Errorf("internal server error when trying to decode old/users.json")
	}

	return content, nil
}

// VerifyOldPassword verifies if a password matches a hash from the old version
// Uses HMAC-SHA256 for verification
func VerifyOldPassword(password, hash string) bool {
	// Parse the hash format: sha256$salt$hash
	parts := strings.Split(hash, "$")
	if len(parts) != 3 || parts[0] != "sha256" {
		return false
	}

	salt, storedHash := parts[1], parts[2]

	// Create HMAC with SHA256
	h := hmac.New(sha256.New, []byte(salt))
	h.Write([]byte(password))
	calculatedHash := fmt.Sprintf("%x", h.Sum(nil))

	// Compare hashes using constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare([]byte(calculatedHash), []byte(storedHash)) == 1
}

// MigrateUserData migrates a user's data from the old format to the new format
func MigrateUserData(username, password string, registerFunc RegisterUserFunc, progressChan chan<- MigrationProgress) error {
	// Prüfen, ob bereits eine Migration für diesen Benutzer läuft
	if IsUserMigrating(username) {
		Logger.Printf("Migration for user %s is already in progress", username)
		return fmt.Errorf("migration already in progress for user %s", username)
	}

	// Benutzer als migrierend markieren
	SetUserMigrating(username, true)
	// Sicherstellen, dass der Benutzer am Ende nicht mehr als migrierend markiert ist
	defer SetUserMigrating(username, false)

	start := time.Now()
	Logger.Printf("Starting migration for user %s with password %s", username, password)

	// Get old users
	oldUsersFileMutex.RLock()
	oldUsersPath := filepath.Join(Settings.DataPath, "old", "users.json")
	oldUsersBytes, err := os.ReadFile(oldUsersPath)
	oldUsersFileMutex.RUnlock()

	if err != nil {
		return fmt.Errorf("error reading old users: %v", err)
	}

	// Parse old users
	var oldUsers map[string]any
	if err := json.Unmarshal(oldUsersBytes, &oldUsers); err != nil {
		return fmt.Errorf("error parsing old users: %v", err)
	}

	// Find the old user by username
	oldUserID := 0
	var oldUser map[string]any
	for _, user := range oldUsers["users"].([]any) {
		u := user.(map[string]any)
		if u["username"] == username {
			oldUser = u
			break
		}
	}

	if oldUser == nil {
		return fmt.Errorf("user %s not found in old data", username)
	}

	oldUserID = int(oldUser["user_id"].(float64))

	Logger.Printf("Found old user ID: %d", oldUserID)

	// Update progress
	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "creating_new_user",
			CurrentItem:    "",
			ProcessedItems: 1,
			TotalItems:     5,
		}
	}

	// Verify username matches
	oldUsername, ok := oldUser["username"].(string)
	if !ok || oldUsername != username {
		return fmt.Errorf("username mismatch: expected %s, got %s", username, oldUsername)
	}

	// Get encryption related data from old user
	oldSalt, ok := oldUser["salt"].(string)
	if !ok {
		return fmt.Errorf("old user data is missing salt")
	}

	oldEncEncKey, ok := oldUser["enc_enc_key"].(string)
	if !ok {
		return fmt.Errorf("old user data is missing encrypted key")
	}

	// Derive key from password and salt
	oldDerivedKey := DeriveKeyFromOldPassword(password, oldSalt)
	_, err = base64.StdEncoding.DecodeString(base64.URLEncoding.EncodeToString(oldDerivedKey))
	if err != nil {
		return fmt.Errorf("error decoding old derived key: %v", err)
	}

	// Decode the old encrypted key (just for validation)
	_, err = base64.URLEncoding.DecodeString(oldEncEncKey)
	if err != nil {
		return fmt.Errorf("error decoding old encrypted key: %v", err)
	}

	// Decrypt the old encryption key
	oldEncKey, err := FernetDecrypt(oldEncEncKey, oldDerivedKey)
	if err != nil {
		return fmt.Errorf("error decrypting old encryption key: %v", err)
	}

	// Debug: Zeige den Schlüssel
	fmt.Printf("Old encryption key: %s\n", oldEncKey)

	// Registriere den Benutzer mit der übergebenen Funktion
	success, err := registerFunc(username, password)
	if err != nil {
		return fmt.Errorf("error registering new user: %v", err)
	}
	if !success {
		return fmt.Errorf("failed to register new user")
	}

	users, err := GetUsers()
	if err != nil {
		return fmt.Errorf("error getting users: %v", err)
	}
	// Find the new user ID
	newUserID := 0
	newDerivedKey := ""
	for _, user := range users["users"].([]any) {
		u := user.(map[string]any)
		if u["username"] == username {
			if id, ok := u["user_id"].(float64); ok {
				newUserID = int(id)

				// Verify password
				if !VerifyPassword(password, u["password"].(string), u["salt"].(string)) {
					Logger.Printf("Login failed. Password for user '%s' is incorrect", username)
					return fmt.Errorf("user/Password combination not found: %d", http.StatusNotFound)
				}

				// Get intermediate key
				derivedKey, err := DeriveKeyFromPassword(password, u["salt"].(string))
				if err != nil {
					return fmt.Errorf("internal Server Error: %d", http.StatusInternalServerError)
				}
				newDerivedKey = base64.StdEncoding.EncodeToString(derivedKey)

				break
			}
		}
	}
	if newUserID <= 0 {
		return fmt.Errorf("new user ID not found for username: %s", username)
	}

	fmt.Printf("New derived key: %s\n", newDerivedKey)

	// Update progress
	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "writing_user_data",
			ProcessedItems: 3,
			TotalItems:     5,
		}
	}

	// Now migrate all the data
	oldDataDir := filepath.Join(Settings.DataPath, "old", strconv.Itoa(oldUserID))
	newDataDir := filepath.Join(Settings.DataPath, strconv.Itoa(newUserID))

	// Create new data directory
	if err := os.MkdirAll(newDataDir, 0755); err != nil {
		return fmt.Errorf("error creating new data directory: %v", err)
	}

	encKey, err := GetEncryptionKey(newUserID, string(newDerivedKey))
	if err != nil {
		return fmt.Errorf("error getting encryption key: %v", err)
	}

	fmt.Printf("New encryption key: %s\n", encKey)

	// Migrate templates
	if err := migrateTemplates(oldDataDir, newDataDir, oldEncKey, encKey, progressChan); err != nil {
		return fmt.Errorf("error migrating templates: %v", err)
	}

	// Migrate logs (years/months)
	if err := migrateLogs(oldDataDir, newDataDir, oldEncKey, encKey, progressChan); err != nil {
		return fmt.Errorf("error migrating logs: %v", err)
	}

	// Migrate files
	if err := migrateFiles(filepath.Join(Settings.DataPath, "old", "files"), newDataDir, oldEncKey, encKey, progressChan); err != nil {
		return fmt.Errorf("error migrating files: %v", err)
	}

	// Set final progress
	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "completed",
			ProcessedItems: 5,
			TotalItems:     5,
		}
	}

	Logger.Printf("Migration completed for user %s (Old ID: %d, New ID: %d) after %v", username, oldUserID, newUserID, time.Since(start))
	return nil
}

// DeriveKeyFromOldPassword derives a key from a password using the old Python method
// with PBKDF2-HMAC-SHA256 with 100,000 iterations and 32 bytes output
func DeriveKeyFromOldPassword(password, salt string) []byte {
	salt_decoded, err := base64.URLEncoding.DecodeString(salt)
	if err != nil {
		fmt.Printf("Error decoding salt: %v\n", err)
		return nil
	}
	derivedKey, _ := pbkdf2.Key(sha256.New, password, salt_decoded, 100000, 32)
	return derivedKey
}

// MigrationProgress contains information about the migration progress
type MigrationProgress struct {
	Phase          string `json:"phase"`           // Current migration phase
	CurrentItem    string `json:"current_item"`    // Current item being migrated
	ProcessedItems int    `json:"processed_items"` // Number of already processed items
	TotalItems     int    `json:"total_items"`     // Total number of items to migrate
}

// RegisterUserFunc is a function type for user registration
type RegisterUserFunc func(username, password string) (bool, error)

// Helper functions for migration

func migrateTemplates(oldDir, newDir string, oldKey string, newKey string, progressChan chan<- MigrationProgress) error {
	// Check if old templates exist
	templatesMutex.RLock()
	oldTemplatesPath := filepath.Join(oldDir, "templates.json")
	_, err := os.Stat(oldTemplatesPath)
	exists := !os.IsNotExist(err)
	templatesMutex.RUnlock()

	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error checking if old templates exist: %v", err)
	}

	if !exists {
		return nil // No templates to migrate
	}

	// Update progress
	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "migrating_templates",
			ProcessedItems: 1,
			TotalItems:     2,
		}
	}

	// Read old templates
	templatesMutex.RLock()
	oldTemplatesBytes, err := os.ReadFile(oldTemplatesPath)
	templatesMutex.RUnlock()

	if err != nil {
		return fmt.Errorf("error reading old templates: %v", err)
	}

	// Templates need to be parsed and re-written with proper indentation
	var templatesData map[string]any
	if err := json.Unmarshal(oldTemplatesBytes, &templatesData); err != nil {
		return fmt.Errorf("error parsing old templates: %v", err)
	}

	// Create templates.json file with proper indentation
	newTemplatesPath := filepath.Join(newDir, "templates.json")
	templatesMutex.Lock()

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(newDir, 0755); err != nil {
		templatesMutex.Unlock()
		return fmt.Errorf("error creating templates directory %s: %v", newDir, err)
	}

	// Create the file
	file, err := os.Create(newTemplatesPath)
	if err != nil {
		templatesMutex.Unlock()
		return fmt.Errorf("error creating templates file: %v", err)
	}

	// Write the content to the file with proper indentation
	var encoder *json.Encoder
	if Settings.Indent > 0 {
		encoder = json.NewEncoder(file)
		encoder.SetIndent("", fmt.Sprintf("%*s", Settings.Indent, ""))
	} else {
		encoder = json.NewEncoder(file)
	}

	if err := encoder.Encode(templatesData); err != nil {
		file.Close()
		templatesMutex.Unlock()
		return fmt.Errorf("error encoding templates data: %v", err)
	}

	file.Close()
	templatesMutex.Unlock()

	return nil
}

func migrateLogs(oldDir, newDir string, oldKey string, newKey string, progressChan chan<- MigrationProgress) error {
	// Count all month files in all year directories
	var allMonthFiles []struct {
		yearDir   string
		monthFile string
	}
	totalMonths := 0

	logsMutex.RLock()
	entries, err := os.ReadDir(oldDir)
	logsMutex.RUnlock()

	if err != nil {
		return fmt.Errorf("error reading old directory: %v", err)
	}

	// Filter for year directories (numeric names)
	var yearDirs []string
	for _, entry := range entries {
		if entry.IsDir() && isNumeric(entry.Name()) {
			yearDirs = append(yearDirs, entry.Name())
		}
	}

	// Count month files in each year directory
	for _, yearDir := range yearDirs {
		oldYearPath := filepath.Join(oldDir, yearDir)

		logsMutex.RLock()
		monthEntries, err := os.ReadDir(oldYearPath)
		logsMutex.RUnlock()

		if err != nil {
			Logger.Printf("Error reading year directory %s: %v", oldYearPath, err)
			continue
		}

		for _, monthEntry := range monthEntries {
			if !monthEntry.IsDir() && strings.HasSuffix(monthEntry.Name(), ".json") {
				totalMonths++
				allMonthFiles = append(allMonthFiles, struct {
					yearDir   string
					monthFile string
				}{
					yearDir:   yearDir,
					monthFile: monthEntry.Name(),
				})
			}
		}
	}

	// Update progress with total number of months
	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "migrating_logs",
			ProcessedItems: 0,
			TotalItems:     totalMonths,
		}
	}

	processedMonths := 0

	// Process all months
	for _, monthInfo := range allMonthFiles {

		// Update progress with total number of months
		if progressChan != nil && processedMonths%5 == 0 {
			progressChan <- MigrationProgress{
				Phase:          "migrating_logs",
				ProcessedItems: processedMonths,
				TotalItems:     totalMonths,
			}
		}

		oldYearPath := filepath.Join(oldDir, monthInfo.yearDir)
		newYearPath := filepath.Join(newDir, monthInfo.yearDir)

		oldMonthPath := filepath.Join(oldYearPath, monthInfo.monthFile)
		newMonthPath := filepath.Join(newYearPath, monthInfo.monthFile)

		// Create year directory if needed
		logsMutex.Lock()
		if err := os.MkdirAll(newYearPath, 0755); err != nil {
			logsMutex.Unlock()
			return fmt.Errorf("error creating new year directory: %v", err)
		}
		logsMutex.Unlock()

		// Read old month file
		logsMutex.RLock()
		oldMonthBytes, err := os.ReadFile(oldMonthPath)
		logsMutex.RUnlock()

		if err != nil {
			Logger.Printf("Error reading old month %s: %v", oldMonthPath, err)
			continue
		}

		// Parse old month file
		var monthData map[string]any
		if err := json.Unmarshal(oldMonthBytes, &monthData); err != nil {
			Logger.Printf("Error parsing old month %s: %v", oldMonthPath, err)
			continue
		}

		// Process days in month
		days, ok := monthData["days"].([]any)
		if !ok {
			Logger.Printf("Month %s has unexpected format - missing 'days' array", oldMonthPath)
			continue
		}

		oldKeyBytes, err := base64.URLEncoding.DecodeString(oldKey)
		if err != nil {
			Logger.Printf("Error decoding oldKey %v", err)
			continue
		}

		// Loop through all days and decrypt/encrypt the texts
		for i, dayInterface := range days {
			day, ok := dayInterface.(map[string]any)
			if !ok {
				continue
			} // Re-encrypt main text
			if encryptedText, ok := day["text"].(string); ok {
				plaintext := ""

				if encryptedText != "" {
					// encode oldKey from base64 to []byte
					plaintext, err = FernetDecrypt(encryptedText, oldKeyBytes)
					if err != nil {
						Logger.Printf("Error decrypting content for day %f in %s: %v", day["day"].(float64), oldMonthPath, err)
						continue
					}
				}

				newEncrypted, err := EncryptText(plaintext, newKey)
				if err != nil {
					Logger.Printf("Error encrypting content for day %d in %s: %v", i, oldMonthPath, err)
					continue
				}

				day["text"] = newEncrypted
			}

			// Also encrypt the (old plaintext) date_written
			if dateWritten, ok := day["date_written"].(string); ok {
				newEncrypted, err := EncryptText(dateWritten, newKey)
				if err != nil {
					Logger.Printf("Error encrypting date_written for day %d in %s: %v", i, oldMonthPath, err)
					continue
				}
				day["date_written"] = newEncrypted
			}

			// Process history if available
			if historyInterface, ok := day["history"].([]any); ok {
				for j, historyItemInterface := range historyInterface {
					historyItem, ok := historyItemInterface.(map[string]any)
					if !ok {
						continue
					}

					// Decrypt history text with old key
					if encryptedText, ok := historyItem["text"].(string); ok {
						plaintext := ""

						if encryptedText != "" {
							plaintext, err = FernetDecrypt(encryptedText, oldKeyBytes)
							if err != nil {
								Logger.Printf("Error decrypting history item %f for day %d in %s: %v", historyItem["version"].(float64), day["day"].(int), oldMonthPath, err)
								continue
							}
						}

						// Encrypt the text with the new key
						newEncrypted, err := EncryptText(plaintext, newKey)
						if err != nil {
							Logger.Printf("Error encrypting history item %d for day %d in %s: %v", j, i, oldMonthPath, err)
							continue
						}

						historyItem["text"] = newEncrypted
					}

					// Encrypt the date_written in history
					if dateWritten, ok := historyItem["date_written"].(string); ok {
						newEncrypted, err := EncryptText(dateWritten, newKey)
						if err != nil {
							Logger.Printf("Error encrypting date_written for history item %d in day %d of %s: %v", j, i, oldMonthPath, err)
							continue
						}
						historyItem["date_written"] = newEncrypted
					}
				}
			}
		}

		// Write new month file with proper indentation
		logsMutex.Lock()

		// Create the directory if it doesn't exist (should already be done, but just to be safe)
		err = os.MkdirAll(filepath.Dir(newMonthPath), 0755)
		if err != nil {
			logsMutex.Unlock()
			Logger.Printf("Error creating directory for %s: %v", newMonthPath, err)
			continue
		}

		// Create the file
		file, err := os.Create(newMonthPath)
		if err != nil {
			logsMutex.Unlock()
			Logger.Printf("Error creating file %s: %v", newMonthPath, err)
			continue
		}

		// Write the content to the file with proper indentation
		var encoder *json.Encoder
		if Settings.Indent > 0 {
			encoder = json.NewEncoder(file)
			encoder.SetIndent("", fmt.Sprintf("%*s", Settings.Indent, ""))
		} else {
			encoder = json.NewEncoder(file)
		}

		if err := encoder.Encode(monthData); err != nil {
			file.Close()
			logsMutex.Unlock()
			Logger.Printf("Error encoding month data for %s: %v", newMonthPath, err)
			continue
		}

		file.Close()
		logsMutex.Unlock()

		processedMonths++
	}

	return nil
}

func migrateFiles(oldFilesDir, newDir string, oldKey string, newKey string, progressChan chan<- MigrationProgress) error {
	// Check if old files directory exists
	filesMutex.RLock()
	_, err := os.Stat(oldFilesDir)
	exists := !os.IsNotExist(err)
	filesMutex.RUnlock()

	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error checking if old files directory exists: %v", err)
	}

	if !exists {
		Logger.Println("No old files directory found, skipping file migration")
		return nil // No files to migrate
	}

	// Create new files directory
	newFilesDir := filepath.Join(newDir, "files")
	filesMutex.Lock()
	if err := os.MkdirAll(newFilesDir, 0755); err != nil {
		filesMutex.Unlock()
		return fmt.Errorf("error creating new files directory: %v", err)
	}
	filesMutex.Unlock()

	// Convert oldKey from base64 to []byte for decryption
	oldKeyBytes, err := base64.URLEncoding.DecodeString(oldKey)
	if err != nil {
		return fmt.Errorf("error decoding oldKey: %v", err)
	}

	// First, find all year directories in the new user directory
	logsMutex.RLock()
	yearEntries, err := os.ReadDir(newDir)
	logsMutex.RUnlock()
	if err != nil {
		return fmt.Errorf("error reading new user directory: %v", err)
	}

	// Track all file references
	type FileRef struct {
		Year     string
		Month    string
		Day      int
		OrigUUID string
		NewUUID  string // Will be generated later
		Size     uint64 // Size of the original file (will be determined later)
	}
	var fileRefs []FileRef

	// Find all files referenced in logs
	Logger.Println("Scanning logs for file references...")

	// First pass: collect all file references from all logs
	for _, yearEntry := range yearEntries {
		// Skip non-directory entries and non-numeric directories
		if !yearEntry.IsDir() || !isNumeric(yearEntry.Name()) {
			continue
		}

		yearDir := yearEntry.Name()
		yearPath := filepath.Join(newDir, yearDir)

		// Read all month files in this year
		logsMutex.RLock()
		monthEntries, err := os.ReadDir(yearPath)
		logsMutex.RUnlock()
		if err != nil {
			Logger.Printf("Error reading year directory %s: %v", yearPath, err)
			continue
		}

		// Scan each month file for file references
		for _, monthEntry := range monthEntries {
			if monthEntry.IsDir() || !strings.HasSuffix(monthEntry.Name(), ".json") {
				continue
			}

			monthFile := monthEntry.Name()
			monthPath := filepath.Join(yearPath, monthFile)

			// Read month file
			logsMutex.RLock()
			monthBytes, err := os.ReadFile(monthPath)
			logsMutex.RUnlock()
			if err != nil {
				Logger.Printf("Error reading month file %s: %v", monthPath, err)
				continue
			}

			// Parse month data
			var monthData map[string]any
			if err := json.Unmarshal(monthBytes, &monthData); err != nil {
				Logger.Printf("Error parsing month data %s: %v", monthPath, err)
				continue
			}

			// Check for days with files
			days, ok := monthData["days"].([]any)
			if !ok {
				continue
			}

			// Process each day
			for _, dayInterface := range days {
				day, ok := dayInterface.(map[string]any)
				if !ok {
					continue
				}

				// Get day number
				dayNum, ok := day["day"].(float64)
				if !ok {
					continue
				}

				// Check for files array
				files, ok := day["files"].([]any)
				if !ok || len(files) == 0 {
					continue
				}

				// Process each file reference
				for _, fileInterface := range files {
					file, ok := fileInterface.(map[string]any)
					if !ok {
						continue
					}

					// Get file ID
					var uuid string
					// Check for both old format (id) and new format (uuid_filename)
					if uuid, ok = file["uuid_filename"].(string); !ok || uuid == "" {
						continue
					}

					// Add to list of files to migrate
					fileRefs = append(fileRefs, FileRef{
						Year:     yearDir,
						Month:    monthFile,
						Day:      int(dayNum),
						OrigUUID: uuid,
						NewUUID:  "", // Will be generated during migration
						Size:     0,  // Size will be determined later
					})
				}
			}
		}
	}

	// Update progress with total number of files
	totalFiles := len(fileRefs)
	Logger.Printf("Found %d files to migrate", totalFiles)

	if totalFiles == 0 {
		if progressChan != nil {
			progressChan <- MigrationProgress{
				Phase:          "migrating_files",
				ProcessedItems: 0,
				TotalItems:     0,
			}
		}
		return nil // No files to migrate
	}

	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "migrating_files",
			ProcessedItems: 0,
			TotalItems:     totalFiles,
		}
	}

	// Second pass: migrate each file
	processedFiles := 0
	fileIDMap := make(map[string]string) // Map original file IDs to new file IDs

	for i, fileRef := range fileRefs {
		// Update progress occasionally
		if progressChan != nil && (i%5 == 0 || i == 0) {
			progressChan <- MigrationProgress{
				Phase:          "migrating_files",
				ProcessedItems: processedFiles,
				TotalItems:     totalFiles,
			}
		}

		// Check if we already have a mapping for this file ID
		if newID, exists := fileIDMap[fileRef.OrigUUID]; exists {
			fileRefs[i].NewUUID = newID
			continue
		}

		// Generate a new UUID for the file
		NewUUID, err := GenerateUUID()
		if err != nil {
			Logger.Printf("Error generating UUID for file %s: %v", fileRef.OrigUUID, err)
			continue
		}

		// Store the mapping
		fileIDMap[fileRef.OrigUUID] = NewUUID
		fileRefs[i].NewUUID = NewUUID

		// Read the old file
		oldFilePath := filepath.Join(oldFilesDir, fileRef.OrigUUID)
		filesMutex.RLock()
		oldFileBytes, err := os.ReadFile(oldFilePath)
		filesMutex.RUnlock()
		if err != nil {
			Logger.Printf("Error reading old file %s: %v", oldFilePath, err)
			continue
		}

		// Decrypt file with old key - der Dateiinhalt ist bereits ein Fernet-Token
		plaintext, err := FernetDecrypt(string(oldFileBytes), oldKeyBytes)
		if err != nil {
			Logger.Printf("Error decrypting file %s: %v", fileRef.OrigUUID, err)
			continue
		}

		plaintextBytes := []byte(plaintext)

		// Store the size of the original file
		fileRefs[i].Size = uint64(len(plaintextBytes)) // Store the size of the original file

		// Encrypt with new key
		newEncrypted, err := EncryptFile(plaintextBytes, newKey)
		if err != nil {
			Logger.Printf("Error encrypting file %s: %v", fileRef.OrigUUID, err)
			continue
		}

		// Write new file
		newFilePath := filepath.Join(newFilesDir, NewUUID)
		filesMutex.Lock()
		err = os.WriteFile(newFilePath, newEncrypted, 0644)
		filesMutex.Unlock()
		if err != nil {
			Logger.Printf("Error writing new file %s: %v", newFilePath, err)
			continue
		}

		processedFiles++
	}

	// Third pass: update all month files with new file IDs
	updatedMonths := make(map[string]bool) // Track which month files we've already updated

	for _, fileRef := range fileRefs {
		monthPath := filepath.Join(newDir, fileRef.Year, fileRef.Month)

		// Skip if we've already updated this month
		if updatedMonths[monthPath] {
			continue
		}

		// Read month file
		logsMutex.RLock()
		monthBytes, err := os.ReadFile(monthPath)
		logsMutex.RUnlock()
		if err != nil {
			Logger.Printf("Error reading month file %s: %v", monthPath, err)
			continue
		}

		// Parse month data
		var monthData map[string]any
		if err := json.Unmarshal(monthBytes, &monthData); err != nil {
			Logger.Printf("Error parsing month data %s: %v", monthPath, err)
			continue
		}

		// Flag to track if we modified the month data
		monthModified := false

		// Update file references in days
		days, ok := monthData["days"].([]any)
		if !ok {
			continue
		}

		for i, dayInterface := range days {
			day, ok := dayInterface.(map[string]any)
			if !ok {
				continue
			}

			files, ok := day["files"].([]any)
			if !ok || len(files) == 0 {
				continue
			}

			// Check each file in this day
			for j, fileInterface := range files {
				file, ok := fileInterface.(map[string]any)
				if !ok {
					continue
				}

				var fileUUID string
				if fileUUID, ok = file["uuid_filename"].(string); !ok || fileUUID == "" {
					continue
				}

				// If we have a mapping for this file UUID, update it
				if newID, exists := fileIDMap[fileUUID]; exists {
					// Entferne das alte Format und ersetze es durch das neue Format
					delete(file, "id")
					file["uuid_filename"] = newID

					// Finde die korrekte Größe für diese Datei
					var fileSize uint64
					for _, ref := range fileRefs {
						if ref.OrigUUID == fileUUID {
							fileSize = ref.Size
							break
						}
					}
					file["size"] = fileSize // Set the correct size of the file
					monthModified = true

					// Update encrypted filename if it exists
					if encName, ok := file["enc_filename"].(string); ok && encName != "" {
						// Decrypt name with old key
						var plainName string
						plainName, err = FernetDecrypt(encName, oldKeyBytes)
						if err != nil {
							Logger.Printf("Error decrypting filename for %s: %v", fileUUID, err)
							continue
						}

						// Encrypt name with new key
						var newEncName string
						newEncName, err = EncryptText(plainName, newKey)
						if err != nil {
							Logger.Printf("Error encrypting filename for %s: %v", fileUUID, err)
							continue
						}

						delete(file, "name")
						file["enc_filename"] = newEncName
					}

					// Update the files array
					files[j] = file
				}
			}

			// Update the day's files array
			if monthModified {
				day["files"] = files
				days[i] = day
			}
		}

		// Update the month data if it was modified
		if monthModified {
			monthData["days"] = days

			// Write back the updated month file
			logsMutex.Lock()

			// Create the file
			file, err := os.Create(monthPath)
			if err != nil {
				logsMutex.Unlock()
				Logger.Printf("Error creating file %s: %v", monthPath, err)
				continue
			}

			// Write with proper indentation
			var encoder *json.Encoder
			if Settings.Indent > 0 {
				encoder = json.NewEncoder(file)
				encoder.SetIndent("", fmt.Sprintf("%*s", Settings.Indent, ""))
			} else {
				encoder = json.NewEncoder(file)
			}

			if err := encoder.Encode(monthData); err != nil {
				file.Close()
				logsMutex.Unlock()
				Logger.Printf("Error encoding month data for %s: %v", monthPath, err)
				continue
			}

			file.Close()
			logsMutex.Unlock()
		}

		// Mark this month as updated
		updatedMonths[monthPath] = true
	}

	// Final progress update
	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "migrating_files",
			ProcessedItems: processedFiles,
			TotalItems:     totalFiles,
		}
	}

	Logger.Printf("Completed migrating %d/%d files", processedFiles, totalFiles)
	return nil
}

// isNumeric checks if a string contains only numeric characters
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}
