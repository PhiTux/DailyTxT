package handlers

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/phitux/dailytxt/backend/utils"
)

// BackupRequest represents the backup request parameters
type BackupRequest struct {
	Username         string `json:"username,omitempty"`
	Password         string `json:"password"`
	Encrypted        bool   `json:"encrypted"`
	StartDate        string `json:"startDate,omitempty"`
	EndDate          string `json:"endDate,omitempty"`
	IncludeFiles     bool   `json:"includeFiles"`
	IncludeTemplates bool   `json:"includeTemplates"`
	IncludeTags      bool   `json:"includeTags"`
	IncludeBookmarks bool   `json:"includeBookmarks"`
}

// Backup handles the export of user data for backup purposes
func Backup(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

	var req BackupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Verify password
	derivedKey, _, err := utils.CheckPasswordForUser(userID, req.Password)
	if err != nil {
		http.Error(w, "Invalid password", http.StatusBadRequest)
		return
	}

	performBackup(w, userID, derivedKey, req)
}

// BackupUser handles the export of user data without login (requires explicit credentials)
func BackupUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req BackupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" {
		http.Error(w, "Username required", http.StatusBadRequest)
		return
	}

	users, err := utils.GetUsers()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var userID int

	usersList, ok := users["users"].([]any)
	if ok {
		for _, u := range usersList {
			user, ok := u.(map[string]any)
			if !ok {
				continue
			}

			if uname, ok := user["username"].(string); ok && strings.EqualFold(uname, req.Username) {
				if id, ok := user["user_id"].(float64); ok {
					userID = int(id)
				}
				break
			}
		}
	}

	// Verify password
	derivedKey, _, err := utils.CheckPasswordForUser(userID, req.Password)
	if err != nil {
		http.Error(w, "Invalid password", http.StatusBadRequest)
		return
	}

	performBackup(w, userID, derivedKey, req)
}

