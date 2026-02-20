package handlers

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/phitux/dailytxt/backend/utils"
)

// validateShareToken decodes and validates a share token from the request query parameter.
// Returns (userID, derivedKey, tokenHash, error).
func validateShareToken(r *http.Request) (int, string, string, error) {
	token := r.URL.Query().Get("token")
	if token == "" {
		return 0, "", "", fmt.Errorf("missing token parameter")
	}

	// Decode the token bytes from base64 URL encoding
	tokenBytes, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return 0, "", "", fmt.Errorf("invalid token format")
	}

	// Compute SHA-256 hash of the raw token bytes for lookup
	hash := sha256.Sum256(tokenBytes)
	tokenHash := base64.URLEncoding.EncodeToString(hash[:])

	// Look up user by token hash
	userID, encDerivedKey, err := utils.GetUserByShareTokenHash(tokenHash)
	if err != nil {
		return 0, "", "", fmt.Errorf("invalid share token")
	}

	// Decrypt the derived key using the full token as the encryption key
	derivedKey, err := utils.DecryptText(encDerivedKey, token)
	if err != nil {
		return 0, "", "", fmt.Errorf("error decrypting derived key")
	}

	return userID, derivedKey, tokenHash, nil
}

func hasValidShareVerificationCookie(r *http.Request, tokenHash string) bool {
	cookie, err := r.Cookie(utils.ShareVerificationCookieName)
	if err != nil {
		return false
	}

	return utils.ValidateShareVerificationCookieValue(cookie.Value, tokenHash)
}

func getVerifiedShareEmail(r *http.Request, tokenHash string) string {
	cookie, err := r.Cookie(utils.ShareVerificationCookieName)
	if err != nil {
		return ""
	}

	email, ok := utils.GetShareVerificationEmailFromCookieValue(cookie.Value, tokenHash)
	if !ok {
		return ""
	}

	return email
}

func getClientIP(r *http.Request) string {
	forwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	realIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}

	return r.RemoteAddr
}

func logShareAccess(userID int, email, ip, event, path string) {
	if err := utils.AddShareAccessLog(userID, email, ip, event, path, time.Now()); err != nil {
		utils.Logger.Printf("Failed to add share access log for user %d: %v", userID, err)
	}
}

type requestShareVerificationCodeRequest struct {
	Email string `json:"email"`
}

type verifyShareVerificationCodeRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type shareVerificationSettingsRequest struct {
	Emails []string `json:"emails"`
}

type shareSMTPSettingsRequest struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
}

