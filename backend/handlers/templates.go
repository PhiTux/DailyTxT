package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/phitux/dailytxt/backend/utils"
)

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
