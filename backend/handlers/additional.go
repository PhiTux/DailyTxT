package handlers

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	htmlpkg "html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/phitux/dailytxt/backend/utils"
)

// EditTagRequest represents the edit tag request
type EditTagRequest struct {
	ID    int    `json:"id"`
	Icon  string `json:"icon"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// EditTag handles editing a tag
func EditTag(w http.ResponseWriter, r *http.Request) {
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

	// Parse request body
	var req EditTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get tags
	content, err := utils.GetTags(userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving tags: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if tags exist
	tags, ok := content["tags"].([]any)
	if !ok {
		http.Error(w, "Tag not found - json error", http.StatusInternalServerError)
		return
	}

	// Get encryption key
	encKey, err := utils.GetEncryptionKey(userID, derivedKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting encryption key: %v", err), http.StatusInternalServerError)
		return
	}

	// Find and update tag
	found := false
	for i, tagInterface := range tags {
		tag, ok := tagInterface.(map[string]any)
		if !ok {
			continue
		}

		if id, ok := tag["id"].(float64); ok && int(id) == req.ID {
			// Encrypt tag data
			encIcon, err := utils.EncryptText(req.Icon, encKey)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error encrypting tag icon: %v", err), http.StatusInternalServerError)
				return
			}

			encName, err := utils.EncryptText(req.Name, encKey)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error encrypting tag name: %v", err), http.StatusInternalServerError)
				return
			}

			encColor, err := utils.EncryptText(req.Color, encKey)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error encrypting tag color: %v", err), http.StatusInternalServerError)
				return
			}

			// Update tag
			tag["icon"] = encIcon
			tag["name"] = encName
			tag["color"] = encColor
			tags[i] = tag
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Tag not found - not in tags", http.StatusInternalServerError)
		return
	}

	// Write tags
	if err := utils.WriteTags(userID, content); err != nil {
		http.Error(w, fmt.Sprintf("Failed to write tag - error writing tags: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success
	utils.JSONResponse(w, http.StatusOK, map[string]bool{
		"success": true,
	})
}

// DeleteTag handles deleting a tag
func DeleteTag(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value(utils.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get tag ID
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid id parameter", http.StatusBadRequest)
		return
	}

	// Get all years and months
	years, err := utils.GetYears(userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving years: %v", err), http.StatusInternalServerError)
		return
	}

	// Remove tag from all logs
	for _, year := range years {
		yearInt, _ := strconv.Atoi(year)
		months, err := utils.GetMonths(userID, year)
		if err != nil {
			continue
		}

		for _, month := range months {
			monthInt, _ := strconv.Atoi(month)
			content, err := utils.GetMonth(userID, yearInt, monthInt)
			if err != nil {
				continue
			}

			days, ok := content["days"].([]any)
			if !ok {
				continue
			}

			// Check each day for the tag
			modified := false
			for i, dayInterface := range days {
				day, ok := dayInterface.(map[string]any)
				if !ok {
					continue
				}

				tags, ok := day["tags"].([]any)
				if !ok {
					continue
				}

				// Find and remove the tag
				for j, tagID := range tags {
					if tagIDFloat, ok := tagID.(float64); ok && int(tagIDFloat) == id {
						// Remove tag
						tags = append(tags[:j], tags[j+1:]...)
						day["tags"] = tags
						days[i] = day
						modified = true
						break
					}
				}
			}

			// Write updated month if modified
			if modified {
				content["days"] = days
				if err := utils.WriteMonth(userID, yearInt, monthInt, content); err != nil {
					http.Error(w, fmt.Sprintf("Failed to delete tag - error writing log: %v", err), http.StatusInternalServerError)
					return
				}
			}
		}
	}

	// Get tags
	content, err := utils.GetTags(userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving tags: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if tags exist
	tags, ok := content["tags"].([]any)
	if !ok {
		http.Error(w, "Tag not found - json error", http.StatusInternalServerError)
		return
	}

	// Find and remove tag
	found := false
	for i, tagInterface := range tags {
		tag, ok := tagInterface.(map[string]any)
		if !ok {
			continue
		}

		if tagID, ok := tag["id"].(float64); ok && int(tagID) == id {
			// Remove tag
			tags = append(tags[:i], tags[i+1:]...)
			content["tags"] = tags
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Tag not found - not in tags", http.StatusInternalServerError)
		return
	}

	// Write tags
	if err := utils.WriteTags(userID, content); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete tag - error writing tags: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success
	utils.JSONResponse(w, http.StatusOK, map[string]bool{
		"success": true,
	})
}

// TagLogRequest represents the tag log request
type TagLogRequest struct {
	Day   int `json:"day"`
	Month int `json:"month"`
	Year  int `json:"year"`
	TagID int `json:"tag_id"`
}

// AddTagToLog handles adding a tag to a log
func AddTagToLog(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value(utils.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req TagLogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get month data
	content, err := utils.GetMonth(userID, req.Year, req.Month)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving month data: %v", err), http.StatusInternalServerError)
		return
	}

	// Get or create days array
	days, ok := content["days"].([]any)
	if !ok {
		days = []any{}
	}

	// Find day
	dayFound := false
	for i, dayInterface := range days {
		day, ok := dayInterface.(map[string]any)
		if !ok {
			continue
		}

		dayNum, ok := day["day"].(float64)
		if !ok || int(dayNum) != req.Day {
			continue
		}

		// Day found, add tag
		dayFound = true
		tags, ok := day["tags"].([]any)
		if !ok {
			tags = []any{}
		}

		// Check if tag already exists
		tagExists := false
		for _, tagID := range tags {
			if tagIDFloat, ok := tagID.(float64); ok && int(tagIDFloat) == req.TagID {
				tagExists = true
				break
			}
		}

		if !tagExists {
			tags = append(tags, float64(req.TagID))
			day["tags"] = tags
			days[i] = day
		}
		break
	}

	if !dayFound {
		// Create new day with tag
		days = append(days, map[string]any{
			"day":  req.Day,
			"tags": []any{float64(req.TagID)},
		})
	}

	// Update days array
	content["days"] = days

	// Write month data
	if err := utils.WriteMonth(userID, req.Year, req.Month, content); err != nil {
		http.Error(w, fmt.Sprintf("Failed to write tag - error writing log: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success
	utils.JSONResponse(w, http.StatusOK, map[string]bool{
		"success": true,
	})
}

// RemoveTagFromLog handles removing a tag from a log
func RemoveTagFromLog(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value(utils.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req TagLogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get month data
	content, err := utils.GetMonth(userID, req.Year, req.Month)
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

	// Find day
	found := false
	for i, dayInterface := range days {
		day, ok := dayInterface.(map[string]any)
		if !ok {
			continue
		}

		dayNum, ok := day["day"].(float64)
		if !ok || int(dayNum) != req.Day {
			continue
		}

		// Day found, check for tags
		tags, ok := day["tags"].([]any)
		if !ok {
			http.Error(w, "Failed to remove tag - not found in log", http.StatusInternalServerError)
			return
		}

		// Find and remove tag
		for j, tagID := range tags {
			if tagIDFloat, ok := tagID.(float64); ok && int(tagIDFloat) == req.TagID {
				// Remove tag
				tags = append(tags[:j], tags[j+1:]...)
				day["tags"] = tags
				days[i] = day
				found = true
				break
			}
		}

		if !found {
			http.Error(w, "Failed to remove tag - not found in log", http.StatusInternalServerError)
			return
		}
		break
	}

	if !found {
		http.Error(w, "Failed to remove tag - not found in log", http.StatusInternalServerError)
		return
	}

	// Update days array
	content["days"] = days

	// Write month data
	if err := utils.WriteMonth(userID, req.Year, req.Month, content); err != nil {
		http.Error(w, fmt.Sprintf("Failed to remove tag - error writing log: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success
	utils.JSONResponse(w, http.StatusOK, map[string]bool{
		"success": true,
	})
}

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

// BookmarkDay handles bookmarking a day
func BookmarkDay(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value(utils.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get parameters
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

	// Get or create days array
	days, ok := content["days"].([]any)
	if !ok {
		days = []any{}
	}

	// Find day
	dayFound := false
	bookmarked := true
	for i, dayInterface := range days {
		dayObj, ok := dayInterface.(map[string]any)
		if !ok {
			continue
		}

		dayNum, ok := dayObj["day"].(float64)
		if !ok || int(dayNum) != day {
			continue
		}

		// Day found, toggle bookmark
		dayFound = true
		if bookmark, ok := dayObj["isBookmarked"].(bool); ok && bookmark {
			dayObj["isBookmarked"] = false
			bookmarked = false
		} else {
			dayObj["isBookmarked"] = true
		}
		days[i] = dayObj
		break
	}

	if !dayFound {
		// Create new day with bookmark
		days = append(days, map[string]any{
			"day":          day,
			"isBookmarked": true,
		})
	}

	// Update days array
	content["days"] = days

	// Write month data
	if err := utils.WriteMonth(userID, year, month, content); err != nil {
		http.Error(w, fmt.Sprintf("Failed to bookmark day - error writing log: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success
	utils.JSONResponse(w, http.StatusOK, map[string]any{
		"success":    true,
		"bookmarked": bookmarked,
	})
}

// SearchTag handles searching logs by tag
func SearchTag(w http.ResponseWriter, r *http.Request) {
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

	// Get parameters
	tagIDStr := r.URL.Query().Get("tag_id")
	if tagIDStr == "" {
		http.Error(w, "Missing tag_id parameter", http.StatusBadRequest)
		return
	}
	tagID, err := strconv.Atoi(tagIDStr)
	if err != nil {
		http.Error(w, "Invalid tag_id parameter", http.StatusBadRequest)
		return
	}

	// Get encryption key
	encKey, err := utils.GetEncryptionKey(userID, derivedKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting encryption key: %v", err), http.StatusInternalServerError)
		return
	}

	// Get all years and months
	years, err := utils.GetYears(userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving years: %v", err), http.StatusInternalServerError)
		return
	}

	// Search for tag
	results := []any{}
	for _, year := range years {
		yearInt, _ := strconv.Atoi(year)
		months, err := utils.GetMonths(userID, year)
		if err != nil {
			continue
		}

		for _, month := range months {
			monthInt, _ := strconv.Atoi(month)
			content, err := utils.GetMonth(userID, yearInt, monthInt)
			if err != nil {
				continue
			}

			days, ok := content["days"].([]any)
			if !ok {
				continue
			}

			// Check each day for the tag
			for _, dayInterface := range days {
				day, ok := dayInterface.(map[string]any)
				if !ok {
					continue
				}

				dayNum, ok := day["day"].(float64)
				if !ok {
					continue
				}

				tags, ok := day["tags"].([]any)
				if !ok {
					continue
				}

				// Check if tag is in tags
				found := false
				for _, t := range tags {
					if tagIDFloat, ok := t.(float64); ok && int(tagIDFloat) == tagID {
						found = true
						break
					}
				}

				if !found {
					continue
				}

				// Get text snippet
				context := ""
				if text, ok := day["text"].(string); ok && text != "" {
					decryptedText, err := utils.DecryptText(text, encKey)
					if err != nil {
						continue
					}
					// Get first few words
					words := strings.Fields(decryptedText)
					if len(words) > 5 {
						context = strings.Join(words[:5], " ")
					} else {
						context = decryptedText
					}
				}

				// Add to results
				results = append(results, map[string]any{
					"year":  yearInt,
					"month": monthInt,
					"day":   int(dayNum),
					"text":  context,
				})
			}
		}
	}

	// Sort results by date
	/*
		sort.Slice(results, func(i, j int) bool {
			ri := results[i].(map[string]any)
			rj := results[j].(map[string]any)

			yearI := ri["year"].(int)
			yearJ := rj["year"].(int)
			if yearI != yearJ {
				return yearI < yearJ
			}

			monthI := ri["month"].(int)
			monthJ := rj["month"].(int)
			if monthI != monthJ {
				return monthI < monthJ
			}

			dayI := ri["day"].(int)
			dayJ := rj["day"].(int)
			return dayI < dayJ
		})
	*/

	// Return results
	utils.JSONResponse(w, http.StatusOK, results)
}

// GetTags handles retrieving a user's tags
func GetTags(w http.ResponseWriter, r *http.Request) {
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

	// Get tags
	content, err := utils.GetTags(userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving tags: %v", err), http.StatusInternalServerError)
		return
	}

	// If no tags, return empty array
	if tags, ok := content["tags"].([]any); !ok || len(tags) == 0 {
		utils.JSONResponse(w, http.StatusOK, []any{})
		return
	}

	// Get encryption key
	encKey, err := utils.GetEncryptionKey(userID, derivedKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting encryption key: %v", err), http.StatusInternalServerError)
		return
	}

	// Decrypt tag data
	tags := content["tags"].([]any)
	result := []any{}

	for _, tagInterface := range tags {
		tag, ok := tagInterface.(map[string]any)
		if !ok {
			continue
		}

		// Decrypt icon, name, and color
		if encIcon, ok := tag["icon"].(string); ok {
			decryptedIcon, err := utils.DecryptText(encIcon, encKey)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error decrypting tag icon: %v", err), http.StatusInternalServerError)
				return
			}
			tag["icon"] = decryptedIcon
		}

		if encName, ok := tag["name"].(string); ok {
			decryptedName, err := utils.DecryptText(encName, encKey)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error decrypting tag name: %v", err), http.StatusInternalServerError)
				return
			}
			tag["name"] = decryptedName
		}

		if encColor, ok := tag["color"].(string); ok {
			decryptedColor, err := utils.DecryptText(encColor, encKey)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error decrypting tag color: %v", err), http.StatusInternalServerError)
				return
			}
			tag["color"] = decryptedColor
		}

		result = append(result, tag)
	}

	// Return tags
	utils.JSONResponse(w, http.StatusOK, result)
}

// TagRequest represents a tag request
type TagRequest struct {
	Icon  string `json:"icon"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// SaveTags handles saving a new tag
func SaveTags(w http.ResponseWriter, r *http.Request) {
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

	// Parse request body
	var req TagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get tags
	content, err := utils.GetTags(userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving tags: %v", err), http.StatusInternalServerError)
		return
	}

	// Create tags array if it doesn't exist
	if _, ok := content["tags"]; !ok {
		content["tags"] = []any{}
	}
	if _, ok := content["next_id"]; !ok {
		content["next_id"] = 1
	}

	// Get encryption key
	encKey, err := utils.GetEncryptionKey(userID, derivedKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting encryption key: %v", err), http.StatusInternalServerError)
		return
	}

	// Encrypt tag data
	encIcon, err := utils.EncryptText(req.Icon, encKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error encrypting tag icon: %v", err), http.StatusInternalServerError)
		return
	}

	encName, err := utils.EncryptText(req.Name, encKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error encrypting tag name: %v", err), http.StatusInternalServerError)
		return
	}

	encColor, err := utils.EncryptText(req.Color, encKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error encrypting tag color: %v", err), http.StatusInternalServerError)
		return
	}

	// Create new tag
	nextID, ok := content["next_id"].(float64)
	if !ok {
		nextID = 1
	}

	newTag := map[string]any{
		"id":    int(nextID),
		"icon":  encIcon,
		"name":  encName,
		"color": encColor,
	}

	// Add tag to tags array
	tags, ok := content["tags"].([]any)
	if !ok {
		tags = []any{}
	}
	tags = append(tags, newTag)
	content["tags"] = tags
	content["next_id"] = nextID + 1

	// Write tags
	if err := utils.WriteTags(userID, content); err != nil {
		http.Error(w, fmt.Sprintf("Error writing tags: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success
	utils.JSONResponse(w, http.StatusOK, map[string]bool{
		"success": true,
	})
}

// Helper functions for search
func getStartIndex(text string, index int) int {
	if index == 0 {
		return 0
	}

	for i := 0; i < 3; i++ {
		startIndex := strings.LastIndex(text[:index-1], " ")
		index = startIndex
		if startIndex == -1 {
			return 0
		}
	}

	return index + 1
}

func getEndIndex(text string, index int) int {
	if index == len(text)-1 {
		return len(text)
	}

	for i := 0; i < 3; i++ {
		endIndex := strings.Index(text[index+1:], " ")
		if endIndex == -1 {
			return len(text)
		}
		index = index + 1 + endIndex
	}

	return index
}

func getContext(text, searchString string, exact bool) string {
	// Replace whitespace with non-breaking space
	re := regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")

	var pos int
	if exact {
		pos = strings.Index(text, searchString)
	} else {
		pos = strings.Index(strings.ToLower(text), strings.ToLower(searchString))
	}

	if pos == -1 {
		return "<em>Dailytxt: Error formatting...</em>"
	}

	start := getStartIndex(text, pos)
	end := getEndIndex(text, pos+len(searchString)-1)
	return text[start:pos] + "<b>" + text[pos:pos+len(searchString)] + "</b>" + text[pos+len(searchString):end]
}

// Search handles searching logs for text
func Search(w http.ResponseWriter, r *http.Request) {
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

	// Get query parameter
	searchString := r.URL.Query().Get("searchString")
	if searchString == "" {
		http.Error(w, "Missing search parameter", http.StatusBadRequest)
		return
	}

	// Get encryption key
	encKey, err := utils.GetEncryptionKey(userID, derivedKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting encryption key: %v", err), http.StatusInternalServerError)
		return
	}

	// Get user directory
	userDir := filepath.Join(utils.Settings.DataPath, strconv.Itoa(userID))
	results := []any{}

	// Traverse all years and months
	yearEntries, err := os.ReadDir(userDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading user directory: %v", err), http.StatusInternalServerError)
		return
	}

	// Regex to match year directories (4 digits)
	yearRegex := regexp.MustCompile(`^\d{4}$`)

	for _, yearEntry := range yearEntries {
		if !yearEntry.IsDir() || !yearRegex.MatchString(yearEntry.Name()) {
			continue
		}
		year := yearEntry.Name()

		// Read month files in year directory
		yearDir := filepath.Join(userDir, year)
		monthEntries, err := os.ReadDir(yearDir)
		if err != nil {
			continue
		}

		// Regex to match month files (2 digits + .json)
		monthRegex := regexp.MustCompile(`^(\d{2})\.json$`)

		for _, monthEntry := range monthEntries {
			if monthEntry.IsDir() {
				continue
			}

			matches := monthRegex.FindStringSubmatch(monthEntry.Name())
			if len(matches) != 2 {
				continue
			}
			month := matches[1]

			// Get month content
			monthInt, _ := strconv.Atoi(month)
			yearInt, _ := strconv.Atoi(year)
			content, err := utils.GetMonth(userID, yearInt, monthInt)
			if err != nil {
				continue
			}

			days, ok := content["days"].([]any)
			if !ok {
				continue
			}

			// Process each day
			for _, dayInterface := range days {
				dayLog, ok := dayInterface.(map[string]any)
				if !ok {
					continue
				}

				dayNum, ok := dayLog["day"].(float64)
				if !ok {
					continue
				}
				day := int(dayNum)

				// Check text
				if text, ok := dayLog["text"].(string); ok {
					decryptedText, err := utils.DecryptText(text, encKey)
					if err != nil {
						continue
					}

					// Apply search logic
					if strings.HasPrefix(searchString, "\"") && strings.HasSuffix(searchString, "\"") {
						// Exact match
						searchTerm := searchString[1 : len(searchString)-1]
						if strings.Contains(decryptedText, searchTerm) {
							context := getContext(decryptedText, searchTerm, true)
							results = append(results, map[string]any{
								"year":  year,
								"month": month,
								"day":   day,
								"text":  context,
							})
						}
					} else if strings.Contains(searchString, "|") {
						// OR search
						words := strings.Split(searchString, "|")
						for _, word := range words {
							wordTrimmed := strings.TrimSpace(word)
							if strings.Contains(strings.ToLower(decryptedText), strings.ToLower(wordTrimmed)) {
								context := getContext(decryptedText, wordTrimmed, false)
								results = append(results, map[string]any{
									"year":  year,
									"month": month,
									"day":   day,
									"text":  context,
								})
								break
							}
						}
					} else if strings.Contains(searchString, " ") {
						// AND search
						words := strings.Split(searchString, " ")
						allWordsMatch := true
						for _, word := range words {
							wordTrimmed := strings.TrimSpace(word)
							if !strings.Contains(strings.ToLower(decryptedText), strings.ToLower(wordTrimmed)) {
								allWordsMatch = false
								break
							}
						}
						if allWordsMatch {
							context := getContext(decryptedText, strings.TrimSpace(words[0]), false)
							results = append(results, map[string]any{
								"year":  year,
								"month": month,
								"day":   day,
								"text":  context,
							})
						}
					} else {
						// Simple search
						if strings.Contains(strings.ToLower(decryptedText), strings.ToLower(searchString)) {
							context := getContext(decryptedText, searchString, false)
							results = append(results, map[string]any{
								"year":  year,
								"month": month,
								"day":   day,
								"text":  context,
							})
						}
					}
				}

				// Check filenames
				if files, ok := dayLog["files"].([]any); ok {
					for _, fileInterface := range files {
						file, ok := fileInterface.(map[string]any)
						if !ok {
							continue
						}

						if encFilename, ok := file["enc_filename"].(string); ok {
							decryptedFilename, err := utils.DecryptText(encFilename, encKey)
							if err != nil {
								continue
							}

							if strings.Contains(strings.ToLower(decryptedFilename), strings.ToLower(searchString)) {
								context := "📎 " + decryptedFilename
								results = append(results, map[string]any{
									"year":  year,
									"month": month,
									"day":   day,
									"text":  context,
								})
								break
							}
						}
					}
				}
			}
		}
	}

	// Sort results by date
	sort.Slice(results, func(i, j int) bool {
		ri := results[i].(map[string]any)
		rj := results[j].(map[string]any)

		yearI, _ := strconv.Atoi(ri["year"].(string))
		yearJ, _ := strconv.Atoi(rj["year"].(string))
		if yearI != yearJ {
			return yearI < yearJ
		}

		monthI, _ := strconv.Atoi(ri["month"].(string))
		monthJ, _ := strconv.Atoi(rj["month"].(string))
		if monthI != monthJ {
			return monthI < monthJ
		}

		dayI := ri["day"].(int)
		dayJ := rj["day"].(int)
		return dayI < dayJ
	})

	// Return results
	utils.JSONResponse(w, http.StatusOK, results)
}

// LogEntry represents a single diary entry for export
type LogEntry struct {
	Year        int
	Month       int
	Day         int
	Text        string
	DateWritten string
	Files       []string
	Tags        []int
}

// ExportData handles exporting user data
func ExportData(w http.ResponseWriter, r *http.Request) {
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

	// Get parameters from URL
	period := r.URL.Query().Get("period")
	if period == "" {
		http.Error(w, "Missing period parameter", http.StatusBadRequest)
		return
	} else if period != "periodAll" && period != "periodVariable" {
		http.Error(w, "Invalid period parameter", http.StatusBadRequest)
		return
	}

	var startYear, startMonth, startDay, endYear, endMonth, endDay int
	var err error

	if period == "periodVariable" {
		startDate := r.URL.Query().Get("startDate")
		if startDate != "" {
			startParts := strings.Split(startDate, "-")
			if len(startParts) != 3 {
				http.Error(w, "Invalid startDate format", http.StatusBadRequest)
				return
			}
			startYear, _ = strconv.Atoi(startParts[0])
			startMonth, _ = strconv.Atoi(startParts[1])
			startDay, _ = strconv.Atoi(startParts[2])
		} else {
			http.Error(w, "Missing startDate parameter", http.StatusBadRequest)
			return
		}

		endDate := r.URL.Query().Get("endDate")
		if endDate != "" {
			endParts := strings.Split(endDate, "-")
			if len(endParts) != 3 {
				http.Error(w, "Invalid endDate format", http.StatusBadRequest)
				return
			}
			endYear, _ = strconv.Atoi(endParts[0])
			endMonth, _ = strconv.Atoi(endParts[1])
			endDay, _ = strconv.Atoi(endParts[2])
		} else {
			http.Error(w, "Missing endDate parameter", http.StatusBadRequest)
			return
		}
	}

	imagesInHTML := r.URL.Query().Get("imagesInHTML") == "true"

	split := r.URL.Query().Get("split")
	if split == "" {
		http.Error(w, "Missing split parameter", http.StatusBadRequest)
		return
	} else if split != "month" && split != "year" && split != "aio" {
		http.Error(w, "Invalid split parameter", http.StatusBadRequest)
		return
	}

	tagsInHTML := r.URL.Query().Get("tagsInHTML") == "true"

	// Get encryption key
	encKey, err := utils.GetEncryptionKey(userID, derivedKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting encryption key: %v", err), http.StatusInternalServerError)
		return
	}

	// Set response headers for ZIP download
	var filename string
	if period == "periodAll" {
		filename = fmt.Sprintf("diary_export_all_%s.zip", time.Now().Format("2006-01-02"))
	} else {
		filename = fmt.Sprintf("diary_export_%d-%02d-%02d_to_%d-%02d-%02d.zip",
			startYear, startMonth, startDay,
			endYear, endMonth, endDay)
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	// Create ZIP writer
	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	// Collect all log entries for HTML generation
	var allEntries []LogEntry
	var yearlyEntries map[int][]LogEntry = make(map[int][]LogEntry)
	var monthlyEntries map[string][]LogEntry = make(map[string][]LogEntry)

	// Track used filenames per directory to avoid conflicts
	var usedFilenamesPerDay map[string]map[string]bool = make(map[string]map[string]bool)

	// Helper function to check if a date is within range
	isDateInRange := func(year, month, day int) bool {
		if period == "periodAll" {
			return true
		}

		// Create time objects for comparison
		targetDate := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		startDateObj := time.Date(startYear, time.Month(startMonth), startDay, 0, 0, 0, 0, time.UTC)
		endDateObj := time.Date(endYear, time.Month(endMonth), endDay, 0, 0, 0, 0, time.UTC)

		return !targetDate.Before(startDateObj) && !targetDate.After(endDateObj)
	}

	// Determine year range to scan
	var minYear, maxYear int
	if period == "periodAll" {
		// For "periodAll", scan a reasonable range
		minYear = 2010
		maxYear = time.Now().Year() + 1
	} else {
		minYear = startYear
		maxYear = endYear
	}

	// Process each year in the range
	for year := minYear; year <= maxYear; year++ {
		startMonthLoop := 1
		endMonthLoop := 12

		// Optimize month range for specific date ranges
		if period == "periodVariable" {
			if year == startYear {
				startMonthLoop = startMonth
			}
			if year == endYear {
				endMonthLoop = endMonth
			}
		}

		for month := startMonthLoop; month <= endMonthLoop; month++ {
			// Get month data
			content, err := utils.GetMonth(userID, year, month)
			if err != nil {
				continue // Skip months that don't exist
			}

			days, ok := content["days"].([]any)
			if !ok {
				continue
			}

			// Process each day in the month
			for _, dayInterface := range days {
				day, ok := dayInterface.(map[string]any)
				if !ok {
					continue
				}

				dayNum, ok := day["day"].(float64)
				if !ok {
					continue
				}

				dayInt := int(dayNum)

				// Check if this specific day is within the date range
				if !isDateInRange(year, month, dayInt) {
					continue
				}

				entry := LogEntry{
					Year:  year,
					Month: month,
					Day:   dayInt,
				}

				// Decrypt text and date_written
				if text, ok := day["text"].(string); ok && text != "" {
					decryptedText, err := utils.DecryptText(text, encKey)
					if err != nil {
						utils.Logger.Printf("Error decrypting text for %d-%d-%d: %v", year, month, dayInt, err)
						continue
					}
					entry.Text = decryptedText

					if dateWritten, ok := day["date_written"].(string); ok && dateWritten != "" {
						decryptedDate, err := utils.DecryptText(dateWritten, encKey)
						if err == nil {
							entry.DateWritten = decryptedDate
						}
					}
				}

				// Process files
				if filesList, ok := day["files"].([]any); ok && len(filesList) > 0 {
					for _, fileInterface := range filesList {
						file, ok := fileInterface.(map[string]any)
						if !ok {
							continue
						}

						fileID, ok := file["uuid_filename"].(string)
						if !ok {
							continue
						}

						encFilename, ok := file["enc_filename"].(string)
						if !ok {
							continue
						}

						// Decrypt filename
						decryptedFilename, err := utils.DecryptText(encFilename, encKey)
						if err != nil {
							utils.Logger.Printf("Error decrypting filename %s: %v", fileID, err)
							continue
						}

						// Read and decrypt file content
						fileContent, err := utils.ReadFile(userID, fileID)
						if err != nil {
							utils.Logger.Printf("Error reading file %s: %v", fileID, err)
							continue
						}

						decryptedContent, err := utils.DecryptFile(fileContent, encKey)
						if err != nil {
							utils.Logger.Printf("Error decrypting file %s: %v", fileID, err)
							continue
						}

						// Create unique filename to avoid conflicts in ZIP
						dayKey := fmt.Sprintf("%d-%02d-%02d", year, month, dayInt)
						if usedFilenamesPerDay[dayKey] == nil {
							usedFilenamesPerDay[dayKey] = make(map[string]bool)
						}
						uniqueFilename := generateUniqueFilename(usedFilenamesPerDay[dayKey], decryptedFilename)

						// Add file to ZIP with unique filename
						filePath := fmt.Sprintf("files/%d-%02d-%02d/%s", year, month, dayInt, uniqueFilename)
						fileWriter, err := zipWriter.Create(filePath)
						if err != nil {
							utils.Logger.Printf("Error creating file in ZIP %s: %v", filePath, err)
							continue
						}

						_, err = fileWriter.Write(decryptedContent)
						if err != nil {
							utils.Logger.Printf("Error writing file to ZIP %s: %v", filePath, err)
							continue
						}

						entry.Files = append(entry.Files, uniqueFilename)
					}
				}

				// Add tags
				if tags, ok := day["tags"].([]any); ok && len(tags) > 0 {
					for _, tag := range tags {
						if tagID, ok := tag.(float64); ok {
							entry.Tags = append(entry.Tags, int(tagID))
						}
					}
				}

				// Add entry if it has content
				if entry.Text != "" || len(entry.Files) > 0 || len(entry.Tags) > 0 {
					allEntries = append(allEntries, entry)

					// Add to yearly collections
					yearlyEntries[year] = append(yearlyEntries[year], entry)

					// Add to monthly collections
					monthKey := fmt.Sprintf("%d-%02d", year, month)
					monthlyEntries[monthKey] = append(monthlyEntries[monthKey], entry)
				}
			}
		}
	}

	// Create HTML files based on split preference
	switch split {
	case "month":
		// Create one HTML per month
		for monthKey, entries := range monthlyEntries {
			if len(entries) > 0 {
				htmlBytes, err := generateHTML(entries, userID, derivedKey, tagsInHTML, imagesInHTML)
				if err != nil {
					utils.Logger.Printf("Error generating HTML for month %s: %v", monthKey, err)
				} else {
					fileName := fmt.Sprintf("diary_%s.html", monthKey)
					htmlWriter, err := zipWriter.Create(fileName)
					if err != nil {
						utils.Logger.Printf("Error creating month HTML in ZIP %s: %v", fileName, err)
					} else {
						_, err = htmlWriter.Write(htmlBytes)
						if err != nil {
							utils.Logger.Printf("Error writing month HTML to ZIP %s: %v", fileName, err)
						}
					}
				}
			}
		}

	case "year":
		// Create one HTML per year
		for year, entries := range yearlyEntries {
			if len(entries) > 0 {
				htmlBytes, err := generateHTML(entries, userID, derivedKey, tagsInHTML, imagesInHTML)
				if err != nil {
					utils.Logger.Printf("Error generating HTML for year %d: %v", year, err)
				} else {
					fileName := fmt.Sprintf("diary_%d.html", year)
					htmlWriter, err := zipWriter.Create(fileName)
					if err != nil {
						utils.Logger.Printf("Error creating year HTML in ZIP %s: %v", fileName, err)
					} else {
						_, err = htmlWriter.Write(htmlBytes)
						if err != nil {
							utils.Logger.Printf("Error writing year HTML to ZIP %s: %v", fileName, err)
						}
					}
				}
			}
		}

	case "aio":
		// Create one single HTML with all entries
		if len(allEntries) > 0 {
			htmlBytes, err := generateHTML(allEntries, userID, derivedKey, tagsInHTML, imagesInHTML)
			if err != nil {
				utils.Logger.Printf("Error generating HTML: %v", err)
			} else {
				// Add HTML to ZIP
				htmlWriter, err := zipWriter.Create("diary_export.html")
				if err != nil {
					utils.Logger.Printf("Error creating HTML in ZIP: %v", err)
				} else {
					_, err = htmlWriter.Write(htmlBytes)
					if err != nil {
						utils.Logger.Printf("Error writing HTML to ZIP: %v", err)
					}
				}
			}
		}
	}
}

// generateHTML creates an HTML document with all diary entries
func generateHTML(entries []LogEntry, userID int, derivedKey string, includeTags bool, includeImages bool) ([]byte, error) {
	// Load and decrypt tags if needed
	var tagMap map[int]Tag
	if includeTags {
		var err error
		tagMap, err = loadAndDecryptTags(userID, derivedKey)
		if err != nil {
			utils.Logger.Printf("Warning: Could not load tags for HTML export: %v", err)
			tagMap = make(map[int]Tag) // Use empty map if loading fails
		}
	}

	// Sort entries by date (year, month, day)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Year != entries[j].Year {
			return entries[i].Year < entries[j].Year
		}
		if entries[i].Month != entries[j].Month {
			return entries[i].Month < entries[j].Month
		}
		return entries[i].Day < entries[j].Day
	})

	var html strings.Builder

	// HTML header with embedded CSS
	html.WriteString(`<!DOCTYPE html>
<html lang="de">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>DailyTxT Export</title>
    <link href="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/themes/prism.min.css" rel="stylesheet">
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            line-height: 1.4;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f9f9f9;
            color: #333;
        }
        .header {
            text-align: center;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px;
            border-radius: 10px;
            margin-bottom: 30px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
        }
        .header h1 {
            margin: 0 0 10px 0;
            font-size: 2.5em;
            font-weight: 300;
        }
        .header p {
            margin: 5px 0;
            opacity: 0.9;
        }
        .entry {
            background: white;
            margin-bottom: 25px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        .entry-date {
            background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
            color: white;
            padding: 15px 20px;
            font-size: 1.2em;
            font-weight: 600;
        }
        .entry-content {
            padding: 20px;
        }
        .entry-text {
            margin-bottom: 15px;
            font-size: 1.1em;
            line-height: 1.5;
        }
        .entry-files, .entry-tags, .entry-images {
            margin-top: 15px;
            padding-top: 15px;
            border-top: 1px solid #eee;
        }
        .entry-files h4, .entry-tags h4, .entry-images h4 {
            margin: 0 0 10px 0;
            color: #666;
            font-size: 0.9em;
            text-transform: uppercase;
            letter-spacing: 1px;
        }
        .file-list {
            list-style: none;
            padding: 0;
        }
        .file-list li {
            background: #f8f9fa;
            padding: 8px 12px;
            margin: 5px 0;
            border-radius: 4px;
            border-left: 3px solid #007bff;
        }
        .tags {
            display: flex;
            flex-wrap: wrap;
            gap: 8px;
        }
        .tag {
            background: #e9ecef;
            color: #495057;
            padding: 4px 8px;
            border-radius: 12px;
            font-size: 0.85em;
            font-weight: 500;
        }
        .image-gallery {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
            gap: 10px;
            margin-top: 10px;
        }
        .image-item {
            text-align: center;
        }
        .image-item img {
            max-width: 100%;
            height: auto;
            border-radius: 4px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .image-filename {
            font-size: 0.8em;
            color: #666;
            margin-top: 5px;
        }
        /* Markdown formatting */
        .entry-text h1, .entry-text h2, .entry-text h3, .entry-text h4, .entry-text h5, .entry-text h6 {
            color: #2c3e50;
            margin-top: 20px;
            margin-bottom: 10px;
        }
        .entry-text h1 { font-size: 2em; }
        .entry-text h2 { font-size: 1.5em; }
        .entry-text h3 { font-size: 1.3em; }
        .entry-text strong { font-weight: 600; color: #2c3e50; }
        .entry-text em { font-style: italic; color: #555; }
        .entry-text code {
            background: #f8f9fa;
            border: 1px solid #e9ecef;
            border-radius: 3px;
            padding: 2px 4px;
            font-family: 'Courier New', monospace;
            font-size: 0.9em;
            color: #e83e8c;
        }
        .entry-text pre {
            background: #f8f9fa;
            border: 1px solid #e9ecef;
            border-radius: 6px;
            padding: 15px;
            overflow-x: auto;
            margin: 15px 0;
            font-family: 'Fira Code', 'Courier New', monospace;
            font-size: 0.9em;
            line-height: 1.4;
        }
        .entry-text pre code {
            background: none;
            border: none;
            padding: 0;
            color: inherit;
            font-size: inherit;
        }
        .entry-text p {
            margin: 8px 0;
            line-height: 1.5;
        }
        .entry-text br {
            line-height: 0.8;
        }
        .entry-text ul, .entry-text ol {
            margin: 10px 0;
            padding-left: 20px;
        }
        .entry-text li {
            margin: 4px 0;
            line-height: 1.4;
        }
        .entry-text blockquote {
            border-left: 4px solid #007bff;
            padding-left: 15px;
            margin: 15px 0;
            color: #666;
            font-style: italic;
        }
        .entry-text a {
            color: #007bff;
            text-decoration: none;
        }
        .entry-text a:hover {
            text-decoration: underline;
        }
        /* Custom Prism.js overrides for better code highlighting */
        .token.comment,
        .token.prolog,
        .token.doctype,
        .token.cdata {
            color: #6a737d;
        }
        .token.punctuation {
            color: #586e75;
        }
        .token.property,
        .token.tag,
        .token.constant,
        .token.symbol,
        .token.deleted {
            color: #d73a49;
        }
        .token.boolean,
        .token.number {
            color: #005cc5;
        }
        .token.selector,
        .token.attr-name,
        .token.string,
        .token.char,
        .token.builtin,
        .token.inserted {
            color: #032f62;
        }
        .token.operator,
        .token.entity,
        .token.url,
        .language-css .token.string,
        .style .token.string,
        .token.variable {
            color: #e36209;
        }
        .token.atrule,
        .token.attr-value,
        .token.function,
        .token.class-name {
            color: #6f42c1;
        }
        .token.keyword {
            color: #d73a49;
        }
            border-radius: 5px;
            padding: 15px;
            overflow-x: auto;
            margin: 15px 0;
        }
        .entry-text pre code {
            background: transparent;
            border: none;
            padding: 0;
            font-size: 0.9em;
            color: #333;
            white-space: pre;
        }
        .entry-text a {
            color: #007bff;
            text-decoration: none;
        }
        .entry-text a:hover {
            color: #0056b3;
            text-decoration: underline;
        }
        .entry-text ul, .entry-text ol {
            margin: 10px 0;
            padding-left: 20px;
        }
        .entry-text li {
            margin: 5px 0;
        }
        .entry-text blockquote {
            border-left: 4px solid #007bff;
            margin: 15px 0;
            padding: 10px 15px;
            background: #f8f9fa;
            font-style: italic;
        }
        .entry-text p {
            margin: 10px 0;
        }
        
        /* Lightbox styles */
        .lightbox {
            display: none;
            position: fixed;
            z-index: 1000;
            left: 0;
            top: 0;
            width: 100%;
            height: 100%;
            background-color: rgba(0, 0, 0, 0.9);
            cursor: pointer;
        }
        .lightbox.active {
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .lightbox-content {
            position: relative;
            max-width: 95vw;
            max-height: 95vh;
            cursor: default;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .lightbox-image {
            max-width: 95vw;
            max-height: 95vh;
            width: auto;
            height: auto;
            object-fit: contain;
        }
        .lightbox-nav {
            position: absolute;
            top: 50%;
            transform: translateY(-50%);
            background: rgba(255, 255, 255, 0.8);
            border: none;
            font-size: 24px;
            padding: 15px 20px;
            cursor: pointer;
            border-radius: 5px;
            color: #333;
            font-weight: bold;
        }
        .lightbox-nav:hover {
            background: rgba(255, 255, 255, 1);
        }
        .lightbox-prev {
            left: 20px;
        }
        .lightbox-next {
            right: 20px;
        }
        .lightbox-close {
            position: absolute;
            top: 20px;
            right: 20px;
            background: rgba(255, 255, 255, 0.8);
            border: none;
            font-size: 24px;
            padding: 10px 15px;
            cursor: pointer;
            border-radius: 5px;
            color: #333;
            font-weight: bold;
        }
        .lightbox-close:hover {
            background: rgba(255, 255, 255, 1);
        }
        .image-item img {
            cursor: pointer;
        }
        
        @media print {
            body {
                background: white;
                font-size: 12pt;
            }
            .entry {
                box-shadow: none;
                border: 1px solid #ddd;
                break-inside: avoid;
            }
        }
    </style>
</head>
<body>`)

	// Header
	html.WriteString(`    <div class="header">
        <h1>DailyTxT Export</h1>`)
	html.WriteString(fmt.Sprintf(`        <p>Benutzer ID: %d</p>`, userID))
	html.WriteString(fmt.Sprintf(`        <p>Exportiert am: %s</p>`, time.Now().Format("02.01.2006 15:04:05")))
	html.WriteString(fmt.Sprintf(`        <p>Anzahl Einträge: %d</p>`, len(entries)))
	html.WriteString(`    </div>
`)

	// Process entries
	for _, entry := range entries {
		html.WriteString(`    <div class="entry">
`)

		// Date header with weekday
		date := time.Date(entry.Year, time.Month(entry.Month), entry.Day, 0, 0, 0, 0, time.UTC)
		weekdays := []string{"Sonntag", "Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag"}
		weekday := weekdays[date.Weekday()]
		dateStr := fmt.Sprintf("%s, %02d.%02d.%d", weekday, entry.Day, entry.Month, entry.Year)
		html.WriteString(fmt.Sprintf(`        <div class="entry-date">%s</div>
`, htmlpkg.EscapeString(dateStr)))

		html.WriteString(`        <div class="entry-content">
`)

		// Entry text
		if entry.Text != "" {
			// Decode HTML entities and render markdown
			text := htmlpkg.UnescapeString(entry.Text)
			text = renderMarkdownToHTML(text)
			html.WriteString(fmt.Sprintf(`            <div class="entry-text">%s</div>
`, text))
		}

		// Images (if enabled and images exist)
		if includeImages && len(entry.Files) > 0 {
			imageFiles := make([]string, 0)
			for _, file := range entry.Files {
				// Check if file is an image (simple check by extension)
				ext := strings.ToLower(filepath.Ext(file))
				if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp" {
					imageFiles = append(imageFiles, file)
				}
			}

			if len(imageFiles) > 0 {
				html.WriteString(`            <div class="entry-images">
                <h4>Bilder</h4>
                <div class="image-gallery">
`)
				for _, imageFile := range imageFiles {
					imagePath := fmt.Sprintf("files/%d-%02d-%02d/%s", entry.Year, entry.Month, entry.Day, imageFile)
					html.WriteString(fmt.Sprintf(`                    <div class="image-item">
                        <img src="%s" data-path="%s" alt="%s" loading="lazy" onclick="openLightbox('%s')">
                        <div class="image-filename">%s</div>
                    </div>
`, htmlpkg.EscapeString(imagePath), htmlpkg.EscapeString(imagePath), htmlpkg.EscapeString(imageFile), htmlpkg.EscapeString(imagePath), htmlpkg.EscapeString(imageFile)))
				}
				html.WriteString(`                </div>
            </div>
`)
			}
		}

		// Tags
		if includeTags && len(entry.Tags) > 0 {
			html.WriteString(`            <div class="entry-tags">
                <h4>Tags</h4>
                <div class="tags">
`)
			for _, tagID := range entry.Tags {
				if tag, exists := tagMap[tagID]; exists {
					// Use decrypted tag information
					style := ""
					if tag.Color != "" {
						style = fmt.Sprintf(` style="background-color: %s; color: white;"`, htmlpkg.EscapeString(tag.Color))
					}
					html.WriteString(fmt.Sprintf(`                    <span class="tag"%s>%s #%s</span>
`, style, htmlpkg.EscapeString(tag.Icon), htmlpkg.EscapeString(tag.Name)))
				} else {
					// Fallback to ID if tag not found
					html.WriteString(fmt.Sprintf(`                    <span class="tag">#%d</span>
`, tagID))
				}
			}
			html.WriteString(`                </div>
            </div>
`)
		}

		// Files
		if len(entry.Files) > 0 {
			html.WriteString(`            <div class="entry-files">
                <h4>Dateien</h4>
                <ul class="file-list">
`)
			for _, file := range entry.Files {
				filePath := fmt.Sprintf("files/%d-%02d-%02d/%s", entry.Year, entry.Month, entry.Day, file)
				html.WriteString(fmt.Sprintf(`                    <li><a href="%s" target="_blank">%s</a></li>
`, htmlpkg.EscapeString(filePath), htmlpkg.EscapeString(file)))
			}
			html.WriteString(`                </ul>
            </div>
`)
		}

		html.WriteString(`        </div>
    </div>
`)
	}

	html.WriteString(`
    <!-- Lightbox for images -->
    <div id="lightbox" class="lightbox" onclick="closeLightbox()">
        <div class="lightbox-content" onclick="event.stopPropagation()">
            <button class="lightbox-close" onclick="closeLightbox()">&times;</button>
            <button class="lightbox-nav lightbox-prev" onclick="changeLightboxImage(-1)">&lt;</button>
            <img id="lightbox-image" class="lightbox-image" src="" alt="">
            <button class="lightbox-nav lightbox-next" onclick="changeLightboxImage(1)">&gt;</button>
        </div>
    </div>

    <script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/components/prism-core.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/plugins/autoloader/prism-autoloader.min.js"></script>
    <script>
        // Enable automatic syntax highlighting
        Prism.highlightAll();

        // Lightbox functionality
        let currentImageIndex = 0;
        let allImages = [];

        // Collect all images on page load
        document.addEventListener('DOMContentLoaded', function() {
            const images = document.querySelectorAll('.image-item img');
            allImages = Array.from(images).map(img => img.dataset.path);
        });

        function openLightbox(imageSrc) {
            currentImageIndex = allImages.indexOf(imageSrc);
            if (currentImageIndex === -1) {
                currentImageIndex = 0;
                allImages = [imageSrc];
            }
            
            document.getElementById('lightbox-image').src = imageSrc;
            document.getElementById('lightbox').classList.add('active');
            
            // Hide navigation buttons if only one image
            const prevBtn = document.querySelector('.lightbox-prev');
            const nextBtn = document.querySelector('.lightbox-next');
            if (allImages.length <= 1) {
                prevBtn.style.display = 'none';
                nextBtn.style.display = 'none';
            } else {
                prevBtn.style.display = 'block';
                nextBtn.style.display = 'block';
            }
        }

        function closeLightbox() {
            document.getElementById('lightbox').classList.remove('active');
        }

        function changeLightboxImage(direction) {
            if (allImages.length <= 1) return;
            
            currentImageIndex += direction;
            if (currentImageIndex >= allImages.length) {
                currentImageIndex = 0;
            } else if (currentImageIndex < 0) {
                currentImageIndex = allImages.length - 1;
            }
            
            document.getElementById('lightbox-image').src = allImages[currentImageIndex];
        }

        // Keyboard navigation
        document.addEventListener('keydown', function(e) {
            if (document.getElementById('lightbox').classList.contains('active')) {
                if (e.key === 'Escape') {
                    closeLightbox();
                } else if (e.key === 'ArrowLeft') {
                    changeLightboxImage(-1);
                } else if (e.key === 'ArrowRight') {
                    changeLightboxImage(1);
                }
            }
        });
    </script>
</body>
</html>`)

	return []byte(html.String()), nil
}

// Tag represents a decrypted tag
type Tag struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Color string `json:"color"`
}

