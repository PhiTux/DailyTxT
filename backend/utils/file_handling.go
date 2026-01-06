package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Mutexes for file operations
var (
	UsersFileMutex    sync.RWMutex // For users.json
	userSettingsMutex sync.RWMutex // FFor user settings
)

// GetUsers retrieves the users from the users.json file
func GetUsers() (map[string]any, error) {
	// Try to open the users.json file
	filePath := filepath.Join(Settings.DataPath, "users.json")
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			Logger.Printf("users.json - File not found")
			return map[string]any{}, nil
		}
		Logger.Printf("Error opening users.json: %v", err)
		return nil, fmt.Errorf("internal server error when trying to open users.json")
	}
	defer file.Close()

	// Read the file content
	var content map[string]any
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&content); err != nil {
		if err == io.EOF {
			return map[string]any{}, nil
		}
		Logger.Printf("Error decoding users.json: %v", err)
		return nil, fmt.Errorf("internal server error when trying to decode users.json")
	}

	return content, nil
}

// WriteUsers writes the users to the users.json file
func WriteUsers(content map[string]any) error {

	// Create the users.json file
	filePath := filepath.Join(Settings.DataPath, "users.json")
	file, err := os.Create(filePath)
	if err != nil {
		Logger.Printf("Error creating users.json: %v", err)
		return fmt.Errorf("internal server error when trying to create users.json")
	}
	defer file.Close()

	// Write the content to the file
	var encoder *json.Encoder
	if Settings.Indent > 0 {
		encoder = json.NewEncoder(file)
		encoder.SetIndent("", fmt.Sprintf("%*s", Settings.Indent, ""))
	} else {
		encoder = json.NewEncoder(file)
	}

	if err := encoder.Encode(content); err != nil {
		Logger.Printf("Error encoding users.json: %v", err)
		return fmt.Errorf("internal server error when trying to encode users.json")
	}

	return nil
}

// GetMonth retrieves the logs for a specific month
func GetMonth(userID int, year, month int) (map[string]any, error) {
	// Try to open the month.json file
	filePath := filepath.Join(Settings.DataPath, fmt.Sprintf("%d/%d/%02d.json", userID, year, month))
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		Logger.Printf("Error opening %s: %v", filePath, err)
		return nil, fmt.Errorf("internal server error when trying to open %d/%02d.json", year, month)
	}
	defer file.Close()

	// Read the file content
	var content map[string]any
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&content); err != nil {
		if err == io.EOF {
			return map[string]any{}, nil
		}
		Logger.Printf("Error decoding %s: %v", filePath, err)
		return nil, fmt.Errorf("internal server error when trying to decode %d/%02d.json", year, month)
	}

	return content, nil
}

// WriteMonth writes the logs for a specific month
func WriteMonth(userID int, year, month int, content map[string]any) error {
	// Create the directory if it doesn't exist
	dirPath := filepath.Join(Settings.DataPath, fmt.Sprintf("%d/%d", userID, year))
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		Logger.Printf("Error creating directory %s: %v", dirPath, err)
		return fmt.Errorf("internal server error when trying to create directory %d/%d", userID, year)
	}

	// Create the month.json file
	filePath := filepath.Join(dirPath, fmt.Sprintf("%02d.json", month))
	file, err := os.Create(filePath)
	if err != nil {
		Logger.Printf("Error creating %s: %v", filePath, err)
		return fmt.Errorf("internal server error when trying to create %d/%02d.json", year, month)
	}
	defer file.Close()

	// Write the content to the file
	var encoder *json.Encoder
	if Settings.Indent > 0 {
		encoder = json.NewEncoder(file)
		encoder.SetIndent("", fmt.Sprintf("%*s", Settings.Indent, ""))
	} else {
		encoder = json.NewEncoder(file)
	}

	if err := encoder.Encode(content); err != nil {
		Logger.Printf("Error encoding %s: %v", filePath, err)
		return fmt.Errorf("internal server error when trying to encode %d/%02d.json", year, month)
	}

	return nil
}

