package webui

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ithena-one/Ithena/packages/cli/localstore"
	"github.com/gorilla/mux"
	"github.com/ithena-one/Ithena/packages/cli/auth"
	"github.com/zalando/go-keyring"
)

//go:embed all:assets
var embeddedAssets embed.FS // Embed the assets directory

var verbose bool

// SetVerbose enables or disables verbose logging for the webui package.
func SetVerbose(v bool) {
	verbose = v
}

const defaultPort = 8675
const ithenaPlatformURL = "https://app.ithena.io"

type apiError struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(apiError{Error: message})
}

// AuthStatusResponse defines the structure for the auth status API response
type AuthStatusResponse struct {
	Authenticated bool   `json:"authenticated"`
	PlatformURL   string `json:"platformURL"`
}

func authStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	token, err := auth.GetToken()

	if err != nil {
		if err == keyring.ErrNotFound {
			// Token not found, definitely not authenticated
			json.NewEncoder(w).Encode(AuthStatusResponse{Authenticated: false, PlatformURL: ithenaPlatformURL})
		} else {
			// Some other error occurred trying to get the token
			log.Printf("Error getting token for auth status: %v", err)
			json.NewEncoder(w).Encode(AuthStatusResponse{Authenticated: false, PlatformURL: ithenaPlatformURL})
		}
		return
	}

	if token != "" {
		// Token exists and is not empty
		json.NewEncoder(w).Encode(AuthStatusResponse{Authenticated: true, PlatformURL: ithenaPlatformURL})
	} else {
		// Token is empty (should ideally be caught by GetToken returning an error, but handle defensively)
		json.NewEncoder(w).Encode(AuthStatusResponse{Authenticated: false, PlatformURL: ithenaPlatformURL})
	}
}

// StartServer initializes and starts the local HTTP server for viewing logs.
func StartServer(port int) {
	if verbose {
		log.Printf("WebUI: Attempting to start server on port %d...", port)
	}

	address := fmt.Sprintf("localhost:%d", port)

	assetsFS, err := fs.Sub(embeddedAssets, "assets")
	if err != nil {
		log.Fatalf("WebUI Fatal: Failed to create sub FS for embedded assets: %v", err)
	}

	router := mux.NewRouter()

	// API routes
	apiRouter := router.PathPrefix("/api").Subrouter()
	apiRouter.HandleFunc("/logs", logsHandler).Methods("GET")
	apiRouter.HandleFunc("/logs/{id}", logDetailHandler).Methods("GET")
	apiRouter.HandleFunc("/auth/status", authStatusHandler).Methods("GET")

	// Serve static assets from the 'assets' directory, accessed via /static/ prefix
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.FS(assetsFS))))

	// Serve index.html for the root path
	router.HandleFunc("/", rootHandler(assetsFS))

	srv := &http.Server{
		Addr:    address,
		Handler: router,
	}

	// Channel to listen for OS signals
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	// Goroutine to start the server
	go func() {
		log.Printf("WebUI: Starting server. Please open your browser to http://%s", address)
		openBrowser(fmt.Sprintf("http://%s", address))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("WebUI Fatal: Could not listen on %s: %v\n", address, err)
		}
	}()

	// Block until a signal is received
	<-stopChan

	log.Println("WebUI: Shutting down server...")

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("WebUI Fatal: Server forced to shutdown: %v", err)
	}

	log.Println("WebUI: Server exited gracefully")
}

func rootHandler(assetsFS fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		file, err := assetsFS.Open("index.html")
		if err != nil {
			log.Printf("WebUI Error: Could not open embedded index.html from assetsFS: %v", err)
			http.Error(w, "Could not load application.", http.StatusInternalServerError)
			return
		}
		defer file.Close()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, err = io.Copy(w, file)
		if err != nil {
			log.Printf("WebUI Error: Could not write embedded index.html to response: %v", err)
		}
	}
}

func logsHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	pageStr := query.Get("page")
	limitStr := query.Get("limit")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20 // Default limit
	}

	filters := localstore.LogQueryFilters{
		Status:    query.Get("status"),
		ToolName:  query.Get("tool_name"),
		McpMethod: query.Get("mcp_method"),
		SearchTerm: query.Get("search"),
	}

	result, err := localstore.QueryLogs(filters, page, limit)
	if err != nil {
		log.Printf("WebUI API Error: Failed to query logs: %v", err)
		http.Error(w, "Failed to retrieve logs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("WebUI API Error: Failed to encode logs response: %v", err)
	}
}

func logDetailHandler(w http.ResponseWriter, r *http.Request) {
	// Assumes path like /api/logs/some-uuid
	// The trailing slash in HandleFunc registration means this matches /api/logs/*
	id := strings.TrimPrefix(r.URL.Path, "/api/logs/")
	if id == "" { // Should not happen if path is /api/logs/some-id but good to check
		http.Error(w, "Log ID is required in the path", http.StatusBadRequest)
		return
	}

	logEntry, err := localstore.GetLogByID(id)
	if err != nil {
		log.Printf("WebUI API Error: Failed to get log by ID %s: %v", id, err)
		http.Error(w, "Failed to retrieve log details", http.StatusInternalServerError)
		return
	}

	if logEntry == nil { // localstore.GetLogByID returns nil, nil if not found
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(logEntry); err != nil {
		log.Printf("WebUI API Error: Failed to encode log detail response for ID %s: %v", id, err)
	}
}

// openBrowser tries to open the URL in the default web browser.
func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start() // .Start() makes it non-blocking
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin": // macOS
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform for opening browser automatically")
	}
	if err != nil {
		log.Printf("WebUI Info: Failed to open browser automatically: %v. Please open manually.", err)
	}
} 