// loadAndDecryptTags loads and decrypts user tags
func loadAndDecryptTags(userID int, derivedKey string) (map[int]Tag, error) {
	content, err := utils.GetTags(userID)
	if err != nil {
		return nil, err
	}

	// Get encryption key
	encKey, err := utils.GetEncryptionKey(userID, derivedKey)
	if err != nil {
		return nil, err
	}

	tagMap := make(map[int]Tag)

	if tags, ok := content["tags"].([]any); ok {
		for _, tagInterface := range tags {
			if tagData, ok := tagInterface.(map[string]any); ok {
				if tagID, ok := tagData["id"].(float64); ok {
					tag := Tag{ID: int(tagID)}

					// Decrypt name
					if encName, ok := tagData["name"].(string); ok {
						if decryptedName, err := utils.DecryptText(encName, encKey); err == nil {
							tag.Name = decryptedName
						}
					}

					// Decrypt icon
					if encIcon, ok := tagData["icon"].(string); ok {
						if decryptedIcon, err := utils.DecryptText(encIcon, encKey); err == nil {
							tag.Icon = decryptedIcon
						}
					}

					// Decrypt color
					if encColor, ok := tagData["color"].(string); ok {
						if decryptedColor, err := utils.DecryptText(encColor, encKey); err == nil {
							tag.Color = decryptedColor
						}
					}

					tagMap[int(tagID)] = tag
				}
			}
		}
	}

	return tagMap, nil
}