type testShareSMTPRequest struct {
	ToEmail  string `json:"to_email"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
}

// GetShareVerificationSettings returns user-specific share verification settings.
func GetShareVerificationSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(utils.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	emails, err := utils.GetShareEmailWhitelist(userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving share verification settings: %v", err), http.StatusInternalServerError)
		return
	}

	utils.JSONResponse(w, http.StatusOK, map[string]any{
		"emails":          emails,
		"smtp_configured": utils.IsShareVerificationEnabled(),
	})
}

// GetShareSMTPSettings returns SMTP settings used for share verification emails.
func GetShareSMTPSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(utils.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userSettings, err := utils.GetShareSMTPSettings(userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving SMTP settings: %v", err), http.StatusInternalServerError)
		return
	}

	effectiveSettings, usingGlobalDefault, err := utils.GetEffectiveShareSMTPSettings(userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving SMTP settings: %v", err), http.StatusInternalServerError)
		return
	}

	utils.JSONResponse(w, http.StatusOK, map[string]any{
		"settings":             userSettings,
		"effective_settings":   effectiveSettings,
		"using_global_default": usingGlobalDefault,
	})
}

// SaveShareSMTPSettings saves SMTP settings used for share verification emails.
func SaveShareSMTPSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(utils.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req shareSMTPSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	settings := utils.ShareSMTPSettings{
		Host:     strings.TrimSpace(req.Host),
		Port:     req.Port,
		Username: strings.TrimSpace(req.Username),
		Password: req.Password,
		From:     utils.NormalizeEmailAddress(req.From),
	}

	if settings.Port <= 0 {
		settings.Port = 587
	}

	if settings.Host != "" || settings.From != "" {
		if settings.Host == "" || settings.From == "" {
			http.Error(w, "Host and From must both be provided", http.StatusBadRequest)
			return
		}
		if !utils.IsValidEmailAddress(settings.From) {
			http.Error(w, "Invalid from email address", http.StatusBadRequest)
			return
		}
	}

	if err := utils.SaveShareSMTPSettings(userID, settings); err != nil {
		http.Error(w, fmt.Sprintf("Error saving SMTP settings: %v", err), http.StatusInternalServerError)
		return
	}

	utils.JSONResponse(w, http.StatusOK, map[string]any{
		"success":  true,
		"settings": settings,
	})
}

// TestShareSMTP sends a test email using provided or saved SMTP settings.
func TestShareSMTP(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(utils.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req testShareSMTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	toEmail := utils.NormalizeEmailAddress(req.ToEmail)
	if !utils.IsValidEmailAddress(toEmail) {
		http.Error(w, "Invalid test recipient email", http.StatusBadRequest)
		return
	}

	settings := utils.ShareSMTPSettings{
		Host:     strings.TrimSpace(req.Host),
		Port:     req.Port,
		Username: strings.TrimSpace(req.Username),
		Password: req.Password,
		From:     utils.NormalizeEmailAddress(req.From),
	}

	if settings.Host == "" && settings.From == "" {
		effectiveSettings, _, err := utils.GetEffectiveShareSMTPSettings(userID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error loading SMTP settings: %v", err), http.StatusInternalServerError)
			return
		}
		settings = effectiveSettings
	}

	if err := utils.SendSMTPTestEmailWithSettings(settings, toEmail); err != nil {
		http.Error(w, fmt.Sprintf("Failed to send test email: %v", err), http.StatusBadRequest)
		return
	}

	utils.JSONResponse(w, http.StatusOK, map[string]any{
		"success": true,
	})
}

// ClearShareAccessLogs clears all stored share access logs for the authenticated user.
func ClearShareAccessLogs(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(utils.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := utils.ClearShareAccessLogs(userID); err != nil {
		http.Error(w, fmt.Sprintf("Error clearing share access logs: %v", err), http.StatusInternalServerError)
		return
	}

	utils.JSONResponse(w, http.StatusOK, map[string]any{
		"success": true,
	})
}

// SaveShareVerificationSettings saves user-specific share verification settings.
func SaveShareVerificationSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(utils.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req shareVerificationSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	normalizedEmails := make([]string, 0, len(req.Emails))
	seen := map[string]bool{}
	for _, email := range req.Emails {
		normalized := utils.NormalizeEmailAddress(email)
		if normalized == "" {
			continue
		}
		if !utils.IsValidEmailAddress(normalized) {
			http.Error(w, fmt.Sprintf("Invalid email address: %s", email), http.StatusBadRequest)
			return
		}
		if seen[normalized] {
			continue
		}
		seen[normalized] = true
		normalizedEmails = append(normalizedEmails, normalized)
	}

	if err := utils.SaveShareEmailWhitelist(userID, normalizedEmails); err != nil {
		http.Error(w, fmt.Sprintf("Error saving share verification settings: %v", err), http.StatusInternalServerError)
		return
	}

	utils.JSONResponse(w, http.StatusOK, map[string]any{
		"success": true,
		"emails":  normalizedEmails,
	})
}

// GetShareAccessLogs returns share access logs for the authenticated user.
func GetShareAccessLogs(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(utils.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	logs, err := utils.GetShareAccessLogs(userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving share access logs: %v", err), http.StatusInternalServerError)
		return
	}

	utils.JSONResponse(w, http.StatusOK, map[string]any{
		"logs": logs,
	})
}

// ShareVerificationStatus returns verification status for a shared link.
func ShareVerificationStatus(w http.ResponseWriter, r *http.Request) {
	userID, _, tokenHash, err := validateShareToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	required, err := utils.IsShareVerificationEnabledForUser(userID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	verified := !required || hasValidShareVerificationCookie(r, tokenHash)

	utils.JSONResponse(w, http.StatusOK, map[string]any{
		"required": required,
		"verified": verified,
	})
}

// RequestShareVerificationCode validates email and sends a one-time verification code.
func RequestShareVerificationCode(w http.ResponseWriter, r *http.Request) {
	userID, _, tokenHash, err := validateShareToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	required, err := utils.IsShareVerificationEnabledForUser(userID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !required {
		http.Error(w, "Share verification is not configured", http.StatusBadRequest)
		return
	}

	var req requestShareVerificationCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	email := utils.NormalizeEmailAddress(req.Email)
	if !utils.IsValidEmailAddress(email) {
		http.Error(w, "Invalid email address", http.StatusBadRequest)
		return
	}

	whitelisted, err := utils.IsShareEmailWhitelistedForUser(userID, email)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !whitelisted {
		http.Error(w, "Email not allowed", http.StatusForbidden)
		return
	}

	code, err := utils.GenerateSixDigitCode()
	if err != nil {
		http.Error(w, "Failed to generate verification code", http.StatusInternalServerError)
		return
	}

	expiresAt := time.Now().Add(time.Duration(utils.Settings.ShareCodeTTLMinutes) * time.Minute)
	utils.StoreShareVerificationCode(tokenHash, email, code, expiresAt)

	if err := utils.SendShareVerificationEmailForUser(userID, email, code); err != nil {
		http.Error(w, "Failed to send verification code", http.StatusInternalServerError)
		return
	}

	logShareAccess(userID, email, getClientIP(r), "code_requested", r.URL.Path)

	utils.JSONResponse(w, http.StatusOK, map[string]any{
		"success": true,
	})
}

// VerifyShareVerificationCode checks submitted code and sets a 1-month cookie.
func VerifyShareVerificationCode(w http.ResponseWriter, r *http.Request) {
	userID, _, tokenHash, err := validateShareToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	required, err := utils.IsShareVerificationEnabledForUser(userID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !required {
		http.Error(w, "Share verification is not configured", http.StatusBadRequest)
		return
	}

	var req verifyShareVerificationCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	email := utils.NormalizeEmailAddress(req.Email)
	code := strings.TrimSpace(req.Code)

	if !utils.IsValidEmailAddress(email) || code == "" {
		http.Error(w, "Invalid email or code", http.StatusBadRequest)
		return
	}

	if !utils.VerifyShareVerificationCode(tokenHash, email, code) {
		http.Error(w, "Invalid or expired verification code", http.StatusForbidden)
		return
	}

	expiresAt := time.Now().Add(time.Duration(utils.Settings.ShareCookieDays) * 24 * time.Hour)
	cookieValue, err := utils.BuildShareVerificationCookieValue(tokenHash, email, expiresAt)
	if err != nil {
		http.Error(w, "Failed to create verification session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     utils.ShareVerificationCookieName,
		Value:    cookieValue,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		Expires:  expiresAt,
	})

	logShareAccess(userID, email, getClientIP(r), "verified", r.URL.Path)

	utils.JSONResponse(w, http.StatusOK, map[string]any{
		"success": true,
	})
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
	userID, _, tokenHash, err := validateShareToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	required, err := utils.IsShareVerificationEnabledForUser(userID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if required && !hasValidShareVerificationCookie(r, tokenHash) {
		http.Error(w, "Verification required", http.StatusForbidden)
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

	if required {
		logShareAccess(userID, getVerifiedShareEmail(r, tokenHash), getClientIP(r), "access", r.URL.Path)
	}
}

// SharedLoadMonthForReading returns decrypted diary entries for a month, using a share token.
func SharedLoadMonthForReading(w http.ResponseWriter, r *http.Request) {
	userID, derivedKey, tokenHash, err := validateShareToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	required, err := utils.IsShareVerificationEnabledForUser(userID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if required && !hasValidShareVerificationCookie(r, tokenHash) {
		http.Error(w, "Verification required", http.StatusForbidden)
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

	if required {
		logShareAccess(userID, getVerifiedShareEmail(r, tokenHash), getClientIP(r), "access", r.URL.Path)
	}
}

// SharedDownloadFile decrypts and streams a file, using a share token.
func SharedDownloadFile(w http.ResponseWriter, r *http.Request) {
	userID, derivedKey, tokenHash, err := validateShareToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	required, err := utils.IsShareVerificationEnabledForUser(userID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if required && !hasValidShareVerificationCookie(r, tokenHash) {
		http.Error(w, "Verification required", http.StatusForbidden)
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

	if required {
		logShareAccess(userID, getVerifiedShareEmail(r, tokenHash), getClientIP(r), "access", r.URL.Path)
	}
}
