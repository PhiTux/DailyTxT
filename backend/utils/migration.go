package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"crypto/hmac"
	"crypto/subtle"
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
func FernetDecrypt(token string, key []byte) ([]byte, error) {
	// Decode token
	tokenBytes, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token encoding: %v", err)
	}

	// Check token length
	if len(tokenBytes) < 1+8+16+1+32 {
		return nil, fmt.Errorf("token too short")
	}

	// Check version
	if tokenBytes[0] != fernetVersion {
		return nil, fmt.Errorf("invalid token version")
	}

	// Extract parts
	timestamp := tokenBytes[1:9]
	iv := tokenBytes[9:25]
	ciphertext := tokenBytes[25 : len(tokenBytes)-32]
	hmacValue := tokenBytes[len(tokenBytes)-32:]

	// Verify HMAC
	if !verifyFernetHMAC(key, tokenBytes[:len(tokenBytes)-32], hmacValue) {
		return nil, fmt.Errorf("invalid token signature")
	}

	// Verify timestamp
	if !verifyFernetTimestamp(timestamp) {
		return nil, fmt.Errorf("token expired")
	}

	// Create cipher
	block, err := aes.NewCipher(key[16:32])
	if err != nil {
		return nil, fmt.Errorf("error creating cipher: %v", err)
	}

	// Decrypt
	plaintext := make([]byte, len(ciphertext))
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(plaintext, ciphertext)

	return plaintext, nil
}

// verifyFernetHMAC verifies the HMAC signature of a Fernet token
func verifyFernetHMAC(key, data, signature []byte) bool {
	// Create HMAC
	h := hmac.New(sha256.New, key[:16])
	h.Write(data)
	expectedSignature := h.Sum(nil)

	// Compare signatures
	return subtle.ConstantTimeCompare(signature, expectedSignature) == 1
}