// generateUniqueFilename creates a unique filename by appending (2), (3), etc. if conflicts exist
func generateUniqueFilename(usedFilenames map[string]bool, originalFilename string) string {
	if !usedFilenames[originalFilename] {
		usedFilenames[originalFilename] = true
		return originalFilename
	}

	// Extract file extension
	ext := filepath.Ext(originalFilename)
	nameWithoutExt := strings.TrimSuffix(originalFilename, ext)

	// Try appending (2), (3), etc.
	counter := 2
	for {
		newFilename := fmt.Sprintf("%s (%d)%s", nameWithoutExt, counter, ext)
		if !usedFilenames[newFilename] {
			usedFilenames[newFilename] = true
			return newFilename
		}
		counter++
	}
}

// renderMarkdownToHTML converts markdown to HTML using gomarkdown library
func renderMarkdownToHTML(text string) string {
	// Create parser with extensions including hard line breaks
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock | parser.HardLineBreak
	p := parser.NewWithExtensions(extensions)

	// Create HTML renderer with options for Prism.js compatibility
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{
		Flags: htmlFlags,
		RenderNodeHook: func(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
			if codeBlock, ok := node.(*ast.CodeBlock); ok && entering {
				lang := string(codeBlock.Info)
				if lang == "" {
					lang = "none"
				}

				// Write opening tag with Prism.js class
				fmt.Fprintf(w, `<pre><code class="language-%s">`, lang)

				// Write escaped content
				content := string(codeBlock.Literal)
				html.EscapeHTML(w, []byte(content))

				// Write closing tag
				w.Write([]byte("</code></pre>"))
				return ast.GoToNext, true
			}
			return ast.GoToNext, false
		},
	}
	renderer := html.NewRenderer(opts)

	// Parse and render
	md := []byte(text)
	htmlBytes := markdown.ToHTML(md, p, renderer)

	return string(htmlBytes)
}