// GetTags retrieves the tags for a specific user
func GetTags(userID int) (map[string]any, error) {
	// Try to open the tags.json file
	filePath := filepath.Join(Settings.DataPath, fmt.Sprintf("%d/tags.json", userID))
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		Logger.Printf("Error opening %s: %v", filePath, err)
		return nil, fmt.Errorf("internal server error when trying to open tags.json")
	}
	defer file.Close()

	// Read the file content
	var content map[string]any
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&content); err != nil {
		if err == io.EOF {
			return map[string]any{}, nil
		}
		Logger.Printf("Error decoding %s: %v", filePath, err)
		return nil, fmt.Errorf("internal server error when trying to decode tags.json")
	}

	return content, nil
}

// WriteTags writes the tags for a specific user
func WriteTags(userID int, content map[string]any) error {
	// Create the directory if it doesn't exist
	dirPath := filepath.Join(Settings.DataPath, fmt.Sprintf("%d", userID))
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		Logger.Printf("Error creating directory %s: %v", dirPath, err)
		return fmt.Errorf("internal server error when trying to create directory %d", userID)
	}

	// Create the tags.json file
	filePath := filepath.Join(dirPath, "tags.json")
	file, err := os.Create(filePath)
	if err != nil {
		Logger.Printf("Error creating %s: %v", filePath, err)
		return fmt.Errorf("internal server error when trying to create tags.json")
	}
	defer file.Close()

	// Write the content to the file
	var encoder *json.Encoder
	if Settings.Development && Settings.Indent > 0 {
		encoder = json.NewEncoder(file)
		encoder.SetIndent("", fmt.Sprintf("%*s", Settings.Indent, ""))
	} else {
		encoder = json.NewEncoder(file)
	}

	if err := encoder.Encode(content); err != nil {
		Logger.Printf("Error encoding %s: %v", filePath, err)
		return fmt.Errorf("internal server error when trying to encode tags.json")
	}

	return nil
}

// GetUserSettings retrieves the settings for a specific user
func GetUserSettings(userID int) (string, error) {
	userSettingsMutex.RLock()
	defer userSettingsMutex.RUnlock()

	// Try to open the settings.encrypted file
	filePath := filepath.Join(Settings.DataPath, fmt.Sprintf("%d/settings.encrypted", userID))
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		Logger.Printf("Error opening %s: %v", filePath, err)
		return "", fmt.Errorf("internal server error when trying to open settings.encrypted")
	}
	defer file.Close()

	// Read the file content
	content, err := io.ReadAll(file)
	if err != nil {
		Logger.Printf("Error reading %s: %v", filePath, err)
		return "", fmt.Errorf("internal server error when trying to read settings.encrypted")
	}

	return string(content), nil
}

// WriteUserSettings writes the settings for a specific user
func WriteUserSettings(userID int, content string) error {
	userSettingsMutex.Lock()
	defer userSettingsMutex.Unlock()

	// Create the directory if it doesn't exist
	dirPath := filepath.Join(Settings.DataPath, fmt.Sprintf("%d", userID))
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		Logger.Printf("Error creating directory %s: %v", dirPath, err)
		return fmt.Errorf("internal server error when trying to create directory %d", userID)
	}

	// Create the settings.encrypted file
	filePath := filepath.Join(dirPath, "settings.encrypted")
	file, err := os.Create(filePath)
	if err != nil {
		Logger.Printf("Error creating %s: %v", filePath, err)
		return fmt.Errorf("internal server error when trying to create settings.encrypted")
	}
	defer file.Close()

	// Write the content to the file
	if _, err := file.WriteString(content); err != nil {
		Logger.Printf("Error writing %s: %v", filePath, err)
		return fmt.Errorf("internal server error when trying to write settings.encrypted")
	}

	return nil
}