// verifyFernetTimestamp verifies that a Fernet token's timestamp is valid
func verifyFernetTimestamp(timestamp []byte) bool {
	// Parse timestamp
	ts := int64(0)
	for _, b := range timestamp {
		ts = (ts << 8) | int64(b)
	}

	// Check if token is expired
	now := time.Now().Unix()
	return now-maxClockSkew <= ts && ts <= now+maxClockSkew
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
func MigrateUserData(username, password string, progressChan chan<- MigrationProgress) error {
	// Prüfen, ob bereits eine Migration für diesen Benutzer läuft
	if IsUserMigrating(username) {
		Logger.Printf("Migration for user %s is already in progress", username)
		return fmt.Errorf("migration already in progress for user %s", username)
	}

	// Benutzer als migrierend markieren
	SetUserMigrating(username, true)
	// Sicherstellen, dass der Benutzer am Ende nicht mehr als migrierend markiert ist
	defer SetUserMigrating(username, false)

	Logger.Printf("Starting migration for user %s", username)

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

	// Set initial progress
	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "initializing",
			CurrentItem:    "Checking user data",
			ProcessedItems: 0,
			TotalItems:     1,
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

	// Update progress
	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "decrypting_keys",
			CurrentItem:    "Decrypting old encryption keys",
			ProcessedItems: 1,
			TotalItems:     5,
		}
	}

	// Derive key from password and salt
	oldDerivedKey := DeriveKeyFromOldPassword(password, oldSalt)
	derKey, err := base64.StdEncoding.DecodeString(base64.URLEncoding.EncodeToString(oldDerivedKey))
	if err != nil {
		return fmt.Errorf("error decoding old derived key: %v", err)
	}
	fmt.Printf("Old derived key: %x\n", derKey)
	fmt.Printf("Old encrypted key: %s\n", oldEncEncKey)

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

	// Update progress
	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "creating_new_user",
			CurrentItem:    "Creating new user",
			ProcessedItems: 1,
			TotalItems:     5,
		}
	}

	// Create new encryption key and user data
	_, newSalt, err := HashPassword(password)
	if err != nil {
		return fmt.Errorf("error hashing password: %v", err)
	}

	newDerivedKey, err := DeriveKeyFromPassword(password, newSalt)
	if err != nil {
		return fmt.Errorf("error deriving key: %v", err)
	}

	// Create a new random encryption key
	newEncKey := make([]byte, 32)
	if _, err := RandRead(newEncKey); err != nil {
		return fmt.Errorf("error generating new encryption key: %v", err)
	}

	// Encrypt the new encryption key
	aead, err := CreateAEAD(newDerivedKey)
	if err != nil {
		return fmt.Errorf("error creating cipher: %v", err)
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := RandRead(nonce); err != nil {
		return fmt.Errorf("error generating nonce: %v", err)
	}

	encryptedNewKey := aead.Seal(nonce, nonce, newEncKey, nil)
	_ = base64.StdEncoding.EncodeToString(encryptedNewKey)

	// Update progress
	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "creating_new_user",
			CurrentItem:    "Adding user to database",
			ProcessedItems: 2,
			TotalItems:     5,
		}
	}

	// Get existing users or create new users object
	newUsers, err := GetUsers()
	if err != nil {
		return fmt.Errorf("error getting users: %v", err)
	}

	// Determine new user ID (must be different from any existing user ID)
	newUserID := oldUserID
	var existingUserIDs = make(map[int]bool)

	if len(newUsers) > 0 {
		// Get existing user IDs
		if usersList, ok := newUsers["users"].([]any); ok {
			for _, u := range usersList {
				user, ok := u.(map[string]any)
				if !ok {
					continue
				}

				if id, ok := user["user_id"].(float64); ok {
					existingUserIDs[int(id)] = true
				}
			}
		}

		// Find a free user ID if the old ID is already taken
		for existingUserIDs[newUserID] {
			newUserID++
		}
	}

	// Write new users
	if err := WriteUsers(newUsers); err != nil {
		return fmt.Errorf("error writing users: %v", err)
	}

	// Update progress
	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "writing_user_data",
			CurrentItem:    "User data saved",
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

	// Update progress
	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "migrating_settings",
			CurrentItem:    "Migrating user settings",
			ProcessedItems: 0,
			TotalItems:     5,
		}
	}

	// Migrate user settings
	if err := migrateUserSettings(oldDataDir, newDataDir, oldEncKey, base64.StdEncoding.EncodeToString(newEncKey), progressChan); err != nil {
		return fmt.Errorf("error migrating settings: %v", err)
	}

	// Migrate templates
	if err := migrateTemplates(oldDataDir, newDataDir, oldEncKey, base64.StdEncoding.EncodeToString(newEncKey), progressChan); err != nil {
		return fmt.Errorf("error migrating templates: %v", err)
	}

	// Migrate tags
	if err := migrateTags(oldDataDir, newDataDir, oldEncKey, base64.StdEncoding.EncodeToString(newEncKey), progressChan); err != nil {
		return fmt.Errorf("error migrating tags: %v", err)
	}

	// Migrate logs (years/months)
	if err := migrateLogs(oldDataDir, newDataDir, oldEncKey, base64.StdEncoding.EncodeToString(newEncKey), progressChan); err != nil {
		return fmt.Errorf("error migrating logs: %v", err)
	}

	// Migrate files
	if err := migrateFiles(oldDataDir, newDataDir, oldEncKey, base64.StdEncoding.EncodeToString(newEncKey), progressChan); err != nil {
		return fmt.Errorf("error migrating files: %v", err)
	}

	// Set final progress
	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "completed",
			CurrentItem:    "Migration completed",
			ProcessedItems: 5,
			TotalItems:     5,
		}
	}

	Logger.Printf("Migration completed for user %s (Old ID: %d, New ID: %d)", username, oldUserID, newUserID)
	return nil
}

/* // DeriveKeyFromOldPassword derives a key from a password using the old method (Python version)
func DeriveKeyFromOldPassword(password, salt string) []byte {
	// In Python version: hash_secret_raw with time_cost=2, memory_cost=2**15, parallelism=1, hash_len=32, type=Type.ID
	// For simplicity, we're using SHA-256 here
	h := sha256.New()
	h.Write([]byte(salt + password))
	derivedKey := h.Sum(nil)
	return derivedKey
} */

// DeriveKeyFromOldPassword derives a key from a password using the old Python method
// with PBKDF2-HMAC-SHA256 with 100,000 iterations and 32 bytes output
func DeriveKeyFromOldPassword(password, salt string) []byte {
	// Use PBKDF2 with HMAC-SHA256, 100,000 iterations, and 32 byte output
	// This matches the Python werkzeug implementation for password hashing
	derivedKey, _ := pbkdf2.Key(sha256.New, password, []byte(salt), 100000, 32)
	return derivedKey
}