func performBackup(w http.ResponseWriter, userID int, derivedKey string, req BackupRequest) {
	// Defaults
	includeFiles := req.IncludeFiles
	includeTemplates := req.IncludeTemplates
	includeTags := req.IncludeTags
	includeBookmarks := req.IncludeBookmarks

	// Get encryption key if needed (for decryption or file ops)
	encKey, err := utils.GetEncryptionKey(userID, derivedKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting encryption key: %v", err), http.StatusInternalServerError)
		return
	}

	// Set headers
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"backup_user_%d.zip\"", userID))

	zw := zip.NewWriter(w)
	defer zw.Close()

	// 1. Export User Data if encrypted
	if req.Encrypted {
		users, err := utils.GetUsers()
		if err == nil {
			if usersList, ok := users["users"].([]any); ok {
				for _, u := range usersList {
					if userMap, ok := u.(map[string]any); ok {
						if id, ok := userMap["user_id"].(float64); ok && int(id) == userID {
							f, err := zw.Create("user.json")
							if err == nil {
								enc := json.NewEncoder(f)
								enc.SetIndent("", fmt.Sprintf("%*s", utils.Settings.Indent, ""))
								enc.Encode(userMap)
							}
							break
						}
					}
				}
			}
		}
	}

	// 2. Export Tags
	if includeTags {
		tagsContent, err := utils.GetTags(userID)
		if err == nil {
			// Remove next_id
			delete(tagsContent, "next_id")

			// If not encrypted export (readable), decrypt the tags
			if !req.Encrypted {
				if tags, ok := tagsContent["tags"].([]any); ok {
					decryptedTags := []any{}
					for _, t := range tags {
						if tag, ok := t.(map[string]any); ok {
							// Decrypt name, color, icon
							if name, ok := tag["name"].(string); ok {
								if decrypted, err := utils.DecryptText(name, encKey); err == nil {
									tag["name"] = decrypted
								}
							}
							if color, ok := tag["color"].(string); ok {
								if decrypted, err := utils.DecryptText(color, encKey); err == nil {
									tag["color"] = decrypted
								}
							}
							if icon, ok := tag["icon"].(string); ok {
								if decrypted, err := utils.DecryptText(icon, encKey); err == nil {
									tag["icon"] = decrypted
								}
							}
							decryptedTags = append(decryptedTags, tag)
						}
					}
					tagsContent["tags"] = decryptedTags
				}
			}

			// Write to ZIP
			f, err := zw.Create("tags.json")
			if err == nil {
				enc := json.NewEncoder(f)
				enc.SetIndent("", fmt.Sprintf("%*s", utils.Settings.Indent, ""))
				enc.Encode(tagsContent)
			}
		}
	}

	// 3. Export Templates
	if includeTemplates {
		templatesContent, err := utils.GetTemplates(userID)
		if err == nil {
			// If not encrypted export (readable), decrypt the templates
			if !req.Encrypted {
				if templates, ok := templatesContent["templates"].([]any); ok {
					decryptedTemplates := []any{}
					for _, t := range templates {
						if templateMap, ok := t.(map[string]any); ok {
							if name, ok := templateMap["name"].(string); ok {
								if decrypted, err := utils.DecryptText(name, encKey); err == nil {
									templateMap["name"] = decrypted
								}
							}
							if text, ok := templateMap["text"].(string); ok {
								if decrypted, err := utils.DecryptText(text, encKey); err == nil {
									templateMap["text"] = decrypted
								}
							}
							decryptedTemplates = append(decryptedTemplates, templateMap)
						}
					}
					templatesContent["templates"] = decryptedTemplates
				}
			}

			// Write to ZIP
			f, err := zw.Create("templates.json")
			if err == nil {
				enc := json.NewEncoder(f)
				enc.SetIndent("", fmt.Sprintf("%*s", utils.Settings.Indent, ""))
				enc.Encode(templatesContent)
			}
		}
	}

	// 4. Export Log Entries
	// Walk data/<userID>/<year>/<month.json>
	userPath := filepath.Join(utils.Settings.DataPath, fmt.Sprintf("%d", userID))

	// Collect file UUIDs to export (uuid -> targetFilename)
	filesToExport := make(map[string]string)
	// Track used filenames to handle duplicates (filename -> unused)
	usedFilenames := make(map[string]bool)

	entries, err := os.ReadDir(userPath)
	if err == nil {
		for _, yearEntry := range entries {
			if !yearEntry.IsDir() {
				continue
			}
			yearStr := yearEntry.Name()
			year, err := strconv.Atoi(yearStr)
			if err != nil {
				continue
			}

			// Date filtering (Year level optimization)
			// Simple check: if we have full date range.
			// Ideally parsing StartDate/EndDate.
			// Here we process month by month.

			monthDir := filepath.Join(userPath, yearStr)
			monthEntries, err := os.ReadDir(monthDir)
			if err != nil {
				continue
			}

			for _, monthEntry := range monthEntries {
				if monthEntry.IsDir() || !strings.HasSuffix(monthEntry.Name(), ".json") {
					continue
				}
				monthStr := strings.TrimSuffix(monthEntry.Name(), ".json")
				month, err := strconv.Atoi(monthStr)
				if err != nil {
					continue
				}

				// Read Month JSON
				monthContent, err := utils.GetMonth(userID, year, month)
				if err != nil {
					continue
				}

				daysArray, ok := monthContent["days"].([]any)
				if !ok {
					continue
				}

				newDays := []any{}
				hasData := false

				for _, dayInterface := range daysArray {
					day, ok := dayInterface.(map[string]any)
					if !ok {
						continue
					}

					dayNum, ok := day["day"].(float64)
					if !ok {
						continue
					}

					// Check Date Range
					currentDateStr := fmt.Sprintf("%04d-%02d-%02d", year, month, int(dayNum))
					if req.StartDate != "" && currentDateStr < req.StartDate {
						continue
					}
					if req.EndDate != "" && currentDateStr > req.EndDate {
						continue
					}

					// Remove history
					delete(day, "history")

					if !includeTags {
						delete(day, "tags")
					}

					if !includeBookmarks {
						delete(day, "isBookmarked")
					}

					if !includeFiles {
						delete(day, "files")
					} else if req.Encrypted {
						if files, ok := day["files"].([]any); ok {
							for _, f := range files {
								if fileMap, ok := f.(map[string]any); ok {
									uuid := ""
									if u, ok := fileMap["uuid"].(string); ok {
										uuid = u
									} else if u, ok := fileMap["uuid_filename"].(string); ok {
										uuid = u
									}
									if uuid != "" {
										filesToExport[uuid] = uuid
									}
								}
							}
						}
					}

					// Decrypt keys if requested
					if !req.Encrypted {
						if encryptedText, ok := day["text"].(string); ok && encryptedText != "" {
							decryptedText, err := utils.DecryptText(encryptedText, encKey)
							if err == nil {
								day["text"] = decryptedText
							}
						}

						if encryptedDate, ok := day["date_written"].(string); ok && encryptedDate != "" {
							decryptedDate, err := utils.DecryptText(encryptedDate, encKey)
							if err == nil {
								day["date_written"] = decryptedDate
							}
						}

						if includeFiles {
							if files, ok := day["files"].([]any); ok {
								newFiles := []any{}
								for _, f := range files {
									if fileMap, ok := f.(map[string]any); ok {
										// Determine filename
										filename := ""
										uuid := ""
										if u, ok := fileMap["uuid"].(string); ok {
											uuid = u
										} else if u, ok := fileMap["uuid_filename"].(string); ok {
											uuid = u
										}

										if encFilename, ok := fileMap["enc_filename"].(string); ok {
											decryptedFilename, err := utils.DecryptText(encFilename, encKey)
											if err == nil {
												filename = decryptedFilename
											}
										}

										// If we have uuid and filename, handle duplicate resolution for ZIP export
										if includeFiles && uuid != "" && filename != "" {
											// Check if we already processed this UUID (e.g. same file in multiple days?)
											// If yes, reuse the assigned filename
											targetName, exists := filesToExport[uuid]
											if !exists {
												targetName = filename
												if usedFilenames[targetName] {
													// Collision
													ext := filepath.Ext(filename)
													nameNoExt := strings.TrimSuffix(filename, ext)
													counter := 2
													for {
														newName := fmt.Sprintf("%s (%d)%s", nameNoExt, counter, ext)
														if !usedFilenames[newName] {
															targetName = newName
															break
														}
														counter++
													}
												}
												usedFilenames[targetName] = true
												filesToExport[uuid] = targetName
											}
											filename = targetName
										}

										// Only keep filename in decrypted JSON
										newFileMap := map[string]any{
											"filename": filename,
										}
										newFiles = append(newFiles, newFileMap)
									}
								}
								day["files"] = newFiles
							}
						} else {
							delete(day, "files")
						}
					}

					newDays = append(newDays, day)
					hasData = true
				}

				if hasData {
					monthContent["days"] = newDays

					// Write to ZIP
					zipPath := fmt.Sprintf("%d/%02d.json", year, month)
					f, err := zw.Create(zipPath)
					if err == nil {
						enc := json.NewEncoder(f)
						enc.SetIndent("", fmt.Sprintf("%*s", utils.Settings.Indent, ""))
						enc.Encode(monthContent)
					}
				}
			}
		}
	}

	// 5. Export Files
	if includeFiles {
		for uuid, targetName := range filesToExport {
			filePath := filepath.Join(utils.Settings.DataPath, fmt.Sprintf("%d", userID), "files", uuid)
			if _, err := os.Stat(filePath); err == nil {
				rawContent, errRead := os.ReadFile(filePath)
				if errRead == nil {
					var contentToWrite []byte
					if req.Encrypted {
						contentToWrite = rawContent
					} else {
						decrypted, errDec := utils.DecryptFile(rawContent, encKey)
						if errDec == nil {
							contentToWrite = decrypted
						} else {
							utils.Logger.Printf("Error decrypting file %s: %v", uuid, errDec)
							continue
						}
					}

					f, err := zw.Create(fmt.Sprintf("files/%s", targetName))
					if err == nil {
						f.Write(contentToWrite)
					}
				}
			}
		}
	}
}