// GetTemplates retrieves the templates for a specific user
func GetTemplates(userID int) (map[string]any, error) {
	// Try to open the templates.json file
	filePath := filepath.Join(Settings.DataPath, fmt.Sprintf("%d/templates.json", userID))
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			Logger.Printf("%s - File not found", filePath)
			return map[string]any{}, nil
		}
		Logger.Printf("Error opening %s: %v", filePath, err)
		return nil, fmt.Errorf("internal server error when trying to open templates.json")
	}
	defer file.Close()

	// Read the file content
	var content map[string]any
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&content); err != nil {
		if err == io.EOF {
			return map[string]any{}, nil
		}
		Logger.Printf("Error decoding %s: %v", filePath, err)
		return nil, fmt.Errorf("internal server error when trying to decode templates.json")
	}

	return content, nil
}

// WriteTemplates writes the templates for a specific user
func WriteTemplates(userID int, content map[string]any) error {
	// Create the directory if it doesn't exist
	dirPath := filepath.Join(Settings.DataPath, fmt.Sprintf("%d", userID))
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		Logger.Printf("Error creating directory %s: %v", dirPath, err)
		return fmt.Errorf("internal server error when trying to create directory %d", userID)
	}

	// Create the templates.json file
	filePath := filepath.Join(dirPath, "templates.json")
	file, err := os.Create(filePath)
	if err != nil {
		Logger.Printf("Error creating %s: %v", filePath, err)
		return fmt.Errorf("internal server error when trying to create templates.json")
	}
	defer file.Close()

	// Write the content to the file
	var encoder *json.Encoder
	if Settings.Development && Settings.Indent > 0 {
		encoder = json.NewEncoder(file)
		encoder.SetIndent("", fmt.Sprintf("%*s", Settings.Indent, ""))
	} else {
		encoder = json.NewEncoder(file)
	}

	if err := encoder.Encode(content); err != nil {
		Logger.Printf("Error encoding %s: %v", filePath, err)
		return fmt.Errorf("internal server error when trying to encode templates.json")
	}

	return nil
}

// WriteFile writes a file for a specific user
func WriteFile(content []byte, userID int, uuid string) error {
	// Create the directory if it doesn't exist
	dirPath := filepath.Join(Settings.DataPath, fmt.Sprintf("%d/files", userID))
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		Logger.Printf("Error creating directory %s: %v", dirPath, err)
		return fmt.Errorf("internal server error when trying to create directory %d/files", userID)
	}

	// Create the file
	filePath := filepath.Join(dirPath, uuid)
	file, err := os.Create(filePath)
	if err != nil {
		Logger.Printf("Error creating %s: %v", filePath, err)
		return fmt.Errorf("internal server error when trying to create file %s", uuid)
	}
	defer file.Close()

	// Write the content to the file
	if _, err := file.Write(content); err != nil {
		Logger.Printf("Error writing %s: %v", filePath, err)
		return fmt.Errorf("internal server error when trying to write file %s", uuid)
	}

	return nil
}

// ReadFile reads a file for a specific user
func ReadFile(userID int, uuid string) ([]byte, error) {
	// Try to open the file
	filePath := filepath.Join(Settings.DataPath, fmt.Sprintf("%d/files/%s", userID, uuid))
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			Logger.Printf("%s - File not found", filePath)
			return nil, fmt.Errorf("file not found")
		}
		Logger.Printf("Error opening %s: %v", filePath, err)
		return nil, fmt.Errorf("internal server error when trying to open file %s", uuid)
	}
	defer file.Close()

	// Read the file content
	content, err := io.ReadAll(file)
	if err != nil {
		Logger.Printf("Error reading %s: %v", filePath, err)
		return nil, fmt.Errorf("internal server error when trying to read file %s", uuid)
	}

	return content, nil
}

// RemoveFile removes a file for a specific user
func RemoveFile(userID int, uuid string) error {
	// Try to remove the file
	filePath := filepath.Join(Settings.DataPath, fmt.Sprintf("%d/files/%s", userID, uuid))
	if err := os.Remove(filePath); err != nil {
		Logger.Printf("Error removing %s: %v", filePath, err)
		return fmt.Errorf("internal server error when trying to remove file %s", uuid)
	}

	return nil
}

