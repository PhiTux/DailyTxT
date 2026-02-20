package handlers

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"

	"github.com/phitux/dailytxt/backend/utils"
)

// validateShareToken decodes and validates a share token from the request query parameter.
// Returns (userID, derivedKey, error).
func validateShareToken(r *http.Request) (int, string, error) {
	token := r.URL.Query().Get("token")
	if token == "" {
		return 0, "", fmt.Errorf("missing token parameter")
	}

	// Decode the token bytes from base64 URL encoding
	tokenBytes, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return 0, "", fmt.Errorf("invalid token format")
	}

	// Compute SHA-256 hash of the raw token bytes for lookup
	hash := sha256.Sum256(tokenBytes)
	tokenHash := base64.URLEncoding.EncodeToString(hash[:])

	// Look up user by token hash
	userID, encDerivedKey, err := utils.GetUserByShareTokenHash(tokenHash)
	if err != nil {
		return 0, "", fmt.Errorf("invalid share token")
	}

	// Decrypt the derived key using the full token as the encryption key
	derivedKey, err := utils.DecryptText(encDerivedKey, token)
	if err != nil {
		return 0, "", fmt.Errorf("error decrypting derived key")
	}

	return userID, derivedKey, nil
}

// GenerateShareToken creates a new share token for the authenticated user.
func GenerateShareToken(w http.ResponseWriter, r *http.Request) {
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

	// Generate a new random token (32 bytes, base64 URL-encoded)
	token := utils.GenerateSecretToken()

	// Compute SHA-256 hash of the raw token bytes for storage
	tokenBytes, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}
	hash := sha256.Sum256(tokenBytes)
	tokenHash := base64.URLEncoding.EncodeToString(hash[:])

	// Encrypt the user's derived key using the share token as the encryption key
	encDerivedKey, err := utils.EncryptText(derivedKey, token)
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	// Persist the token hash and encrypted derived key
	if err := utils.SaveShareToken(userID, tokenHash, encDerivedKey); err != nil {
		http.Error(w, fmt.Sprintf("Error saving share token: %v", err), http.StatusInternalServerError)
		return
	}

	utils.JSONResponse(w, http.StatusOK, map[string]any{
		"success": true,
		"token":   token,
	})
}

// RevokeShareToken removes the share token for the authenticated user.
func RevokeShareToken(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(utils.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := utils.DeleteShareToken(userID); err != nil {
		http.Error(w, fmt.Sprintf("Error revoking share token: %v", err), http.StatusInternalServerError)
		return
	}

	utils.JSONResponse(w, http.StatusOK, map[string]any{
		"success": true,
	})
}

// GetShareTokenInfo returns whether the authenticated user has an active share token.
func GetShareTokenInfo(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(utils.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	hasToken := utils.HasShareToken(userID)
	utils.JSONResponse(w, http.StatusOK, map[string]any{
		"has_token": hasToken,
	})
}

// SharedGetMarkedDays returns days with entries for a given month, using a share token.
func SharedGetMarkedDays(w http.ResponseWriter, r *http.Request) {
	userID, _, err := validateShareToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

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

	content, err := utils.GetMonth(userID, year, month)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving month data: %v", err), http.StatusInternalServerError)
		return
	}

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
			if _, ok := day["text"].(string); ok {
				daysWithLogs = append(daysWithLogs, int(dayNum))
			}
			if files, ok := day["files"].([]any); ok && len(files) > 0 {
				daysWithFiles = append(daysWithFiles, int(dayNum))
			}
			if bookmarked, ok := day["isBookmarked"].(bool); ok && bookmarked {
				daysBookmarked = append(daysBookmarked, int(dayNum))
			}
		}
	}

	utils.JSONResponse(w, http.StatusOK, map[string]any{
		"days_with_logs":  daysWithLogs,
		"days_with_files": daysWithFiles,
		"days_bookmarked": daysBookmarked,
	})
}

// SharedLoadMonthForReading returns decrypted diary entries for a month, using a share token.
func SharedLoadMonthForReading(w http.ResponseWriter, r *http.Request) {
	userID, derivedKey, err := validateShareToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
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

	content, err := utils.GetMonth(userID, year, month)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving month data: %v", err), http.StatusInternalServerError)
		return
	}

	encKey, err := utils.GetEncryptionKey(userID, derivedKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting encryption key: %v", err), http.StatusInternalServerError)
		return
	}

	days, ok := content["days"].([]any)
	if !ok {
		utils.JSONResponse(w, http.StatusOK, []any{})
		return
	}

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

		resultDay := map[string]any{
			"day": int(dayNum),
		}

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

		if tags, ok := day["tags"].([]any); ok && len(tags) > 0 {
			resultDay["tags"] = tags
		}

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

		if _, hasText := resultDay["text"]; hasText {
			result = append(result, resultDay)
		} else if _, hasFiles := resultDay["files"]; hasFiles {
			result = append(result, resultDay)
		} else if _, hasTags := resultDay["tags"]; hasTags {
			result = append(result, resultDay)
		}
	}

	utils.JSONResponse(w, http.StatusOK, result)
}

// SharedDownloadFile decrypts and streams a file, using a share token.
func SharedDownloadFile(w http.ResponseWriter, r *http.Request) {
	userID, derivedKey, err := validateShareToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	uuid := r.URL.Query().Get("uuid")
	if uuid == "" {
		http.Error(w, "Missing uuid parameter", http.StatusBadRequest)
		return
	}

	encKey, err := utils.GetEncryptionKey(userID, derivedKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting encryption key: %v", err), http.StatusInternalServerError)
		return
	}

	encryptedFile, err := utils.ReadFile(userID, uuid)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading file: %v", err), http.StatusInternalServerError)
		return
	}
	defer func() { encryptedFile = nil }()

	decryptedFile, err := utils.DecryptFile(encryptedFile, encKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error decrypting file: %v", err), http.StatusInternalServerError)
		return
	}
	defer func() { decryptedFile = nil }()

	encryptedFile = nil

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment")

	if _, err := w.Write(decryptedFile); err != nil {
		utils.Logger.Printf("Error writing shared file response: %v", err)
	}
}
