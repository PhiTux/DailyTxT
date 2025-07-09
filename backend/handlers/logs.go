package handlers

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/phitux/dailytxt/backend/utils"
)

// LogRequest represents the log request body
type LogRequest struct {
	Day         int    `json:"day"`
	Month       int    `json:"month"`
	Year        int    `json:"year"`
	Text        string `json:"text"`
	DateWritten string `json:"date_written"`
}

// SaveLog handles saving a log entry
func SaveLog(w http.ResponseWriter, r *http.Request) {
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
	var req LogRequest
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

	// Check if there's a previous log to move to history
	historyAvailable := false
	days, ok := content["days"].([]any)
	if ok {
		for i, dayInterface := range days {
			day, ok := dayInterface.(map[string]any)
			if !ok {
				continue
			}

			dayNum, ok := day["day"].(float64)
			if !ok || int(dayNum) != req.Day {
				continue
			}

			// If this day has text, move it to history
			if text, ok := day["text"].(string); ok && text != "" {
				historyAvailable = true
				historyVersion := 0

				// Get or create history array
				var history []any
				if historyArray, ok := day["history"].([]any); ok {
					history = historyArray
					// Find highest version
					for _, historyItem := range history {
						if historyMap, ok := historyItem.(map[string]any); ok {
							if version, ok := historyMap["version"].(float64); ok && int(version) > historyVersion {
								historyVersion = int(version)
							}
						}
					}
				} else {
					history = []any{}
				}

				historyVersion++
				history = append(history, map[string]any{
					"version":      historyVersion,
					"text":         day["text"],
					"date_written": day["date_written"],
				})

				day["history"] = history
				days[i] = day
			}
			break
		}
		content["days"] = days
	}

	// Get encryption key
	encKey, err := utils.GetEncryptionKey(userID, derivedKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting encryption key: %v", err), http.StatusInternalServerError)
		return
	}

	// Encrypt text and date_written
	encryptedText, err := utils.EncryptText(req.Text, encKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error encrypting text: %v", err), http.StatusInternalServerError)
		return
	}

	encryptedDateWritten, err := utils.EncryptText(html.EscapeString(req.DateWritten), encKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error encrypting date_written: %v", err), http.StatusInternalServerError)
		return
	}

	// Save new log
	found := false
	if days, ok := content["days"].([]any); ok {
		for i, dayInterface := range days {
			day, ok := dayInterface.(map[string]any)
			if !ok {
				continue
			}

			dayNum, ok := day["day"].(float64)
			if !ok || int(dayNum) != req.Day {
				continue
			}

			// Update existing day
			day["text"] = encryptedText
			day["date_written"] = encryptedDateWritten
			days[i] = day
			found = true
			break
		}

		if !found {
			// Add new day
			days = append(days, map[string]any{
				"day":          req.Day,
				"text":         encryptedText,
				"date_written": encryptedDateWritten,
			})
		}

		content["days"] = days
	} else {
		// Create new days array
		content["days"] = []any{
			map[string]any{
				"day":          req.Day,
				"text":         encryptedText,
				"date_written": encryptedDateWritten,
			},
		}
	}

	// Write month data
	if err := utils.WriteMonth(userID, req.Year, req.Month, content); err != nil {
		http.Error(w, fmt.Sprintf("Error writing month data: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success
	utils.JSONResponse(w, http.StatusOK, map[string]any{
		"success":           true,
		"history_available": historyAvailable,
	})
}

// GetLog handles retrieving a log entry
func GetLog(w http.ResponseWriter, r *http.Request) {
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
	year, err := strconv.Atoi(r.URL.Query().Get("year"))
	if err != nil {
		http.Error(w, "Invalid year parameter", http.StatusBadRequest)
		return
	}
	month, err := strconv.Atoi(r.URL.Query().Get("month"))
	if err != nil {
		http.Error(w, "Invalid month parameter", http.StatusBadRequest)
		return
	}
	dayValue, err := strconv.Atoi(r.URL.Query().Get("day"))
	if err != nil {
		http.Error(w, "Invalid day parameter", http.StatusBadRequest)
		return
	}

	// Get month data
	content, err := utils.GetMonth(userID, year, month)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving month data: %v", err), http.StatusInternalServerError)
		return
	}

	// Default empty response
	dummy := map[string]any{
		"text":         "",
		"date_written": "",
		"files":        []any{},
		"tags":         []any{},
	}

	// Check if days exist
	days, ok := content["days"].([]any)
	if !ok {
		utils.JSONResponse(w, http.StatusOK, dummy)
		return
	}

	// Find the day
	for _, dayInterface := range days {
		day, ok := dayInterface.(map[string]any)
		if !ok {
			continue
		}

		dayNum, ok := day["day"].(float64)
		if !ok || int(dayNum) != dayValue {
			continue
		}

		// Get encryption key
		encKey, err := utils.GetEncryptionKey(userID, derivedKey)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error getting encryption key: %v", err), http.StatusInternalServerError)
			return
		}

		// Decrypt text and date_written
		text := ""
		dateWritten := ""
		historyAvailable := false

		if encryptedText, ok := day["text"].(string); ok && encryptedText != "" {
			decryptedText, err := utils.DecryptText(encryptedText, encKey)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error decrypting text: %v", err), http.StatusInternalServerError)
				return
			}
			text = decryptedText
		}

		if encryptedDate, ok := day["date_written"].(string); ok && encryptedDate != "" {
			decryptedDate, err := utils.DecryptText(encryptedDate, encKey)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error decrypting date_written: %v", err), http.StatusInternalServerError)
				return
			}
			dateWritten = decryptedDate
		}

		// Check for history
		if history, ok := day["history"].([]any); ok && len(history) > 0 {
			historyAvailable = true
		}

		// Decrypt filenames if files exist
		files := []any{}
		if filesList, ok := day["files"].([]any); ok {
			for _, fileInterface := range filesList {
				file, ok := fileInterface.(map[string]any)
				if !ok {
					continue
				}

				if encFilename, ok := file["enc_filename"].(string); ok {
					decryptedFilename, err := utils.DecryptText(encFilename, encKey)
					if err != nil {
						http.Error(w, fmt.Sprintf("Error decrypting filename: %v", err), http.StatusInternalServerError)
						return
					}
					fileCopy := make(map[string]any)
					for k, v := range file {
						fileCopy[k] = v
					}
					fileCopy["filename"] = decryptedFilename
					files = append(files, fileCopy)
				}
			}
		}

		// Get tags
		tags := []any{}
		if tagsList, ok := day["tags"].([]any); ok {
			tags = tagsList
		}

		// Return log data
		utils.JSONResponse(w, http.StatusOK, map[string]any{
			"text":              text,
			"date_written":      dateWritten,
			"files":             files,
			"tags":              tags,
			"history_available": historyAvailable,
		})
		return
	}

	// If day not found, return empty response
	utils.JSONResponse(w, http.StatusOK, dummy)
}

