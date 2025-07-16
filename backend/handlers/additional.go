package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

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

	// Read file
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading file: %v", err), http.StatusInternalServerError)
		return
	}

	// Get encryption key
	encKey, err := utils.GetEncryptionKey(userID, derivedKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting encryption key: %v", err), http.StatusInternalServerError)
		return
	}

	// Encrypt file
	encryptedFile, err := utils.EncryptFile(fileBytes, encKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error encrypting file: %v", err), http.StatusInternalServerError)
		return
	}

	// Write file
	if err := utils.WriteFile(encryptedFile, userID, uuid); err != nil {
		http.Error(w, fmt.Sprintf("Error writing file: %v", err), http.StatusInternalServerError)
		return
	}

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

	// Decrypt file
	decryptedFile, err := utils.DecryptFile(encryptedFile, encKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error decrypting file: %v", err), http.StatusInternalServerError)
		return
	}

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
