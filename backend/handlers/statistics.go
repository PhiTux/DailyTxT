package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/phitux/dailytxt/backend/utils"
)

// Load user statistics:
// - each logged day with amount of words for each day
// - amount of files for each day
// - tags for each day
func GetStatistics(w http.ResponseWriter, r *http.Request) {
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

	// Prepare encryption key for decrypting texts and filenames
	encKey, err := utils.GetEncryptionKey(userID, derivedKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting encryption key: %v", err), http.StatusInternalServerError)
		return
	}

	// Define response structure (per day only)
	type DayStat struct {
		Year          int   `json:"year"`
		Month         int   `json:"month"`
		Day           int   `json:"day"`
		WordCount     int   `json:"wordCount"`
		FileCount     int   `json:"fileCount"`
		FileSizeBytes int64 `json:"fileSizeBytes"`
		Tags          []int `json:"tags"`
		IsBookmarked  bool  `json:"isBookmarked"`
	}

	dayStats := []DayStat{}

	// Get all years
	years, err := utils.GetYears(userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving years: %v", err), http.StatusInternalServerError)
		return
	}

	// Iterate years and months
	for _, yearStr := range years {
		yearInt, _ := strconv.Atoi(yearStr)
		months, err := utils.GetMonths(userID, yearStr)
		if err != nil {
			continue // skip problematic year
		}
		for _, monthStr := range months {
			monthInt, _ := strconv.Atoi(monthStr)
			content, err := utils.GetMonth(userID, yearInt, monthInt)
			if err != nil {
				continue
			}
			daysArr, ok := content["days"].([]any)
			if !ok {
				continue
			}
			for _, dayInterface := range daysArr {
				dayMap, ok := dayInterface.(map[string]any)
				if !ok {
					continue
				}
				dayNumFloat, ok := dayMap["day"].(float64)
				if !ok {
					continue
				}
				dayNum := int(dayNumFloat)

				// Word count (decrypt text if present)
				wordCount := 0
				if encText, ok := dayMap["text"].(string); ok && encText != "" {
					if decrypted, err := utils.DecryptText(encText, encKey); err == nil {
						wordCount = CountWords(decrypted)
					}
				}

				// File count and total size
				fileCount := 0
				var totalFileSize int64 = 0
				if filesAny, ok := dayMap["files"].([]any); ok {
					fileCount = len(filesAny)
					// Calculate total file size for this day
					for _, fileInterface := range filesAny {
						if fileMap, ok := fileInterface.(map[string]any); ok {
							if sizeAny, ok := fileMap["size"]; ok {
								// Handle both int64 and float64 types
								switch size := sizeAny.(type) {
								case int64:
									totalFileSize += size
								case float64:
									totalFileSize += int64(size)
								case int:
									totalFileSize += int64(size)
								}
							}
						}
					}
				}

				// Tags (IDs are numeric)
				var tagIDs []int
				if tagsAny, ok := dayMap["tags"].([]any); ok {
					for _, t := range tagsAny {
						if tf, ok := t.(float64); ok {
							tagIDs = append(tagIDs, int(tf))
						}
					}
				}

				// Bookmark flag
				isBookmarked := false
				if bmRaw, ok := dayMap["isBookmarked"]; ok {
					if b, ok2 := bmRaw.(bool); ok2 {
						isBookmarked = b
					} else if f, ok2 := bmRaw.(float64); ok2 { // if stored as number
						isBookmarked = f != 0
					}
				}

				dayStats = append(dayStats, DayStat{
					Year:          yearInt,
					Month:         monthInt,
					Day:           dayNum,
					WordCount:     wordCount,
					FileCount:     fileCount,
					FileSizeBytes: totalFileSize,
					Tags:          tagIDs,
					IsBookmarked:  isBookmarked,
				})
			}
		}
	}

	// Sort days by date descending (latest first) if desired; currently ascending by traversal. Keep ascending.

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dayStats)
}

func CountWords(text string) int {
	count := 0
	i := 0
	textLength := len(text)

	for i < textLength {
		r, size := utf8.DecodeRuneInString(text[i:])
		if r == utf8.RuneError {
			i++
			continue
		}

		// Count CJK characters individually
		if IsCJK(r) {
			count++
			i += size
			continue
		}

		// Count other sequences as words
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			// Start of a new word
			count++
			i += size

			// Attempt to consume the rest of the same word
			for i < textLength {
				nextR, nextSize := utf8.DecodeRuneInString(text[i:])

				// Stop merging immediately if a CJK character is encountered
				if IsCJK(nextR) {
					break
				}

				// Merge if the next character is a standard letter or number
				if unicode.IsLetter(nextR) || unicode.IsNumber(nextR) {
					i += nextSize
					continue
				}

				// Handle hyphens (-) and apostrophes (')
				if nextR == '-' || nextR == '\'' {
					lookAheadIdx := i + nextSize
					if lookAheadIdx < textLength {
						followR, _ := utf8.DecodeRuneInString(text[lookAheadIdx:])
						// Merge if the character following the punctuation is valid
						// (e.g., "state-of-the-art", "don't").
						if unicode.IsLetter(followR) || unicode.IsNumber(followR) {
							i += nextSize
							continue
						}
					}
				}

				// End of the word for any other character
				break
			}
			continue
		}

		// Skip all other characters (spaces, punctuation, emojis, etc.)
		i += size
	}

	return count
}

func IsCJK(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
		(r >= 0x3400 && r <= 0x4DBF) || // CJK Extension A
		(r >= 0xF900 && r <= 0xFAFF) || // CJK Compatibility
		(r >= 0x3040 && r <= 0x309F) || // Hiragana
		(r >= 0x30A0 && r <= 0x30FF) || // Katakana
		(r >= 0xAC00 && r <= 0xD7A3) // Hangul Syllables
}
