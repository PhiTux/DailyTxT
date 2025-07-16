package handlers

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
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
			if bookmarked, ok := day["isBookmarked"].(bool); ok && bookmarked {
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
	month, err := strconv.Atoi(r.URL.Query().Get("month"))
	if err != nil {
		http.Error(w, "Invalid month parameter", http.StatusBadRequest)
		return
	}
	day, err := strconv.Atoi(r.URL.Query().Get("day"))
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

// LoadMonthForReading handles loading a month for reading
func LoadMonthForReading(w http.ResponseWriter, r *http.Request) {
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

	// Get encryption key
	encKey, err := utils.GetEncryptionKey(userID, derivedKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting encryption key: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if days exist
	days, ok := content["days"].([]any)
	if !ok {
		utils.JSONResponse(w, http.StatusOK, []any{})
		return
	}

	// Process days
	result := []any{}
	for _, dayInterface := range days {
		day, ok := dayInterface.(map[string]any)
		if !ok {
			continue
		}

		dayNum, ok := day["day"].(float64)
		if !ok {
			continue
		}

		// Create result day
		resultDay := map[string]any{
			"day": int(dayNum),
		}

		// Decrypt text and date_written
		if text, ok := day["text"].(string); ok && text != "" {
			decryptedText, err := utils.DecryptText(text, encKey)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error decrypting text: %v", err), http.StatusInternalServerError)
				return
			}
			resultDay["text"] = decryptedText

			if dateWritten, ok := day["date_written"].(string); ok && dateWritten != "" {
				decryptedDate, err := utils.DecryptText(dateWritten, encKey)
				if err != nil {
					http.Error(w, fmt.Sprintf("Error decrypting date_written: %v", err), http.StatusInternalServerError)
					return
				}
				resultDay["date_written"] = decryptedDate
			}
		}

		// Get tags
		if tags, ok := day["tags"].([]any); ok && len(tags) > 0 {
			resultDay["tags"] = tags
		}

		// Decrypt filenames if files exist
		if filesList, ok := day["files"].([]any); ok && len(filesList) > 0 {
			files := []any{}
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
			resultDay["files"] = files
		}

		// Add day to result if it has content
		if _, hasText := resultDay["text"]; hasText {
			result = append(result, resultDay)
		} else if _, hasFiles := resultDay["files"]; hasFiles {
			result = append(result, resultDay)
		} else if _, hasTags := resultDay["tags"]; hasTags {
			result = append(result, resultDay)
		}
	}

	// Return result
	utils.JSONResponse(w, http.StatusOK, result)
}

// GetHistory handles retrieving log history
func GetHistory(w http.ResponseWriter, r *http.Request) {
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

	// Get encryption key
	encKey, err := utils.GetEncryptionKey(userID, derivedKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting encryption key: %v", err), http.StatusInternalServerError)
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
		utils.JSONResponse(w, http.StatusOK, []any{})
		return
	}

	// Find day
	for _, dayInterface := range days {
		dayObj, ok := dayInterface.(map[string]any)
		if !ok {
			continue
		}

		dayNum, ok := dayObj["day"].(float64)
		if !ok || int(dayNum) != day {
			continue
		}

		// Check for history
		history, ok := dayObj["history"].([]any)
		if !ok || len(history) == 0 {
			utils.JSONResponse(w, http.StatusOK, []any{})
			return
		}

		// Decrypt history entries
		result := []any{}
		for _, historyInterface := range history {
			historyEntry, ok := historyInterface.(map[string]any)
			if !ok {
				continue
			}

			text, ok := historyEntry["text"].(string)
			if !ok {
				continue
			}

			dateWritten, ok := historyEntry["date_written"].(string)
			if !ok {
				continue
			}

			// Decrypt text and date
			decryptedText, err := utils.DecryptText(text, encKey)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error decrypting history text: %v", err), http.StatusInternalServerError)
				return
			}

			decryptedDate, err := utils.DecryptText(dateWritten, encKey)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error decrypting history date: %v", err), http.StatusInternalServerError)
				return
			}

			result = append(result, map[string]any{
				"text":         decryptedText,
				"date_written": decryptedDate,
			})
		}

		// Return history
		utils.JSONResponse(w, http.StatusOK, result)
		return
	}

	// Day not found
	utils.JSONResponse(w, http.StatusOK, []any{})
}
