package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/phitux/dailytxt/backend/handlers"
	"github.com/phitux/dailytxt/backend/middleware"
	"github.com/phitux/dailytxt/backend/utils"
)

// longTimeoutEndpoints defines endpoints that need extended timeouts
var longTimeoutEndpoints = map[string]bool{
	"/logs/uploadFile":   true,
	"/logs/downloadFile": true,
	"/logs/exportData":   true,
	"/users/login":       true,
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

	// Load settings
	if err := utils.InitSettings(); err != nil {
		logger.Fatalf("Failed to initialize settings: %v", err)
	}

	// Check and handle old data migration if needed
	utils.HandleOldData(logger)

	// Create a new router
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("POST /users/login", handlers.Login)
	mux.HandleFunc("GET /users/migrationProgress", handlers.GetMigrationProgress)
	mux.HandleFunc("GET /users/isRegistrationAllowed", handlers.IsRegistrationAllowed)
	mux.HandleFunc("POST /users/register", handlers.RegisterHandler)
	mux.HandleFunc("GET /users/logout", handlers.Logout)
	mux.HandleFunc("GET /users/check", middleware.RequireAuth(handlers.CheckLogin))
	mux.HandleFunc("GET /users/getUserSettings", middleware.RequireAuth(handlers.GetUserSettings))
	mux.HandleFunc("POST /users/saveUserSettings", middleware.RequireAuth(handlers.SaveUserSettings))
	mux.HandleFunc("POST /users/changePassword", middleware.RequireAuth(handlers.ChangePassword))
	mux.HandleFunc("POST /users/deleteAccount", middleware.RequireAuth(handlers.DeleteAccount))
	mux.HandleFunc("POST /users/createBackupCodes", middleware.RequireAuth(handlers.CreateBackupCodes))
	mux.HandleFunc("GET /users/statistics", middleware.RequireAuth(handlers.GetStatistics))

	mux.HandleFunc("POST /logs/saveLog", middleware.RequireAuth(handlers.SaveLog))
	mux.HandleFunc("GET /logs/getLog", middleware.RequireAuth(handlers.GetLog))
	mux.HandleFunc("GET /logs/getMarkedDays", middleware.RequireAuth(handlers.GetMarkedDays))
	mux.HandleFunc("GET /logs/getTags", middleware.RequireAuth(handlers.GetTags))
	mux.HandleFunc("POST /logs/saveNewTag", middleware.RequireAuth(handlers.SaveTags))
	mux.HandleFunc("POST /logs/editTag", middleware.RequireAuth(handlers.EditTag))
	mux.HandleFunc("GET /logs/deleteTag", middleware.RequireAuth(handlers.DeleteTag))
	mux.HandleFunc("POST /logs/addTagToLog", middleware.RequireAuth(handlers.AddTagToLog))
	mux.HandleFunc("POST /logs/removeTagFromLog", middleware.RequireAuth(handlers.RemoveTagFromLog))
	mux.HandleFunc("GET /logs/getTemplates", middleware.RequireAuth(handlers.GetTemplates))
	mux.HandleFunc("POST /logs/saveTemplates", middleware.RequireAuth(handlers.SaveTemplates))
	mux.HandleFunc("GET /logs/getALookBack", middleware.RequireAuth(handlers.GetALookBack))
	mux.HandleFunc("GET /logs/searchString", middleware.RequireAuth(handlers.Search))
	mux.HandleFunc("GET /logs/searchTag", middleware.RequireAuth(handlers.SearchTag))
	mux.HandleFunc("GET /logs/loadMonthForReading", middleware.RequireAuth(handlers.LoadMonthForReading))
	mux.HandleFunc("POST /logs/uploadFile", middleware.RequireAuth(handlers.UploadFile))
	mux.HandleFunc("GET /logs/downloadFile", middleware.RequireAuth(handlers.DownloadFile))
	mux.HandleFunc("GET /logs/deleteFile", middleware.RequireAuth(handlers.DeleteFile))
	mux.HandleFunc("GET /logs/getHistory", middleware.RequireAuth(handlers.GetHistory))
	mux.HandleFunc("GET /logs/bookmarkDay", middleware.RequireAuth(handlers.BookmarkDay))
	mux.HandleFunc("GET /logs/deleteDay", middleware.RequireAuth(handlers.DeleteDay))
	mux.HandleFunc("GET /logs/exportData", middleware.RequireAuth(handlers.ExportData))

	// Admin routes
	mux.HandleFunc("POST /admin/validate-password", handlers.ValidateAdminPassword)
	mux.HandleFunc("POST /admin/users", handlers.GetAllUsers)
	mux.HandleFunc("POST /admin/delete-user", handlers.DeleteUser)

	// Create a handler chain with Timeout, Logger and CORS middleware
	// Timeout middleware will be executed first, then Logger, then CORS
	handler := timeoutMiddleware(middleware.Logger(middleware.CORS(mux)))

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