// GetMarkedDays handles retrieving a month's logs
func GetMarkedDays(w http.ResponseWriter, r *http.Request) {
	// Get user ID and derived key from context
	userID, ok := r.Context().Value(utils.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get parameters from URL
	year, err := strconv.Atoi(r.URL.Query().Get("year"))
	if err != nil {
		http.Error(w, "Invalid year parameter", http.StatusBadRequest)
		return
	}
	month, err := strconv.Atoi(r.URL.Query().Get("month"))
	if err != nil {
		http.Error(w, "Invalid month parameter", http.StatusBadRequest)
		return
	}

	// Get month data
	content, err := utils.GetMonth(userID, year, month)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving month data: %v", err), http.StatusInternalServerError)
		return
	}

	// Extract days with logs, files, and bookmarks
	daysWithLogs := []int{}
	daysWithFiles := []int{}
	daysBookmarked := []int{}

	if days, ok := content["days"].([]any); ok {
		for _, dayInterface := range days {
			day, ok := dayInterface.(map[string]any)
			if !ok {
				continue
			}

			dayNum, ok := day["day"].(float64)
			if !ok {
				continue
			}

			// Check for text
			if _, ok := day["text"].(string); ok {
				daysWithLogs = append(daysWithLogs, int(dayNum))
			}

			// Check for files
			if files, ok := day["files"].([]any); ok && len(files) > 0 {
				daysWithFiles = append(daysWithFiles, int(dayNum))
			}

			// Check if bookmarked
			if bookmarked, ok := day["bookmarked"].(bool); ok && bookmarked {
				daysBookmarked = append(daysBookmarked, int(dayNum))
			}
		}
	}

	// Return month data
	utils.JSONResponse(w, http.StatusOK, map[string]any{
		"days_with_logs":  daysWithLogs,
		"days_with_files": daysWithFiles,
		"days_bookmarked": daysBookmarked,
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

// TemplatesRequest represents a templates request
type TemplatesRequest struct {
	Templates []struct {
		Name string `json:"name"`
		Text string `json:"text"`
	} `json:"templates"`
}

// GetTemplates handles retrieving a user's templates
func GetTemplates(w http.ResponseWriter, r *http.Request) {
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

	// Get templates
	content, err := utils.GetTemplates(userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving templates: %v", err), http.StatusInternalServerError)
		return
	}

	// If no templates, return empty array
	if templates, ok := content["templates"].([]any); !ok || len(templates) == 0 {
		utils.JSONResponse(w, http.StatusOK, []any{})
		return
	}

	// Get encryption key
	encKey, err := utils.GetEncryptionKey(userID, derivedKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting encryption key: %v", err), http.StatusInternalServerError)
		return
	}

	// Decrypt template data
	templates := content["templates"].([]any)
	result := []any{}

	for _, templateInterface := range templates {
		template, ok := templateInterface.(map[string]any)
		if !ok {
			continue
		}

		// Decrypt name and text
		if encName, ok := template["name"].(string); ok {
			decryptedName, err := utils.DecryptText(encName, encKey)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error decrypting template name: %v", err), http.StatusInternalServerError)
				return
			}
			template["name"] = decryptedName
		}

		if encText, ok := template["text"].(string); ok {
			decryptedText, err := utils.DecryptText(encText, encKey)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error decrypting template text: %v", err), http.StatusInternalServerError)
				return
			}
			template["text"] = decryptedText
		}

		result = append(result, template)
	}

	// Return templates
	utils.JSONResponse(w, http.StatusOK, result)
}

// SaveTemplates handles saving templates
func SaveTemplates(w http.ResponseWriter, r *http.Request) {
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
	var req TemplatesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get encryption key
	encKey, err := utils.GetEncryptionKey(userID, derivedKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting encryption key: %v", err), http.StatusInternalServerError)
		return
	}

	// Create new templates content
	content := map[string]any{
		"templates": []any{},
	}

	// Encrypt template data
	templates := []any{}
	for _, template := range req.Templates {
		encName, err := utils.EncryptText(template.Name, encKey)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error encrypting template name: %v", err), http.StatusInternalServerError)
			return
		}

		encText, err := utils.EncryptText(template.Text, encKey)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error encrypting template text: %v", err), http.StatusInternalServerError)
			return
		}

		templates = append(templates, map[string]any{
			"name": encName,
			"text": encText,
		})
	}

	content["templates"] = templates

	// Write templates
	if err := utils.WriteTemplates(userID, content); err != nil {
		http.Error(w, fmt.Sprintf("Error writing templates: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success
	utils.JSONResponse(w, http.StatusOK, map[string]bool{
		"success": true,
	})
}

// GetOnThisDay handles retrieving logs from previous years on the same day
func GetOnThisDay(w http.ResponseWriter, r *http.Request) {
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
	month, err := strconv.Atoi(r.PathValue("month"))
	if err != nil {
		http.Error(w, "Invalid month parameter", http.StatusBadRequest)
		return
	}
	day, err := strconv.Atoi(r.PathValue("day"))
	if err != nil {
		http.Error(w, "Invalid day parameter", http.StatusBadRequest)
		return
	}

	// Get query parameters
	lastYears := r.URL.Query().Get("last_years")
	if lastYears == "" {
		http.Error(w, "Missing last_years parameter", http.StatusBadRequest)
		return
	}

	// Parse years
	yearStr := strings.Split(lastYears, ",")
	years := []int{}
	currentYear, err := strconv.Atoi(r.URL.Query().Get("year"))
	if err != nil {
		http.Error(w, "Invalid year parameter", http.StatusBadRequest)
		return
	}

	for _, y := range yearStr {
		if val, err := strconv.Atoi(y); err == nil {
			years = append(years, currentYear-val)
		}
	}

	// Get encryption key
	encKey, err := utils.GetEncryptionKey(userID, derivedKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting encryption key: %v", err), http.StatusInternalServerError)
		return
	}

	// Get logs from previous years
	results := []any{}
	for _, year := range years {
		content, err := utils.GetMonth(userID, year, month)
		if err != nil {
			continue
		}

		days, ok := content["days"].([]any)
		if !ok {
			continue
		}

		for _, dayInterface := range days {
			dayLog, ok := dayInterface.(map[string]any)
			if !ok {
				continue
			}

			dayNum, ok := dayLog["day"].(float64)
			if !ok || int(dayNum) != day {
				continue
			}

			text, ok := dayLog["text"].(string)
			if !ok || text == "" {
				continue
			}

			// Decrypt text
			decryptedText, err := utils.DecryptText(text, encKey)
			if err != nil {
				continue
			}

			results = append(results, map[string]any{
				"years_old": currentYear - year,
				"day":       day,
				"month":     month,
				"year":      year,
				"text":      decryptedText,
			})
			break
		}
	}

	// Return results
	utils.JSONResponse(w, http.StatusOK, results)
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
	searchString := r.URL.Query().Get("q")
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