// MigrationProgress enthält Informationen zum Fortschritt der Migration
type MigrationProgress struct {
	Phase          string `json:"phase"`           // Aktuelle Migrationsphase
	CurrentItem    string `json:"current_item"`    // Aktuelles Element, das migriert wird
	ProcessedItems int    `json:"processed_items"` // Anzahl der bereits verarbeiteten Elemente
	TotalItems     int    `json:"total_items"`     // Gesamtanzahl der zu migrierenden Elemente
}

// Helper functions for migration

func migrateUserSettings(oldDir, newDir string, oldKey []byte, newKey string, progressChan chan<- MigrationProgress) error {
	// Lock für alte Einstellungen
	userSettingsMutex.RLock()
	// Check if old settings exist
	oldSettingsPath := filepath.Join(oldDir, "settings.encrypted")
	_, err := os.Stat(oldSettingsPath)
	exists := !os.IsNotExist(err)
	userSettingsMutex.RUnlock()

	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error checking if old settings exist: %v", err)
	}

	if !exists {
		return nil // No settings to migrate
	}

	// Update progress
	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "migrating_settings",
			CurrentItem:    "Reading old settings",
			ProcessedItems: 1,
			TotalItems:     4,
		}
	}

	// Read old settings mit Lock
	userSettingsMutex.RLock()
	oldSettingsBytes, err := os.ReadFile(oldSettingsPath)
	userSettingsMutex.RUnlock()

	if err != nil {
		return fmt.Errorf("error reading old settings: %v", err)
	}

	// Update progress
	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "migrating_settings",
			CurrentItem:    "Decrypting old settings",
			ProcessedItems: 2,
			TotalItems:     4,
		}
	}

	// Decrypt old settings
	oldSettingsPlain, err := FernetDecrypt(string(oldSettingsBytes), oldKey)
	if err != nil {
		return fmt.Errorf("error decrypting old settings: %v", err)
	}

	// Update progress
	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "migrating_settings",
			CurrentItem:    "Encrypting settings with new key",
			ProcessedItems: 3,
			TotalItems:     4,
		}
	}

	// Encrypt with new key
	newSettingsEnc, err := EncryptText(string(oldSettingsPlain), newKey)
	if err != nil {
		return fmt.Errorf("error encrypting new settings: %v", err)
	}

	// Update progress
	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "migrating_settings",
			CurrentItem:    "Writing new settings",
			ProcessedItems: 3,
			TotalItems:     4,
		}
	}

	// Write new settings mit Lock
	newSettingsPath := filepath.Join(newDir, "settings.encrypted")
	userSettingsMutex.Lock()
	err = os.WriteFile(newSettingsPath, []byte(newSettingsEnc), 0644)
	userSettingsMutex.Unlock()

	if err != nil {
		return fmt.Errorf("error writing new settings: %v", err)
	}

	// Update progress
	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "migrating_settings",
			CurrentItem:    "Settings migration completed",
			ProcessedItems: 4,
			TotalItems:     4,
		}
	}

	return nil
}

func migrateTemplates(oldDir, newDir string, oldKey []byte, newKey string, progressChan chan<- MigrationProgress) error {
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
			CurrentItem:    "Reading templates",
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

	// Update progress
	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "migrating_templates",
			CurrentItem:    "Writing templates",
			ProcessedItems: 1,
			TotalItems:     2,
		}
	}

	// Templates are not encrypted, just copy
	newTemplatesPath := filepath.Join(newDir, "templates.json")
	templatesMutex.Lock()
	err = os.WriteFile(newTemplatesPath, oldTemplatesBytes, 0644)
	templatesMutex.Unlock()

	if err != nil {
		return fmt.Errorf("error writing new templates: %v", err)
	}

	// Update progress
	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "migrating_templates",
			CurrentItem:    "Templates migration completed",
			ProcessedItems: 2,
			TotalItems:     2,
		}
	}

	return nil
}

