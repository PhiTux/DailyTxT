package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/phitux/dailytxt/backend/utils"
)

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
		return "<em>DailyTxT: Error formatting...</em>"
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
		http.Error(w, "No logs found to be searched", http.StatusNotFound)
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
						words := strings.SplitSeq(searchString, "|")
						for word := range words {
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
