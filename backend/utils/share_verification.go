package utils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const ShareVerificationCookieName = "share_verification"

type ShareSMTPSettings struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
}

type shareVerificationCodeEntry struct {
	Code      string
	ExpiresAt time.Time
}

type shareVerificationCookiePayload struct {
	TokenHash string `json:"token_hash"`
	Email     string `json:"email"`
	Exp       int64  `json:"exp"`
}

var (
	shareVerificationCodeStore = map[string]shareVerificationCodeEntry{}
	shareVerificationCodeMu    sync.RWMutex
)

func IsShareVerificationEnabled() bool {
	return Settings.SMTPHost != "" &&
		Settings.SMTPFrom != ""
}

func IsShareSMTPSettingsConfigured(settings ShareSMTPSettings) bool {
	return strings.TrimSpace(settings.Host) != "" && strings.TrimSpace(settings.From) != ""
}

func GetEffectiveShareSMTPSettings(userID int) (ShareSMTPSettings, bool, error) {
	userSettings, err := GetShareSMTPSettings(userID)
	if err != nil {
		return ShareSMTPSettings{}, false, err
	}

	if IsShareSMTPSettingsConfigured(userSettings) {
		if userSettings.Port <= 0 {
			userSettings.Port = 587
		}
		return userSettings, false, nil
	}

	global := ShareSMTPSettings{
		Host:     Settings.SMTPHost,
		Port:     Settings.SMTPPort,
		Username: Settings.SMTPUsername,
		Password: Settings.SMTPPassword,
		From:     Settings.SMTPFrom,
	}
	if global.Port <= 0 {
		global.Port = 587
	}

	return global, true, nil
}

func IsShareVerificationEnabledForUser(userID int) (bool, error) {
	settings, _, err := GetEffectiveShareSMTPSettings(userID)
	if err != nil {
		return false, err
	}

	if !IsShareSMTPSettingsConfigured(settings) {
		return false, nil
	}

	whitelist, err := GetShareEmailWhitelist(userID)
	if err != nil {
		return false, err
	}

	return len(whitelist) > 0, nil
}