func migrateTags(oldDir, newDir string, oldKey []byte, newKey string, progressChan chan<- MigrationProgress) error {
	// Check if old tags exist
	tagsMutex.RLock()
	oldTagsPath := filepath.Join(oldDir, "tags.json")
	_, err := os.Stat(oldTagsPath)
	exists := !os.IsNotExist(err)
	tagsMutex.RUnlock()

	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error checking if old tags exist: %v", err)
	}

	if !exists {
		return nil // No tags to migrate
	}

	// Update progress
	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "migrating_tags",
			CurrentItem:    "Reading tags",
			ProcessedItems: 1,
			TotalItems:     2,
		}
	}

	// Read old tags
	tagsMutex.RLock()
	oldTagsBytes, err := os.ReadFile(oldTagsPath)
	tagsMutex.RUnlock()

	if err != nil {
		return fmt.Errorf("error reading old tags: %v", err)
	}

	// Update progress
	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "migrating_tags",
			CurrentItem:    "Writing tags",
			ProcessedItems: 1,
			TotalItems:     2,
		}
	}

	// Tags are not encrypted, just copy
	newTagsPath := filepath.Join(newDir, "tags.json")
	tagsMutex.Lock()
	err = os.WriteFile(newTagsPath, oldTagsBytes, 0644)
	tagsMutex.Unlock()

	if err != nil {
		return fmt.Errorf("error writing new tags: %v", err)
	}

	// Update progress
	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "migrating_tags",
			CurrentItem:    "Tags migration completed",
			ProcessedItems: 2,
			TotalItems:     2,
		}
	}

	return nil
}

func migrateLogs(oldDir, newDir string, oldKey []byte, newKey string, progressChan chan<- MigrationProgress) error {
	// Get all year directories
	logsMutex.RLock()
	entries, err := os.ReadDir(oldDir)
	logsMutex.RUnlock()

	if err != nil {
		return fmt.Errorf("error reading old directory: %v", err)
	}

	// Count total years and collect year names
	totalYears := 0
	var yearDirs []string
	for _, entry := range entries {
		if entry.IsDir() && isNumeric(entry.Name()) {
			totalYears++
			yearDirs = append(yearDirs, entry.Name())
		}
	}

	// Update progress
	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "migrating_logs",
			CurrentItem:    fmt.Sprintf("Found %d years to migrate", totalYears),
			ProcessedItems: 0,
			TotalItems:     totalYears,
		}
	}

	yearCount := 0
	for _, yearDir := range yearDirs {
		oldYearPath := filepath.Join(oldDir, yearDir)
		newYearPath := filepath.Join(newDir, yearDir)

		// Create new year directory
		logsMutex.Lock()
		if err := os.MkdirAll(newYearPath, 0755); err != nil {
			logsMutex.Unlock()
			return fmt.Errorf("error creating new year directory: %v", err)
		}
		logsMutex.Unlock()

		// Get all month files
		logsMutex.RLock()
		monthEntries, err := os.ReadDir(oldYearPath)
		logsMutex.RUnlock()

		if err != nil {
			return fmt.Errorf("error reading old year directory: %v", err)
		}

		// Count total months
		var monthFiles []string
		for _, monthEntry := range monthEntries {
			if !monthEntry.IsDir() && strings.HasSuffix(monthEntry.Name(), ".json") {
				monthFiles = append(monthFiles, monthEntry.Name())
			}
		}

		// Update progress for this year
		if progressChan != nil {
			progressChan <- MigrationProgress{
				Phase:          "migrating_logs",
				CurrentItem:    fmt.Sprintf("Migrating year %s (%d/%d) - %d months", yearDir, yearCount+1, totalYears, len(monthFiles)),
				ProcessedItems: yearCount,
				TotalItems:     totalYears,
			}
		}

		monthCount := 0
		for _, monthFile := range monthFiles {
			oldMonthPath := filepath.Join(oldYearPath, monthFile)
			newMonthPath := filepath.Join(newYearPath, monthFile)

			// Read old month
			logsMutex.RLock()
			oldMonthBytes, err := os.ReadFile(oldMonthPath)
			logsMutex.RUnlock()

			if err != nil {
				Logger.Printf("Error reading old month %s: %v", oldMonthPath, err)
				continue
			}

			// Parse old month
			var oldMonth map[string]any
			if err := json.Unmarshal(oldMonthBytes, &oldMonth); err != nil {
				Logger.Printf("Error parsing old month %s: %v", oldMonthPath, err)
				continue
			}

			// Decrypt all encrypted fields in logs
			for day, dayData := range oldMonth {
				dayMap, ok := dayData.(map[string]any)
				if !ok {
					continue
				}

				if encrypted, ok := dayMap["content"].(string); ok {
					// Decrypt content
					plaintext, err := FernetDecrypt(encrypted, oldKey)
					if err != nil {
						Logger.Printf("Error decrypting content for day %s: %v", day, err)
						continue
					}

					// Encrypt with new key
					newEncrypted, err := EncryptText(string(plaintext), newKey)
					if err != nil {
						Logger.Printf("Error encrypting content for day %s: %v", day, err)
						continue
					}

					dayMap["content"] = newEncrypted
				}
			}

			// Write new month
			newMonthBytes, err := json.Marshal(oldMonth)
			if err != nil {
				Logger.Printf("Error marshaling new month %s: %v", newMonthPath, err)
				continue
			}

			logsMutex.Lock()
			err = os.WriteFile(newMonthPath, newMonthBytes, 0644)
			logsMutex.Unlock()

			if err != nil {
				Logger.Printf("Error writing new month %s: %v", newMonthPath, err)
				continue
			}

			monthCount++
			if monthCount%5 == 0 && progressChan != nil && len(monthFiles) > 0 {
				// Update progress occasionally
				progressChan <- MigrationProgress{
					Phase:          "migrating_logs",
					CurrentItem:    fmt.Sprintf("Migrating year %s - month %s (%d/%d)", yearDir, monthFile, monthCount, len(monthFiles)),
					ProcessedItems: yearCount*100 + monthCount,
					TotalItems:     totalYears * 100, // Approximation
				}
			}
		}

		yearCount++
		if progressChan != nil {
			progressChan <- MigrationProgress{
				Phase:          "migrating_logs",
				CurrentItem:    fmt.Sprintf("Completed year %s (%d/%d)", yearDir, yearCount, totalYears),
				ProcessedItems: yearCount,
				TotalItems:     totalYears,
			}
		}
	}

	// Update progress
	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "migrating_logs",
			CurrentItem:    "Logs migration completed",
			ProcessedItems: totalYears,
			TotalItems:     totalYears,
		}
	}

	return nil
}

