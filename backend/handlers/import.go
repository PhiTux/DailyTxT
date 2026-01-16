package handlers

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/phitux/dailytxt/backend/utils"
)

// ImportData handles the import of user data
func ImportData(w http.ResponseWriter, r *http.Request) {
	// 1. Auth check
	val := r.Context().Value(utils.UserIDKey)
	if val == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID, ok := val.(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	valKey := r.Context().Value(utils.DerivedKeyKey)
	if valKey == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	derivedKey, ok := valKey.(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 2. Parse Multipart
	// Up to 50 MB will be kept in memory, rest will be stored in temp files
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		http.Error(w, fmt.Sprintf("Error parsing form: %v", err), http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Missing file part", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}

	encryptedStr := r.FormValue("encrypted")
	isEncrypted := encryptedStr == "true"
	password := r.FormValue("password")

	// Helpers for safe extraction
	getString := func(m map[string]any, key string) string {
		if v, ok := m[key].(string); ok {
			return v
		}
		return ""
	}
	getFloat64 := func(m map[string]any, key string) float64 {
		if v, ok := m[key].(float64); ok {
			return v
		}
		return 0
	}

	// 3. Open Zip
	zipReader, err := zip.NewReader(bytes.NewReader(fileBytes), int64(len(fileBytes)))
	if err != nil {
		http.Error(w, "Invalid zip file", http.StatusBadRequest)
		return
	}

	// 4. Secure Key Derivation (Check Password/Backup codes)
	var importKey string
	var importEncKey string

	if isEncrypted {

		// Find user.json
		var userFile *zip.File
		for _, f := range zipReader.File {
			if f.Name == "user.json" {
				userFile = f
				break
			}
		}

		if userFile == nil {
			http.Error(w, "Invalid backup: user.json missing", http.StatusBadRequest)
			return
		}

		rc, err := userFile.Open()
		if err != nil {
			http.Error(w, "Error opening user.json", http.StatusInternalServerError)
			return
		}

		var userMap map[string]any
		if err := json.NewDecoder(rc).Decode(&userMap); err != nil {
			rc.Close()
			http.Error(w, "Invalid user.json format", http.StatusBadRequest)
			return
		}
		rc.Close()

		storedHash := getString(userMap, "password")
		if storedHash == "" {
			http.Error(w, "Invalid user.json: password missing", http.StatusBadRequest)
			return
		}

		salt := getString(userMap, "salt")

		// Extract encrypted encryption key from userMap
		encEncKey := getString(userMap, "enc_enc_key")
		if encEncKey == "" {
			http.Error(w, "Invalid backup: enc_enc_key missing", http.StatusBadRequest)
			return
		}

		// Verify password
		found := false
		if utils.VerifyPassword(password, storedHash) {
			// Password correct
			dkBytes, err := utils.DeriveKeyFromPassword(password, salt)
			if err != nil {
				http.Error(w, "Error deriving key", http.StatusInternalServerError)
				return
			}
			importKey = base64.StdEncoding.EncodeToString(dkBytes)
			found = true
		} else {
			// Check backup codes
			if codes, ok := userMap["backup_codes"].([]any); ok {
				for _, c := range codes {
					codeMap, ok := c.(map[string]any)
					if !ok {
						continue
					}
					codeHash := getString(codeMap, "password")
					if utils.VerifyPassword(password, codeHash) {
						// Match!
						codeSalt := getString(codeMap, "salt")
						encDerKey := getString(codeMap, "enc_derived_key")

						// Derive temp key from backup code
						tempKeyBytes, err := utils.DeriveKeyFromPassword(password, codeSalt)
						if err != nil {
							continue
						}

						// Decrypt the stored derived_key using the temp key
						// Using URLEncoding here as per security.go logic for backup codes
						decryptedKey, err := utils.DecryptText(encDerKey, base64.URLEncoding.EncodeToString(tempKeyBytes))
						if err == nil {
							importKey = decryptedKey
							found = true
							break
						}
					}
				}
			}
		}

		if !found {
			http.Error(w, "Invalid password or backup code", http.StatusBadRequest)
			return
		}

		// Now that we have the imported derived_key, decrypt enc_enc_key

		// Decode derived key
		derivedKeyBytes, err := base64.StdEncoding.DecodeString(importKey)
		if err != nil {
			http.Error(w, "error decoding derived key", http.StatusInternalServerError)
			return
		}

		// Create Fernet cipher
		aead, err := utils.CreateAEAD(derivedKeyBytes)
		if err != nil {
			http.Error(w, "error creating cipher", http.StatusInternalServerError)
			return
		}

		// Decode encrypted key
		encEncKeyBytes, err := base64.StdEncoding.DecodeString(encEncKey)
		if err != nil {
			http.Error(w, "error decoding encrypted key", http.StatusInternalServerError)
			return
		}

		// Extract nonce from encrypted key
		if len(encEncKeyBytes) < aead.NonceSize() {
			http.Error(w, "encrypted key too short", http.StatusInternalServerError)
			return
		}
		nonce, encKeyBytes := encEncKeyBytes[:aead.NonceSize()], encEncKeyBytes[aead.NonceSize():]

		// Decrypt key
		keyBytes, err := aead.Open(nil, nonce, encKeyBytes, nil)
		if err != nil {
			http.Error(w, "error decrypting key", http.StatusInternalServerError)
			return
		}

		// Return base64-encoded key
		importEncKey = base64.URLEncoding.EncodeToString(keyBytes)
	}

	// Prepare current user encryption key
	currentEncKey, err := utils.GetEncryptionKey(userID, derivedKey)
	if err != nil {
		http.Error(w, "Error getting encryption key", http.StatusInternalServerError)
		return
	}

	// 5. Process Tags

	// Map OldID -> NewID
	tagIDMap := make(map[int]int)

	// Load current tags
	currentTagsRaw, err := utils.GetTags(userID)
	if err != nil {
		http.Error(w, "Error loading tags", http.StatusInternalServerError)
		return
	}

	// If empty init
	if currentTagsRaw["tags"] == nil {
		currentTagsRaw["tags"] = []any{}
		currentTagsRaw["next_id"] = float64(1)
	}

	// Read imported tags
	var importedTagsFile *zip.File
	for _, f := range zipReader.File {
		if f.Name == "tags.json" {
			importedTagsFile = f
			break
		}
	}

	if importedTagsFile != nil {
		rc, _ := importedTagsFile.Open()
		var importedTagsMap map[string]any
		json.NewDecoder(rc).Decode(&importedTagsMap) // Ignore errors
		rc.Close()

		if iTags, ok := importedTagsMap["tags"].([]any); ok {
			cTags := currentTagsRaw["tags"].([]any)
			nextID := int(currentTagsRaw["next_id"].(float64))

			for _, t := range iTags {
				tag := t.(map[string]any)
				oldID := int(getFloat64(tag, "id"))

				// Decrypt properties if needed
				name := getString(tag, "name")
				color := getString(tag, "color")
				icon := getString(tag, "icon")

				if isEncrypted {
					name, _ = utils.DecryptText(name, importEncKey)
					color, _ = utils.DecryptText(color, importEncKey)
					icon, _ = utils.DecryptText(icon, importEncKey)
				}

				// Decrypt existing tags to compare (expensive but necessary)

				// Find duplicate
				var matchID int
				found := false
				for _, cur := range cTags {
					curTag := cur.(map[string]any)
					cName, _ := utils.DecryptText(getString(curTag, "name"), currentEncKey)

					if cName == name {
						matchID = int(getFloat64(curTag, "id"))
						found = true
						break
					}
				}

				if found {
					tagIDMap[oldID] = matchID
				} else {
					// Create new
					newID := nextID
					nextID++
					tagIDMap[oldID] = newID

					// Re-encrypt
					encName, _ := utils.EncryptText(name, currentEncKey)
					encColor, _ := utils.EncryptText(color, currentEncKey)
					encIcon, _ := utils.EncryptText(icon, currentEncKey)

					newTag := map[string]any{
						"id":    float64(newID),
						"name":  encName,
						"color": encColor,
						"icon":  encIcon,
					}
					cTags = append(cTags, newTag)
				}
			}
			currentTagsRaw["tags"] = cTags
			currentTagsRaw["next_id"] = float64(nextID)
			utils.WriteTags(userID, currentTagsRaw)
		}
	}

	// 6. Process Files
	// Map FileKey -> NewUUID and Size
	// FileKey: UUID (if encrypted imp) OR Filename (if decrypted imp)
	type importedFile struct {
		NewUUID string
		Size    int64
	}
	fileMap := make(map[string]importedFile)

	for _, f := range zipReader.File {
		if strings.HasPrefix(f.Name, "files/") && !f.FileInfo().IsDir() {
			fname := filepath.Base(f.Name)
			// Skip dotfiles (Mac artifacts etc)
			if strings.HasPrefix(fname, ".") {
				continue
			}

			// Generate new UUID
			newUUID, _ := utils.GenerateUUID()

			// Read content
			rc, err := f.Open()
			if err != nil {
				utils.Logger.Printf("Error opening zip file %s: %v", f.Name, err)
				continue
			}

			content, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				utils.Logger.Printf("Error reading zip file %s: %v", f.Name, err)
				continue
			}

			// Decrypt if needed
			var finalContent []byte
			if isEncrypted {
				// If encrypted import, file on disk (and in zip) is encrypted.
				// We must decrypt it with importEncKey
				dec, err := utils.DecryptFile(content, importEncKey)
				if err != nil {
					utils.Logger.Printf("Error decrypting imported file %s: %v", fname, err)
					// skip or error?
					continue
				}
				finalContent = dec
				fileMap[fname] = importedFile{NewUUID: newUUID, Size: int64(len(finalContent))} // fname is UUID
			} else {
				// Decrypted import: content is plain.
				finalContent = content
				fileMap[fname] = importedFile{NewUUID: newUUID, Size: int64(len(finalContent))} // fname is Filename
			}

			// Write
			encContent, err := utils.EncryptFile(finalContent, currentEncKey)
			if err != nil {
				utils.Logger.Printf("Error encrypting new file: %v", err)
				continue
			}

			if err := utils.WriteFile(encContent, userID, newUUID); err != nil {
				utils.Logger.Printf("Error writing file %s: %v", newUUID, err)
				continue
			}
		}
	}

	// 7. Process Logs
	for _, f := range zipReader.File {
		// Match YYYY/MM.json
		if strings.HasSuffix(f.Name, ".json") && strings.Contains(f.Name, "/") && !strings.Contains(f.Name, "files/") {
			// Validate it looks like YYYY/MM.json
			parts := strings.Split(f.Name, "/")
			if len(parts) != 2 {
				continue
			} // simple check
			yearStr := parts[0]
			monthStr := strings.TrimSuffix(parts[1], ".json")
			year, err1 := strconv.Atoi(yearStr)
			month, err2 := strconv.Atoi(monthStr)

			if err1 == nil && err2 == nil {
				// Process Month
				rc, _ := f.Open()
				var mData map[string]any
				json.NewDecoder(rc).Decode(&mData)
				rc.Close()

				days, _ := mData["days"].([]any)

				// Load existing month
				currentMonthData, _ := utils.GetMonth(userID, year, month)
				if currentMonthData["days"] == nil {
					currentMonthData["days"] = []any{}
				}
				cDays := currentMonthData["days"].([]any)

				for _, d := range days {
					importDay := d.(map[string]any)
					dayNum := int(getFloat64(importDay, "day"))

					// Decrypt Import Day
					var plainText, plainDate string

					if isEncrypted {
						plainText, _ = utils.DecryptText(getString(importDay, "text"), importEncKey)
						plainDate, _ = utils.DecryptText(getString(importDay, "date_written"), importEncKey)
					} else {
						plainText = getString(importDay, "text")
						plainDate = getString(importDay, "date_written")
					}

					// Handle Files
					var newFiles []any
					if files, ok := importDay["files"].([]any); ok {
						for _, fi := range files {
							fMap := fi.(map[string]any)

							// Find key
							var key string
							var originalFilename string

							if isEncrypted {
								// Encrypted Import: has uuid and enc_filename
								key = getString(fMap, "uuid_filename")
								encName := getString(fMap, "enc_filename")
								originalFilename, _ = utils.DecryptText(encName, importEncKey)
							} else {
								// Decrypted Import: has filename
								key = getString(fMap, "filename")
								originalFilename = key // plain
							}

							if fileInfo, exists := fileMap[key]; exists {
								// Match found
								newEncName, _ := utils.EncryptText(originalFilename, currentEncKey)

								newFiles = append(newFiles, map[string]any{
									"uuid_filename": fileInfo.NewUUID,
									"enc_filename":  newEncName,
									"size":          fileInfo.Size,
								})
							}
						}
					}

					if len(newFiles) > 0 {
						importDay["files"] = newFiles
					} else {
						delete(importDay, "files")
					}

					// Handle Tags
					var newTagIDs []any
					if tags, ok := importDay["tags"].([]any); ok {
						for _, tid := range tags {
							oldID := int(tid.(float64))
							if newID, ok := tagIDMap[oldID]; ok {
								newTagIDs = append(newTagIDs, float64(newID))
							}
						}
					}

					if len(newTagIDs) > 0 {
						importDay["tags"] = newTagIDs
					} else {
						delete(importDay, "tags")
					}

					// Re-Encrypt Text/Date
					if plainText != "" {
						encText, _ := utils.EncryptText(plainText, currentEncKey)
						importDay["text"] = encText
					} else {
						delete(importDay, "text")
					}

					if plainDate != "" {
						encDate, _ := utils.EncryptText(plainDate, currentEncKey)
						importDay["date_written"] = encDate
					} else {
						delete(importDay, "date_written")
					}

					// Merge into cDays
					// Check if day exists
					foundDay := false
					for i, cd := range cDays {
						cDay := cd.(map[string]any)
						if int(getFloat64(cDay, "day")) == dayNum {
							// Exists - Move to History
							foundDay = true

							cText := getString(cDay, "text")
							if cText != "" {
								// Imported data becomes MAIN. Old main becomes history.

								var history []any
								if h, ok := cDay["history"].([]any); ok {
									history = h
								} else {
									history = []any{}
								}

								// Add current main to history
								maxVer := 0
								for _, h := range history {
									if v, ok := h.(map[string]any)["version"].(float64); ok && int(v) > maxVer {
										maxVer = int(v)
									}
								}

								history = append(history, map[string]any{
									"version":      float64(maxVer + 1),
									"text":         cDay["text"],
									"date_written": cDay["date_written"],
								})

								importDay["history"] = history

							} else {
								// Keep old history if any
								if h, ok := cDay["history"].([]any); ok {
									importDay["history"] = h
								}
							}

							// merge existing files to importDay["files"]
							if cFiles, ok := cDay["files"].([]any); ok {
								var impFiles []any
								if f, ok := importDay["files"].([]any); ok {
									impFiles = f
								}

								impFiles = append(impFiles, cFiles...)
								importDay["files"] = impFiles
							}

							cDays[i] = importDay
							break
						}
					}

					if !foundDay {
						cDays = append(cDays, importDay)
					}
				}
				currentMonthData["days"] = cDays
				utils.WriteMonth(userID, year, month, currentMonthData)
			}
		}
	}

	// 8. Process Templates
	var templatesFile *zip.File
	for _, f := range zipReader.File {
		if f.Name == "templates.json" {
			templatesFile = f
			break
		}
	}
	if templatesFile != nil {
		rc, _ := templatesFile.Open()
		var tmplData map[string]any
		json.NewDecoder(rc).Decode(&tmplData)
		rc.Close()

		if items, ok := tmplData["templates"].([]any); ok {
			currTmplData, _ := utils.GetTemplates(userID)
			if currTmplData["templates"] == nil {
				currTmplData["templates"] = []any{}
			}
			cItems := currTmplData["templates"].([]any)

			for _, item := range items {
				t := item.(map[string]any)
				name := getString(t, "name")
				text := getString(t, "text")

				if isEncrypted {
					name, _ = utils.DecryptText(name, importEncKey)
					text, _ = utils.DecryptText(text, importEncKey)
				}

				// Check duplicate
				dup := false
				for _, ci := range cItems {
					ct := ci.(map[string]any)
					cName, _ := utils.DecryptText(getString(ct, "name"), currentEncKey)
					if cName == name {
						dup = true
						break
					}
				}

				if !dup {
					eName, _ := utils.EncryptText(name, currentEncKey)
					eText, _ := utils.EncryptText(text, currentEncKey)
					cItems = append(cItems, map[string]any{"name": eName, "text": eText})
				}
			}
			currTmplData["templates"] = cItems
			utils.WriteTemplates(userID, currTmplData)
		}
	}

	// Success
	utils.JSONResponse(w, http.StatusOK, map[string]any{"success": true})
}