/*
// BACKUP: Old custom markdown renderer - kept for reference
// renderMarkdownToHTML converts simple markdown to HTML
func renderMarkdownToHTML_OLD(text string) string {
	lines := strings.Split(text, "\n")
	var result strings.Builder
	var inCodeBlock bool
	var codeBlockContent strings.Builder
	var codeLanguage string

	for _, line := range lines {
		// Handle code blocks
		if strings.HasPrefix(line, "```") {
			if !inCodeBlock {
				// Starting a code block
				inCodeBlock = true
				codeLanguage = strings.TrimSpace(line[3:])
				codeBlockContent.Reset()
				continue
			} else {
				// Ending a code block
				inCodeBlock = false
				codeContent := codeBlockContent.String()
				if codeLanguage != "" {
					result.WriteString(fmt.Sprintf(`<pre><code class="language-%s">%s</code></pre>`,
						htmlpkg.EscapeString(codeLanguage), htmlpkg.EscapeString(codeContent)))
				} else {
					result.WriteString(fmt.Sprintf(`<pre><code>%s</code></pre>`,
						htmlpkg.EscapeString(codeContent)))
				}
				codeLanguage = ""
				continue
			}
		}

		// If we're in a code block, accumulate the content
		if inCodeBlock {
			if codeBlockContent.Len() > 0 {
				codeBlockContent.WriteString("\n")
			}
			codeBlockContent.WriteString(line)
			continue
		}

		// Normal line processing (when not in code block)
		line = strings.TrimSpace(line)

		if line == "" {
			result.WriteString("<br>")
			continue
		}

		// Handle headings
		if strings.HasPrefix(line, "###") {
			content := strings.TrimSpace(line[3:])
			result.WriteString(fmt.Sprintf("<h3>%s</h3>", htmlpkg.EscapeString(content)))
			continue
		}
		if strings.HasPrefix(line, "##") {
			content := strings.TrimSpace(line[2:])
			result.WriteString(fmt.Sprintf("<h2>%s</h2>", htmlpkg.EscapeString(content)))
			continue
		}
		if strings.HasPrefix(line, "#") {
			content := strings.TrimSpace(line[1:])
			result.WriteString(fmt.Sprintf("<h1>%s</h1>", htmlpkg.EscapeString(content)))
			continue
		}

		// Handle list items
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			content := strings.TrimSpace(line[2:])
			content = renderInlineMarkdownToHTML(content)
			result.WriteString(fmt.Sprintf("<ul><li>%s</li></ul>", content))
			continue
		}

		// Handle blockquotes
		if strings.HasPrefix(line, "> ") {
			content := strings.TrimSpace(line[2:])
			content = renderInlineMarkdownToHTML(content)
			result.WriteString(fmt.Sprintf("<blockquote>%s</blockquote>", content))
			continue
		}

		// Regular paragraph with inline formatting
		line = renderInlineMarkdownToHTML(line)
		result.WriteString(fmt.Sprintf("<p>%s</p>", line))
	}

	// Handle unclosed code block
	if inCodeBlock {
		codeContent := codeBlockContent.String()
		if codeLanguage != "" {
			result.WriteString(fmt.Sprintf(`<pre><code class="language-%s">%s</code></pre>`,
				htmlpkg.EscapeString(codeLanguage), htmlpkg.EscapeString(codeContent)))
		} else {
			result.WriteString(fmt.Sprintf(`<pre><code>%s</code></pre>`,
				htmlpkg.EscapeString(codeContent)))
		}
	}

	return result.String()
}

// renderInlineMarkdownToHTML processes inline markdown formatting
func renderInlineMarkdownToHTML_OLD(text string) string {
	// Escape HTML first
	text = htmlpkg.EscapeString(text)

	// Links with title: [text](url "title")
	linkWithTitleRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\s+"([^"]+)"\)`)
	text = linkWithTitleRegex.ReplaceAllStringFunc(text, func(match string) string {
		matches := linkWithTitleRegex.FindStringSubmatch(match)
		if len(matches) == 4 {
			linkText := matches[1]
			url := matches[2]
			title := matches[3]
			return fmt.Sprintf(`<a href="%s" title="%s" target="_blank" rel="noopener noreferrer">%s</a>`, url, title, linkText)
		}
		return match
	})

	// Links without title: [text](url)
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	text = linkRegex.ReplaceAllStringFunc(text, func(match string) string {
		matches := linkRegex.FindStringSubmatch(match)
		if len(matches) == 3 {
			linkText := matches[1]
			url := matches[2]
			return fmt.Sprintf(`<a href="%s" target="_blank" rel="noopener noreferrer">%s</a>`, url, linkText)
		}
		return match
	})

	// Bold (**text** or __text__)
	boldRegex := regexp.MustCompile(`\*\*(.*?)\*\*`)
	text = boldRegex.ReplaceAllString(text, "<strong>$1</strong>")

	boldRegex2 := regexp.MustCompile(`__(.*?)__`)
	text = boldRegex2.ReplaceAllString(text, "<strong>$1</strong>")

	// Italic (*text* or _text_)
	italicRegex := regexp.MustCompile(`\*(.*?)\*`)
	text = italicRegex.ReplaceAllString(text, "<em>$1</em>")

	italicRegex2 := regexp.MustCompile(`_(.*?)_`)
	text = italicRegex2.ReplaceAllString(text, "<em>$1</em>")

	// Code (`text`)
	codeRegex := regexp.MustCompile("`(.*?)`")
	text = codeRegex.ReplaceAllString(text, "<code>$1</code>")

	return text
}
*/