func migrateFiles(oldDir, newDir string, oldKey []byte, newKey string, progressChan chan<- MigrationProgress) error {
	// Check if old files directory exists
	filesMutex.RLock()
	oldFilesDir := filepath.Join(oldDir, "files")
	_, err := os.Stat(oldFilesDir)
	exists := !os.IsNotExist(err)
	filesMutex.RUnlock()

	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error checking if old files directory exists: %v", err)
	}

	if !exists {
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

	// Get all files
	filesMutex.RLock()
	entries, err := os.ReadDir(oldFilesDir)
	filesMutex.RUnlock()

	if err != nil {
		return fmt.Errorf("error reading old files directory: %v", err)
	}

	totalFiles := len(entries)
	fileCount := 0

	// Update progress
	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "migrating_files",
			CurrentItem:    fmt.Sprintf("Found %d files to migrate", totalFiles),
			ProcessedItems: 0,
			TotalItems:     totalFiles,
		}
	}

	for idx, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()
		oldFilePath := filepath.Join(oldFilesDir, fileName)
		newFilePath := filepath.Join(newFilesDir, fileName)

		// Update progress occasionally
		if progressChan != nil && (idx%5 == 0 || idx == 0) {
			progressChan <- MigrationProgress{
				Phase:          "migrating_files",
				CurrentItem:    fmt.Sprintf("Migrating file %s (%d/%d)", fileName, idx+1, totalFiles),
				ProcessedItems: idx,
				TotalItems:     totalFiles,
			}
		}

		// Read old file
		filesMutex.RLock()
		oldFileBytes, err := os.ReadFile(oldFilePath)
		filesMutex.RUnlock()

		if err != nil {
			Logger.Printf("Error reading old file %s: %v", oldFilePath, err)
			continue
		}

		// Decrypt file
		plaintext, err := FernetDecrypt(string(oldFileBytes), oldKey)
		if err != nil {
			Logger.Printf("Error decrypting file %s: %v", fileName, err)
			continue
		}

		// Encrypt with new key
		newEncrypted, err := EncryptFile(plaintext, newKey)
		if err != nil {
			Logger.Printf("Error encrypting file %s: %v", fileName, err)
			continue
		}

		// Write new file
		filesMutex.Lock()
		err = os.WriteFile(newFilePath, newEncrypted, 0644)
		filesMutex.Unlock()

		if err != nil {
			Logger.Printf("Error writing new file %s: %v", newFilePath, err)
			continue
		}

		fileCount++
	}

	// Update final progress
	if progressChan != nil {
		progressChan <- MigrationProgress{
			Phase:          "migrating_files",
			CurrentItem:    fmt.Sprintf("Files migration completed (%d files)", fileCount),
			ProcessedItems: fileCount,
			TotalItems:     totalFiles,
		}
	}

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
