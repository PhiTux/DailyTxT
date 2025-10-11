package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/phitux/dailytxt/backend/handlers"
	"github.com/phitux/dailytxt/backend/middleware"
	"github.com/phitux/dailytxt/backend/utils"
)

// Application version - loaded from local 'version' file at startup
var AppVersion string

// longTimeoutEndpoints defines endpoints that need extended/none timeouts
// Paths are checked against the request URL path as seen by the top-level handler.
var longTimeoutEndpoints = map[string]bool{
	"/api/logs/uploadFile":   true,
	"/api/logs/downloadFile": true,
	"/api/logs/exportData":   true,
	"/api/users/login":       true,
}

// timeoutMiddleware applies different timeouts based on the endpoint
func timeoutMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if this endpoint needs a long timeout
		if longTimeoutEndpoints[r.URL.Path] {
			// No timeout for these endpoints - let them run as long as needed
			next.ServeHTTP(w, r)
		} else {
			// Apply 15 second timeout for normal endpoints
			handler := http.TimeoutHandler(next, 15*time.Second, "Request timeout")
			handler.ServeHTTP(w, r)
		}
	})
}

func main() {
	// Setup logging
	logger := log.New(os.Stdout, "dailytxt: ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	logger.Println("Server starting...")

	// Read application version as the very first step
	data, err := os.ReadFile("version")
	if err != nil {
		log.Fatalf("Failed to read 'version' file: %v", err)
	}
	AppVersion = strings.TrimSpace(string(data))
	if AppVersion == "" {
		log.Fatalf("'version' file is empty")
	}

	// Set application version
	utils.SetVersion(AppVersion)
	logger.Printf("DailyTxT version: %s", AppVersion)

	// Load settings
	if err := utils.InitSettings(); err != nil {
		logger.Fatalf("Failed to initialize settings: %v", err)
	}

	// Check and handle old data migration if needed
	utils.HandleOldData(logger)

	// API sub-router
	api := http.NewServeMux()

	// Public routes (no authentication required)
	api.HandleFunc("GET /version", utils.GetVersionInfo)

	// Users
	api.HandleFunc("POST /users/login", handlers.Login)
	api.HandleFunc("GET /users/migrationProgress", handlers.GetMigrationProgress)
	api.HandleFunc("GET /users/isRegistrationAllowed", handlers.IsRegistrationAllowed)
	api.HandleFunc("POST /users/register", handlers.RegisterHandler)
	api.HandleFunc("GET /users/logout", handlers.Logout)
	api.HandleFunc("GET /users/check", middleware.RequireAuth(handlers.CheckLogin))
	api.HandleFunc("GET /users/getUserSettings", middleware.RequireAuth(handlers.GetUserSettings))
	api.HandleFunc("POST /users/saveUserSettings", middleware.RequireAuth(handlers.SaveUserSettings))
	api.HandleFunc("POST /users/changePassword", middleware.RequireAuth(handlers.ChangePassword))
	api.HandleFunc("POST /users/changeUsername", middleware.RequireAuth(handlers.ChangeUsername))
	api.HandleFunc("POST /users/deleteAccount", middleware.RequireAuth(handlers.DeleteAccount))
	api.HandleFunc("POST /users/createBackupCodes", middleware.RequireAuth(handlers.CreateBackupCodes))
	api.HandleFunc("POST /users/validatePassword", middleware.RequireAuth(handlers.ValidatePassword))
	api.HandleFunc("GET /users/statistics", middleware.RequireAuth(handlers.GetStatistics))

	// Logs
	api.HandleFunc("POST /logs/saveLog", middleware.RequireAuth(handlers.SaveLog))
	api.HandleFunc("GET /logs/getLog", middleware.RequireAuth(handlers.GetLog))
	api.HandleFunc("GET /logs/getMarkedDays", middleware.RequireAuth(handlers.GetMarkedDays))
	api.HandleFunc("GET /logs/getTags", middleware.RequireAuth(handlers.GetTags))
	api.HandleFunc("POST /logs/saveNewTag", middleware.RequireAuth(handlers.SaveTags))
	api.HandleFunc("POST /logs/editTag", middleware.RequireAuth(handlers.EditTag))
	api.HandleFunc("GET /logs/deleteTag", middleware.RequireAuth(handlers.DeleteTag))
	api.HandleFunc("POST /logs/addTagToLog", middleware.RequireAuth(handlers.AddTagToLog))
	api.HandleFunc("POST /logs/removeTagFromLog", middleware.RequireAuth(handlers.RemoveTagFromLog))
	api.HandleFunc("GET /logs/getTemplates", middleware.RequireAuth(handlers.GetTemplates))
	api.HandleFunc("POST /logs/saveTemplates", middleware.RequireAuth(handlers.SaveTemplates))
	api.HandleFunc("GET /logs/getALookBack", middleware.RequireAuth(handlers.GetALookBack))
	api.HandleFunc("GET /logs/searchString", middleware.RequireAuth(handlers.Search))
	api.HandleFunc("GET /logs/searchTag", middleware.RequireAuth(handlers.SearchTag))
	api.HandleFunc("GET /logs/loadMonthForReading", middleware.RequireAuth(handlers.LoadMonthForReading))
	api.HandleFunc("POST /logs/uploadFile", middleware.RequireAuth(handlers.UploadFile))
	api.HandleFunc("GET /logs/downloadFile", middleware.RequireAuth(handlers.DownloadFile))
	api.HandleFunc("GET /logs/deleteFile", middleware.RequireAuth(handlers.DeleteFile))
	api.HandleFunc("POST /logs/renameFile", middleware.RequireAuth(handlers.RenameFile))
	api.HandleFunc("POST /logs/reorderFiles", middleware.RequireAuth(handlers.ReorderFiles))
	api.HandleFunc("GET /logs/getHistory", middleware.RequireAuth(handlers.GetHistory))
	api.HandleFunc("GET /logs/bookmarkDay", middleware.RequireAuth(handlers.BookmarkDay))
	api.HandleFunc("GET /logs/deleteDay", middleware.RequireAuth(handlers.DeleteDay))
	api.HandleFunc("GET /logs/exportData", middleware.RequireAuth(handlers.ExportData))

	// Admin routes
	api.HandleFunc("POST /admin/validate-password", middleware.RequireAuth(handlers.ValidateAdminPassword))
	api.HandleFunc("POST /admin/get-data", middleware.RequireAuth(handlers.GetAdminData))
	api.HandleFunc("POST /admin/delete-user", middleware.RequireAuth(handlers.DeleteUser))
	api.HandleFunc("POST /admin/delete-old-data", middleware.RequireAuth(handlers.DeleteOldData))
	api.HandleFunc("POST /admin/open-registration", middleware.RequireAuth(handlers.OpenRegistrationTemp))

	// Root mux mounts API under /api/
	rootMux := http.NewServeMux()
	rootMux.Handle("/api/", http.StripPrefix("/api", api))

	var handler http.Handler = rootMux

	// Create a handler chain with Timeout, Logger and CORS middleware
	// Timeout middleware will be executed first, then Logger, then CORS
	if len(utils.Settings.AllowedHosts) == 0 {
		logger.Println("Warning: ALLOWED_HOSTS is empty, CORS will not allow any cross-origin requests")
	} else {
		handler = middleware.CORS(rootMux)
	}
	handler = timeoutMiddleware(middleware.Logger(handler))

	// Create the server without ReadTimeout/WriteTimeout (managed by middleware)
	server := &http.Server{
		Addr:        ":8000",
		Handler:     handler,
		IdleTimeout: 60 * time.Second, // Keep IdleTimeout for cleanup
	}

	// Start the server in a goroutine
	go func() {
		logger.Println("Server listening on :8000")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Println("Shutting down server...")

	// Create a deadline to wait for
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Doesn't block if no connections, otherwise wait until the timeout
	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Println("Server stopped gracefully")
}