func NormalizeEmailAddress(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func IsValidEmailAddress(email string) bool {
	parsed, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	return NormalizeEmailAddress(parsed.Address) == NormalizeEmailAddress(email)
}

func IsShareEmailWhitelisted(email string, whitelist []string) bool {
	normalized := NormalizeEmailAddress(email)
	for _, allowed := range whitelist {
		if normalized == NormalizeEmailAddress(allowed) {
			return true
		}
	}
	return false
}

func IsShareEmailWhitelistedForUser(userID int, email string) (bool, error) {
	whitelist, err := GetShareEmailWhitelist(userID)
	if err != nil {
		return false, err
	}

	return IsShareEmailWhitelisted(email, whitelist), nil
}

func generateShareCodeStoreKey(tokenHash, email string) string {
	return tokenHash + "|" + NormalizeEmailAddress(email)
}

func GenerateSixDigitCode() (string, error) {
	max := int64(1000000)
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	var value int64
	for _, b := range buf {
		value = (value << 8) | int64(b)
	}
	value = value % max
	return fmt.Sprintf("%06d", value), nil
}

func StoreShareVerificationCode(tokenHash, email, code string, expiresAt time.Time) {
	shareVerificationCodeMu.Lock()
	defer shareVerificationCodeMu.Unlock()

	shareVerificationCodeStore[generateShareCodeStoreKey(tokenHash, email)] = shareVerificationCodeEntry{
		Code:      code,
		ExpiresAt: expiresAt,
	}
}

func VerifyShareVerificationCode(tokenHash, email, code string) bool {
	shareVerificationCodeMu.Lock()
	defer shareVerificationCodeMu.Unlock()

	key := generateShareCodeStoreKey(tokenHash, email)
	entry, exists := shareVerificationCodeStore[key]
	if !exists {
		return false
	}

	if time.Now().After(entry.ExpiresAt) {
		delete(shareVerificationCodeStore, key)
		return false
	}

	if strings.TrimSpace(code) != entry.Code {
		return false
	}

	delete(shareVerificationCodeStore, key)
	return true
}

func SendShareVerificationEmail(toEmail, code string) error {
	settings := ShareSMTPSettings{
		Host:     Settings.SMTPHost,
		Port:     Settings.SMTPPort,
		Username: Settings.SMTPUsername,
		Password: Settings.SMTPPassword,
		From:     Settings.SMTPFrom,
	}
	return SendShareVerificationEmailWithSettings(settings, toEmail, code)
}

func SendShareVerificationEmailForUser(userID int, toEmail, code string) error {
	settings, _, err := GetEffectiveShareSMTPSettings(userID)
	if err != nil {
		return err
	}

	return SendShareVerificationEmailWithSettings(settings, toEmail, code)
}

func SendShareVerificationEmailWithSettings(settings ShareSMTPSettings, toEmail, code string) error {
	if !IsShareSMTPSettingsConfigured(settings) {
		return fmt.Errorf("SMTP is not configured")
	}
	if settings.Port <= 0 {
		settings.Port = 587
	}

	from := settings.From
	addr := settings.Host + ":" + strconv.Itoa(settings.Port)

	var auth smtp.Auth
	if settings.Username != "" {
		auth = smtp.PlainAuth("", settings.Username, settings.Password, settings.Host)
	}

	subject := "DailyTxT share verification code"
	body := "Your verification code is: " + code + "\r\n\r\nThis code expires in " + strconv.Itoa(Settings.ShareCodeTTLMinutes) + " minutes."
	message := "From: " + from + "\r\n" +
		"To: " + toEmail + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		body + "\r\n"

	return smtp.SendMail(addr, auth, from, []string{toEmail}, []byte(message))
}

func SendSMTPTestEmailWithSettings(settings ShareSMTPSettings, toEmail string) error {
	if !IsValidEmailAddress(strings.TrimSpace(toEmail)) {
		return fmt.Errorf("invalid test recipient email")
	}

	if !IsShareSMTPSettingsConfigured(settings) {
		return fmt.Errorf("SMTP is not configured")
	}
	if settings.Port <= 0 {
		settings.Port = 587
	}

	from := settings.From
	addr := settings.Host + ":" + strconv.Itoa(settings.Port)

	var auth smtp.Auth
	if settings.Username != "" {
		auth = smtp.PlainAuth("", settings.Username, settings.Password, settings.Host)
	}

	subject := "DailyTxT SMTP test email"
	body := "This is a test email from DailyTxT share verification settings."
	message := "From: " + from + "\r\n" +
		"To: " + toEmail + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		body + "\r\n"

	return smtp.SendMail(addr, auth, from, []string{toEmail}, []byte(message))
}

func BuildShareVerificationCookieValue(tokenHash, email string, expiresAt time.Time) (string, error) {
	payload := shareVerificationCookiePayload{
		TokenHash: tokenHash,
		Email:     NormalizeEmailAddress(email),
		Exp:       expiresAt.Unix(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	signature := signShareVerificationPayload(encodedPayload)
	return encodedPayload + "." + signature, nil
}

func ValidateShareVerificationCookieValue(value, tokenHash string) bool {
	payload, ok := parseShareVerificationCookieValue(value, tokenHash)
	return ok && payload.Email != ""
}

func GetShareVerificationEmailFromCookieValue(value, tokenHash string) (string, bool) {
	payload, ok := parseShareVerificationCookieValue(value, tokenHash)
	if !ok || payload.Email == "" {
		return "", false
	}
	return payload.Email, true
}

func parseShareVerificationCookieValue(value, tokenHash string) (shareVerificationCookiePayload, bool) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return shareVerificationCookiePayload{}, false
	}

	payloadPart := parts[0]
	signaturePart := parts[1]

	expectedSignature := signShareVerificationPayload(payloadPart)
	if !hmac.Equal([]byte(signaturePart), []byte(expectedSignature)) {
		return shareVerificationCookiePayload{}, false
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadPart)
	if err != nil {
		return shareVerificationCookiePayload{}, false
	}

	var payload shareVerificationCookiePayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return shareVerificationCookiePayload{}, false
	}

	if payload.TokenHash != tokenHash {
		return shareVerificationCookiePayload{}, false
	}

	if time.Now().After(time.Unix(payload.Exp, 0)) {
		return shareVerificationCookiePayload{}, false
	}

	return payload, true
}

func signShareVerificationPayload(encodedPayload string) string {
	mac := hmac.New(sha256.New, []byte(Settings.SecretToken))
	mac.Write([]byte(encodedPayload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
