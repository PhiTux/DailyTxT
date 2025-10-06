package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/phitux/dailytxt/backend/utils"
)

// UploadFile handles uploading a file
func UploadFile(w http.ResponseWriter, r *http.Request) {
	// Get user ID and derived key from context
	userID, ok := r.Context().Value(utils.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	derivedKey, ok := r.Context().Value(utils.DerivedKeyKey).(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse form
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
		http.Error(w, fmt.Sprintf("Error parsing form: %v", err), http.StatusBadRequest)
		return
	}

	// Get form values
	dayStr := r.FormValue("day")
	if dayStr == "" {
		http.Error(w, "Missing day parameter", http.StatusBadRequest)
		return
	}
	day, err := strconv.Atoi(dayStr)
	if err != nil {
		http.Error(w, "Invalid day parameter", http.StatusBadRequest)
		return
	}

	monthStr := r.FormValue("month")
	if monthStr == "" {
		http.Error(w, "Missing month parameter", http.StatusBadRequest)
		return
	}
	month, err := strconv.Atoi(monthStr)
	if err != nil {
		http.Error(w, "Invalid month parameter", http.StatusBadRequest)
		return
	}

	yearStr := r.FormValue("year")
	if yearStr == "" {
		http.Error(w, "Missing year parameter", http.StatusBadRequest)
		return
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		http.Error(w, "Invalid year parameter", http.StatusBadRequest)
		return
	}

	uuid := r.FormValue("uuid")
	if uuid == "" {
		http.Error(w, "Missing uuid parameter", http.StatusBadRequest)
		return
	}

	// Get file
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting file: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Get encryption key first (before reading large file)
	encKey, err := utils.GetEncryptionKey(userID, derivedKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting encryption key: %v", err), http.StatusInternalServerError)
		return
	}

	// Read file into a buffer (more memory efficient)
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading file: %v", err), http.StatusInternalServerError)
		return
	}
	// Ensure fileBytes is cleared when function exits
	defer func() { fileBytes = nil }()

	// Encrypt file
	encryptedFile, err := utils.EncryptFile(fileBytes, encKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error encrypting file: %v", err), http.StatusInternalServerError)
		return
	}
	// Ensure encryptedFile is cleared when function exits
	defer func() { encryptedFile = nil }()

	// Clear original file data from memory immediately after encryption
	fileBytes = nil

	// Write file
	if err := utils.WriteFile(encryptedFile, userID, uuid); err != nil {
		http.Error(w, fmt.Sprintf("Error writing file: %v", err), http.StatusInternalServerError)
		return
	}

	// Clear encrypted data from memory immediately after writing
	encryptedFile = nil

	// Get month data
	content, err := utils.GetMonth(userID, year, month)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving month data: %v", err), http.StatusInternalServerError)
		return
	}

	// Encrypt filename
	encFilename, err := utils.EncryptText(header.Filename, encKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error encrypting filename: %v", err), http.StatusInternalServerError)
		return
	}

	// Create new file entry
	newFile := map[string]any{
		"enc_filename":  encFilename,
		"uuid_filename": uuid,
		"size":          header.Size,
	}

	// Add file to day
	days, ok := content["days"].([]any)
	if !ok {
		days = []any{}
	}

	dayFound := false
	for i, dayInterface := range days {
		dayObj, ok := dayInterface.(map[string]any)
		if !ok {
			continue
		}

		dayNum, ok := dayObj["day"].(float64)
		if !ok || int(dayNum) != day {
			continue
		}

		// Add file to existing day
		dayFound = true
		files, ok := dayObj["files"].([]any)
		if !ok {
			files = []any{}
		}
		files = append(files, newFile)
		dayObj["files"] = files
		days[i] = dayObj
		break
	}

	if !dayFound {
		// Create new day with file
		days = append(days, map[string]any{
			"day":   day,
			"files": []any{newFile},
		})
	}

	// Update days array
	content["days"] = days

	// Write month data
	if err := utils.WriteMonth(userID, year, month, content); err != nil {
		// Cleanup on error
		utils.RemoveFile(userID, uuid)
		http.Error(w, fmt.Sprintf("Error writing month data: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success
	utils.JSONResponse(w, http.StatusOK, map[string]bool{
		"success": true,
	})
}

// DownloadFile handles downloading a file
func DownloadFile(w http.ResponseWriter, r *http.Request) {
	// Get user ID and derived key from context
	userID, ok := r.Context().Value(utils.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	derivedKey, ok := r.Context().Value(utils.DerivedKeyKey).(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get uuid parameter
	uuid := r.URL.Query().Get("uuid")
	if uuid == "" {
		http.Error(w, "Missing uuid parameter", http.StatusBadRequest)
		return
	}

	// Get encryption key
	encKey, err := utils.GetEncryptionKey(userID, derivedKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting encryption key: %v", err), http.StatusInternalServerError)
		return
	}

	// Read file
	encryptedFile, err := utils.ReadFile(userID, uuid)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading file: %v", err), http.StatusInternalServerError)
		return
	}
	// Ensure encryptedFile is cleared when function exits
	defer func() { encryptedFile = nil }()

	// Decrypt file
	decryptedFile, err := utils.DecryptFile(encryptedFile, encKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error decrypting file: %v", err), http.StatusInternalServerError)
		return
	}
	// Ensure decryptedFile is cleared when function exits
	defer func() { decryptedFile = nil }()

	// Clear encrypted data from memory immediately after decryption
	encryptedFile = nil

	// Set response headers for streaming
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment")

	// Write file to response
	if _, err := w.Write(decryptedFile); err != nil {
		http.Error(w, fmt.Sprintf("Error writing response: %v", err), http.StatusInternalServerError)
		return
	}
}

// DeleteFile handles deleting a file
func DeleteFile(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value(utils.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get parameters
	uuid := r.URL.Query().Get("uuid")
	if uuid == "" {
		http.Error(w, "Missing uuid parameter", http.StatusBadRequest)
		return
	}

	dayStr := r.URL.Query().Get("day")
	if dayStr == "" {
		http.Error(w, "Missing day parameter", http.StatusBadRequest)
		return
	}
	day, err := strconv.Atoi(dayStr)
	if err != nil {
		http.Error(w, "Invalid day parameter", http.StatusBadRequest)
		return
	}

	monthStr := r.URL.Query().Get("month")
	if monthStr == "" {
		http.Error(w, "Missing month parameter", http.StatusBadRequest)
		return
	}
	month, err := strconv.Atoi(monthStr)
	if err != nil {
		http.Error(w, "Invalid month parameter", http.StatusBadRequest)
		return
	}

	yearStr := r.URL.Query().Get("year")
	if yearStr == "" {
		http.Error(w, "Missing year parameter", http.StatusBadRequest)
		return
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		http.Error(w, "Invalid year parameter", http.StatusBadRequest)
		return
	}

	// Get month data
	content, err := utils.GetMonth(userID, year, month)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving month data: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if days exist
	days, ok := content["days"].([]any)
	if !ok {
		http.Error(w, "Day not found - json error", http.StatusInternalServerError)
		return
	}

	// Find day and file
	fileFound := false
	for i, dayInterface := range days {
		dayObj, ok := dayInterface.(map[string]any)
		if !ok {
			continue
		}

		dayNum, ok := dayObj["day"].(float64)
		if !ok || int(dayNum) != day {
			continue
		}

		// Check for files
		files, ok := dayObj["files"].([]any)
		if !ok {
			continue
		}

		// Find file
		for j, fileInterface := range files {
			file, ok := fileInterface.(map[string]any)
			if !ok {
				continue
			}

			uuidFilename, ok := file["uuid_filename"].(string)
			if !ok || uuidFilename != uuid {
				continue
			}

			// Remove file from array
			if err := utils.RemoveFile(userID, uuid); err != nil {
				http.Error(w, fmt.Sprintf("Failed to delete file: %v", err), http.StatusInternalServerError)
				return
			}

			files = append(files[:j], files[j+1:]...)
			dayObj["files"] = files
			days[i] = dayObj
			fileFound = true
			break
		}

		if fileFound {
			break
		}
	}

	if !fileFound {
		http.Error(w, "Failed to delete file - not found in log", http.StatusInternalServerError)
		return
	}

	// Update days array
	content["days"] = days

	// Write month data
	if err := utils.WriteMonth(userID, year, month, content); err != nil {
		http.Error(w, fmt.Sprintf("Failed to write changes of deleted file: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success
	utils.JSONResponse(w, http.StatusOK, map[string]bool{
		"success": true,
	})
}

// RenameFileRequest represents the rename file request body
type RenameFileRequest struct {
	UUID        string `json:"uuid"`
	NewFilename string `json:"new_filename"`
	Day         int    `json:"day"`
	Month       int    `json:"month"`
	Year        int    `json:"year"`
}

// RenameFile handles renaming a file
func RenameFile(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value(utils.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	derivedKey, ok := r.Context().Value(utils.DerivedKeyKey).(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req RenameFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	req.NewFilename = strings.TrimSpace(req.NewFilename)
	if req.NewFilename == "" {
		utils.JSONResponse(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"message": "New filename cannot be empty",
		})
		return
	}

	if req.UUID == "" {
		utils.JSONResponse(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"message": "File UUID is required",
		})
		return
	}

	// Get month data
	content, err := utils.GetMonth(userID, req.Year, req.Month)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving month data: %v", err), http.StatusInternalServerError)
		return
	}

	encKey, err := utils.GetEncryptionKey(userID, derivedKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting encryption key: %v", err), http.StatusInternalServerError)
		return
	}

	enc_filename, err := utils.EncryptText(req.NewFilename, encKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error encrypting text: %v", err), http.StatusInternalServerError)
		return
	}

	// Find and update the file
	days, ok := content["days"].([]any)
	if !ok {
		utils.JSONResponse(w, http.StatusNotFound, map[string]any{
			"success": false,
			"message": "No days found",
		})
		return
	}

	found := false
	for _, d := range days {
		day, ok := d.(map[string]any)
		if !ok {
			continue
		}

		dayNum, ok := day["day"].(float64)
		if !ok || int(dayNum) != req.Day {
			continue
		}

		files, ok := day["files"].([]any)
		if !ok {
			continue
		}

		// Find and rename the specific file
		for _, f := range files {
			file, ok := f.(map[string]any)
			if !ok {
				continue
			}

			if uuid, ok := file["uuid_filename"].(string); ok && uuid == req.UUID {
				file["enc_filename"] = enc_filename
				found = true
				break
			}
		}

		if found {
			break
		}
	}

	if !found {
		utils.JSONResponse(w, http.StatusNotFound, map[string]any{
			"success": false,
			"message": "File not found",
		})
		return
	}

	// Save the updated month data
	if err := utils.WriteMonth(userID, req.Year, req.Month, content); err != nil {
		http.Error(w, fmt.Sprintf("Error writing month data: %v", err), http.StatusInternalServerError)
		return
	}

	utils.Logger.Printf("File renamed successfully for user %d: %s -> %s", userID, req.UUID, req.NewFilename)
	utils.JSONResponse(w, http.StatusOK, map[string]bool{"success": true})
}

// ReorderFilesRequest represents the reorder files request body
type ReorderFilesRequest struct {
	Day       int            `json:"day"`
	Month     int            `json:"month"`
	Year      int            `json:"year"`
	FileOrder map[string]int `json:"file_order"` // UUID -> order index
}

// ReorderFiles handles reordering files within a day
func ReorderFiles(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value(utils.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req ReorderFilesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.FileOrder) == 0 {
		utils.JSONResponse(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"message": "File order mapping is required",
		})
		return
	}

	// Get month data
	content, err := utils.GetMonth(userID, req.Year, req.Month)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving month data: %v", err), http.StatusInternalServerError)
		return
	}

	// Find and reorder files for the specific day
	days, ok := content["days"].([]any)
	if !ok {
		utils.JSONResponse(w, http.StatusNotFound, map[string]any{
			"success": false,
			"message": "No days found",
		})
		return
	}

	found := false
	for _, d := range days {
		day, ok := d.(map[string]any)
		if !ok {
			continue
		}

		dayNum, ok := day["day"].(float64)
		if !ok || int(dayNum) != req.Day {
			continue
		}

		files, ok := day["files"].([]any)
		if !ok {
			continue
		}

		// Create a slice to hold files with their new order
		type fileWithOrder struct {
			file  map[string]any
			order int
		}

		var orderedFiles []fileWithOrder

		// Assign order to each file
		for _, f := range files {
			file, ok := f.(map[string]any)
			if !ok {
				continue
			}

			uuid, ok := file["uuid_filename"].(string)
			if !ok {
				continue
			}

			if order, exists := req.FileOrder[uuid]; exists {
				orderedFiles = append(orderedFiles, fileWithOrder{file: file, order: order})
			} else {
				// Files not in the reorder map get appended at the end
				orderedFiles = append(orderedFiles, fileWithOrder{file: file, order: len(req.FileOrder)})
			}
		}

		// Sort files by their order
		for i := 0; i < len(orderedFiles)-1; i++ {
			for j := i + 1; j < len(orderedFiles); j++ {
				if orderedFiles[i].order > orderedFiles[j].order {
					orderedFiles[i], orderedFiles[j] = orderedFiles[j], orderedFiles[i]
				}
			}
		}

		// Update the files array with the new order
		newFiles := make([]any, len(orderedFiles))
		for i, fileWithOrder := range orderedFiles {
			newFiles[i] = fileWithOrder.file
		}
		day["files"] = newFiles

		found = true
		break
	}

	if !found {
		utils.JSONResponse(w, http.StatusNotFound, map[string]any{
			"success": false,
			"message": "Day not found",
		})
		return
	}

	// Save the updated month data
	if err := utils.WriteMonth(userID, req.Year, req.Month, content); err != nil {
		http.Error(w, fmt.Sprintf("Error writing month data: %v", err), http.StatusInternalServerError)
		return
	}

	utils.JSONResponse(w, http.StatusOK, map[string]bool{"success": true})
}
