package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

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

	// Check for duplicate tag names
	tags, ok := content["tags"].([]any)
	if ok {
		for _, tagInterface := range tags {
			tag, ok := tagInterface.(map[string]any)
			if !ok {
				continue
			}

			if encName, ok := tag["name"].(string); ok {
				decryptedName, err := utils.DecryptText(encName, encKey)
				if err != nil {
					http.Error(w, fmt.Sprintf("Error decrypting tag name: %v", err), http.StatusInternalServerError)
					return
				}
				if decryptedName == req.Name {
					http.Error(w, "Tag name already exists", http.StatusBadRequest)
					return
				}
			}
		}
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
	tags, ok = content["tags"].([]any)
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