// GetYears returns the years available for a specific user
func GetYears(userID int) ([]string, error) {
	// Try to read the user directory
	dirPath := filepath.Join(Settings.DataPath, fmt.Sprintf("%d", userID))
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			Logger.Printf("%s - Directory not found", dirPath)
			return []string{}, nil
		}
		Logger.Printf("Error reading directory %s: %v", dirPath, err)
		return nil, fmt.Errorf("internal server error when trying to read directory %d", userID)
	}

	// Filter years
	years := []string{}
	for _, entry := range entries {
		if entry.IsDir() && len(entry.Name()) == 4 {
			// Check if the name is a valid year (4 digits)
			if _, err := strconv.Atoi(entry.Name()); err == nil {
				years = append(years, entry.Name())
			}
		}
	}

	return years, nil
}

// GetMonths returns the months available for a specific user and year
func GetMonths(userID int, year string) ([]string, error) {
	// Try to read the year directory
	dirPath := filepath.Join(Settings.DataPath, fmt.Sprintf("%d/%s", userID, year))
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			Logger.Printf("%s - Directory not found", dirPath)
			return []string{}, nil
		}
		Logger.Printf("Error reading directory %s: %v", dirPath, err)
		return nil, fmt.Errorf("internal server error when trying to read directory %d/%s", userID, year)
	}

	// Filter months
	months := []string{}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			// Extract month from filename (remove .json)
			month := strings.TrimSuffix(entry.Name(), ".json")
			months = append(months, month)
		}
	}

	return months, nil
}

func DeleteUserData(userID int) error {
	// Try to remove the user directory
	dirPath := filepath.Join(Settings.DataPath, strconv.Itoa(userID))
	if err := os.RemoveAll(dirPath); err != nil {
		Logger.Printf("Error removing directory %s: %v", dirPath, err)
		return fmt.Errorf("internal server error when trying to remove user data for ID %d", userID)
	}

	return nil
}

// saves the hash, salt and encrypted derived key of the backup codes to the users.json file
func SaveBackupCodes(userID int, codes []map[string]any) error {
	UsersFileMutex.Lock()
	defer UsersFileMutex.Unlock()

	// Get the current users
	users, err := GetUsers()
	if err != nil {
		return fmt.Errorf("error getting users: %v", err)
	}

	// Find the user with the given ID in the users array
	usersList, ok := users["users"].([]any)
	if !ok {
		return fmt.Errorf("invalid users format, 'users' is not an array")
	}

	var foundUser map[string]any
	for _, u := range usersList {
		uMap, ok := u.(map[string]any)
		if !ok {
			continue
		}
		if id, ok := uMap["user_id"].(float64); ok && int(id) == userID {
			foundUser = uMap
			break
		}
	}

	if foundUser == nil {
		return fmt.Errorf("user with ID %d does not exist", userID)
	}

	// Save the backup codes to the user's data
	foundUser["backup_codes"] = codes

	// Write the updated users back to the file
	if err := WriteUsers(users); err != nil {
		return fmt.Errorf("error writing users: %v", err)
	}

	return nil
}


func GetChangelog() (map[string]any, error) {
	// Try to open the changelog.json file
	filePath := "changelog.json"
	file, err := os.Open(filePath)
	if err != nil {
		Logger.Printf("Error opening %s: %v", filePath, err)
		return nil, fmt.Errorf("internal server error when trying to open changelog.json")
	}
	defer file.Close()

	// Read the file content
	var content map[string]any
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&content); err != nil {
		if err == io.EOF {
			return map[string]any{}, nil
		}
		Logger.Printf("Error decoding %s: %v", filePath, err)
		return nil, fmt.Errorf("internal server error when trying to decode changelog.json")
	}
	
	return content, nil
}