package observability

import (
	"bytes"
	// "crypto/tls" // Unused
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os" // For os.Stderr for info message
	"sync"
	"time"

	"github.com/fatih/color" // For colored output
	"github.com/google/uuid"
	"github.com/ithena-one/Ithena/packages/cli/auth"
	"github.com/ithena-one/Ithena/packages/cli/jsonrpc"
	"github.com/ithena-one/Ithena/packages/cli/localstore" // Import for local storage
	"github.com/ithena-one/Ithena/packages/cli/telemetry"  // Import telemetry package
	"github.com/ithena-one/Ithena/packages/cli/types"      // Import the new types package
)

// verbose controls internal debug logging for this package
var verbose bool

// SetVerbose enables or disables verbose logging for the observability package.
func SetVerbose(v bool) {
	verbose = v
	localstore.SetVerbose(v) // Pass verbosity to localstore as well
}

// --- Struct for Observability Payload (Matches API) ---
// AuditRecord struct is now defined in the types package
// type AuditRecord struct { ... }

var (
	ProxyVersion = "0.1.0-dev"
)

const (
	logChannelBufferSize = 100
	defaultBatchSize     = 20
	defaultBatchInterval = 15 * time.Second
)

type logJob struct {
	record     types.AuditRecord // Use types.AuditRecord
	observeUrl string
}

var (
	logChan           chan logJob
	wg                sync.WaitGroup
	bufferMutex       sync.Mutex
	logBuffer         []types.AuditRecord // Use types.AuditRecord
	lastSentTime      time.Time
	batchSize         = defaultBatchSize
	batchInterval     = defaultBatchInterval
	currentObserveUrl string

	// For local logging mode message and DB init
	localLogInfoOnce sync.Once
	localDBInitOnce  sync.Once
)

func InitObservability() {
	logChan = make(chan logJob, logChannelBufferSize)
	logBuffer = make([]types.AuditRecord, 0, batchSize)
	lastSentTime = time.Now()
	wg.Add(1)
	go logSender()
	// Don't initialize local DB here; do it on first actual need if not authenticated.
	log.Println("Observability worker started.")
}

func ShutdownObservability() {
	log.Println("Observability: Shutting down...")
	close(logChan)
	wg.Wait()
	telemetry.Shutdown() // Call telemetry shutdown
	log.Println("Observability worker stopped gracefully.")
}

func logSender() {
	defer wg.Done()
	ticker := time.NewTicker(batchInterval / 2)
	defer ticker.Stop()

	for {
		select {
		case job, ok := <-logChan:
			if !ok {
				if verbose {
					log.Println("Observability: Log channel closed, flushing remaining buffer...")
				}
				flushBuffer() // Will handle local save or remote send based on auth status
				return
			}

			bufferMutex.Lock()
			if len(logBuffer) == 0 {
				currentObserveUrl = job.observeUrl
			}
			if job.observeUrl == currentObserveUrl || len(logBuffer) == 0 {
				if len(logBuffer) == 0 {
					currentObserveUrl = job.observeUrl
				}
				logBuffer = append(logBuffer, job.record)
				if verbose {
					log.Printf("Observability: Added Record ID %s to buffer (Size: %d)", job.record.ID, len(logBuffer))
				}
			} else {
				if verbose {
					log.Printf("Observability Info: Received job with different observeUrl (%s vs %s) for Record ID %s. Flushing current buffer...", job.observeUrl, currentObserveUrl, job.record.ID)
				}
				flushBufferLocked()
				currentObserveUrl = job.observeUrl
				logBuffer = append(logBuffer, job.record)
				if verbose {
					log.Printf("Observability: Started new buffer with Record ID %s (Size: 1)", job.record.ID)
				}
			}
			bufferSize := len(logBuffer)
			bufferMutex.Unlock()

			if bufferSize >= batchSize {
				if verbose {
					log.Printf("Observability: Buffer full (Size: %d >= %d), flushing...", bufferSize, batchSize)
				}
				flushBuffer() // Will handle local save or remote send based on auth status
			}

		case <-ticker.C:
			bufferMutex.Lock()
			if len(logBuffer) > 0 && time.Since(lastSentTime) >= batchInterval {
				if verbose {
					log.Printf("Observability: Batch interval reached (%s), flushing buffer (Size: %d)...", batchInterval, len(logBuffer))
				}
				flushBufferLocked() // Will handle local save or remote send based on auth status
			}
			bufferMutex.Unlock()
		}
	}
}

func flushBuffer() {
	bufferMutex.Lock()
	defer bufferMutex.Unlock()
	flushBufferLocked()
}

func flushBufferLocked() {
	if len(logBuffer) == 0 {
		return
	}

	sendingBuffer := make([]types.AuditRecord, len(logBuffer))
	copy(sendingBuffer, logBuffer)
	sendUrl := currentObserveUrl

	logBuffer = make([]types.AuditRecord, 0, batchSize)
	currentObserveUrl = ""
	lastSentTime = time.Now()

	if verbose {
		log.Printf("Observability: Preparing to flush %d records. Target URL if authenticated: %s", len(sendingBuffer), sendUrl)
	}

	wg.Add(1)
	go func(batch []types.AuditRecord, url string) {
		defer wg.Done()
		sendOrStoreBatch(batch, url) // Renamed function for clarity
	}(sendingBuffer, sendUrl)
}

// sendOrStoreBatch decides whether to send the batch to the remote server or store it locally.
func sendOrStoreBatch(batch []types.AuditRecord, observeUrl string) {
	if len(batch) == 0 {
		return
	}

	authToken, authErr := auth.GetToken()

	if authErr != nil || authToken == "" { // Not authenticated or error fetching token
		// Ensure local DB is initialized (only once)
		localDBInitOnce.Do(func() {
			if verbose {
				log.Println("Observability: First-time local save attempt, initializing local DB...")
			}
			if err := localstore.InitDB(""); err != nil {
				log.Printf("Observability CRITICAL: Failed to initialize local database: %v. Local logs will be lost.", err)
				// If DB init fails, subsequent saves in this execution will also fail the DB check in localstore.SaveBatch
			}
		})

		// Show local logging info message (only once)
		localLogInfoOnce.Do(func() {
			fmt.Fprintln(os.Stderr, color.YellowString("---------------------------------------------------------------------"))
			fmt.Fprintln(os.Stderr, color.CyanString("INFO: Not authenticated. Storing logs locally."))
			fmt.Fprintln(os.Stderr, color.CyanString("      Use 'ithena-cli logs show' to view them."))
			fmt.Fprintln(os.Stderr, color.YellowString("---------------------------------------------------------------------"))
		})

		if verbose {
			log.Printf("Observability: Not authenticated. Saving batch of %d logs locally.", len(batch))
		}
		err := localstore.SaveBatch(batch)
		if err != nil {
			log.Printf("Observability Error: Failed to save batch locally (Size: %d): %v", len(batch), err)
		}
		return // Do not proceed to send to platform
	}

	// Authenticated: Proceed to send to the platform
	if verbose {
		log.Printf("Observability: Authenticated. Sending batch (Size: %d) to %s", len(batch), observeUrl)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	maxRetries := 3
	baseDelay := 1 * time.Second
	var lastHttpErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := baseDelay * time.Duration(1<<(attempt-1))
			if verbose {
				log.Printf("Observability: Retrying batch send (Attempt %d/%d) after %v delay... (Size: %d)", attempt, maxRetries, delay, len(batch))
			}
			time.Sleep(delay)
			// Re-check token in case it expired and was refreshed by another process, or if this is a very long retry cycle.
			// However, for CLI, token is usually long-lived or auth is re-triggered. For simplicity, using initially fetched token.
		}

		payloadBytes, err := json.Marshal(batch)
		if err != nil {
			log.Printf("Observability Error: Failed to marshal batch (Size: %d): %v. Batch not sent.", len(batch), err)
			if len(batch) > 0 {
				log.Printf("  (First Record ID: %s)", batch[0].ID)
			}
			return
		}

		req, err := http.NewRequest("POST", observeUrl, bytes.NewBuffer(payloadBytes))
		if err != nil {
			log.Printf("Observability Error: Failed to create HTTP request for batch (Size: %d): %v. Batch not sent.", len(batch), err)
			return
		}

		req.Header.Set("Authorization", "Bearer "+authToken)
		req.Header.Set("Content-Type", "application/json")

		if verbose {
			log.Printf("Observability: Sending batch HTTP request (Attempt %d, Size: %d)...", attempt, len(batch))
		}

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Observability Error (Attempt %d): HTTP request failed for batch (Size: %d): %v", attempt, len(batch), err)
			lastHttpErr = err
			if attempt == maxRetries {
				log.Printf("Observability Error: Max retries reached for batch send (Size: %d). Last error: %v. Batch not sent.", len(batch), lastHttpErr)
			}
			continue
		}

		respBodyBytes, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if verbose {
				log.Printf("Observability: Batch (Size: %d) sent successfully (Status: %s)", len(batch), resp.Status)
			}
			return
		}

		log.Printf("Observability Error (Attempt %d): Batch send failed (Size: %d) with status %s.", attempt, len(batch), resp.Status)
		if readErr != nil {
			log.Printf("  Additionally, failed to read response body: %v", readErr)
		} else {
			log.Printf("  Response Body: %s", string(respBodyBytes))
		}
		lastHttpErr = fmt.Errorf("batch send failed with status %s", resp.Status)

		if attempt == maxRetries {
			log.Printf("Observability Error: Max retries reached for batch send (Size: %d). Last error: %v. Batch not sent.", len(batch), lastHttpErr)
		}
	}
	if verbose && lastHttpErr != nil {
		log.Printf("Observability: Failed to send batch (Size: %d) after %d retries to %s.", len(batch), maxRetries+1, observeUrl)
	}
}

// SendLog queues an audit record to be processed by the observability worker.
func SendLog(record types.AuditRecord, observeUrl string) {
	// Add proxy version to the record before sending
	// This ensures it's set if the global var was updated after init
	// However, AuditRecord.ProxyVersion is a pointer, so direct assignment works if it's set once globally.
	// For safety, let's ensure it's explicitly assigned if it was nil.
	if record.ProxyVersion == nil {
		// Create a copy to avoid modifying the global var if it's a pointer type
		// For string, direct assignment is fine.
		versionStr := ProxyVersion // Use the package global
		record.ProxyVersion = &versionStr
	}

	// Generate UUID for the log entry if it's not already set
	if record.ID == "" {
		record.ID = uuid.New().String()
	}

	// Ensure timestamp is set (should be request start time, in ISO 8601)
	if record.Timestamp == "" {
		record.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}

	job := logJob{
		record:     record,
		observeUrl: observeUrl,
	}

	// Try to send, but don't block if the channel is full
	select {
	case logChan <- job:
		if verbose {
			log.Printf("Observability: Queued log Record ID: %s", record.ID)
		}
	default:
		// This case should ideally not be hit often if buffer size is adequate and worker is responsive.
		log.Printf("Observability Warning: Log channel full. Dropping log Record ID: %s. Consider increasing buffer or checking worker performance.", record.ID)
	}

	// Send telemetry event for MCP log captured
	telemetryProperties := map[string]interface{}{}
	if record.TargetServerAlias != nil {
		telemetryProperties["target_alias"] = *record.TargetServerAlias
	}
	if record.McpMethod != nil {
		telemetryProperties["mcp_method"] = *record.McpMethod
	}
	if record.ToolName != nil {
		telemetryProperties["tool_name"] = *record.ToolName
	}
	if record.Status != "" {
		telemetryProperties["status"] = record.Status
	}
	// Add more non-sensitive, aggregated properties if needed in the future.
	// Example: record.DurationMs if it's deemed useful and non-identifying.
	// if record.DurationMs != nil {
	// 	telemetryProperties["duration_ms"] = *record.DurationMs
	// }

	telemetry.TrackEvent("mcp_log_captured", telemetryProperties)
}

// RecordRpcCompletion is a utility function to create and send an AuditRecord for a completed JSON-RPC interaction.
func RecordRpcCompletion(
	resp jsonrpc.Response, // The JSON-RPC response object
	duration time.Duration, // Total duration of the call
	alias *string, // Alias for the target server from config
	method *string, // The MCP method called (e.g., "tool/call")
	requestParams interface{}, // The parameters sent in the request
	requestStartTime time.Time, // When the request was initiated
	observeUrl string, // The URL for the observability API endpoint
) {
	status := "success"
	var responsePreview interface{}
	var errorDetails interface{}

	if resp.Error != nil {
		status = "failure"
		errorDetails = resp.Error // Capture the full error object
	} else {
		responsePreview = resp.Result // Capture the result on success
	}

	// Ensure McpMethod and ToolName are correctly extracted or set based on context
	// For generic RPC, method is primary. If it's a tool_call, details might be inside requestParams.
	var toolNameExtract *string
	if method != nil && *method == "tool/call" { // Example condition
		// Attempt to extract tool_name from requestParams if it's a map
		if paramsMap, ok := requestParams.(map[string]interface{}); ok {
			if tn, ok := paramsMap["tool_name"].(string); ok {
				toolNameExtract = &tn
			}
		}
	}
	if method == nil && toolNameExtract != nil {
		// If only tool_name is found (e.g. it's a pure tool call not using mcp_method style)
		// This logic might need refinement based on how you differentiate.
	}

	durationMs := duration.Milliseconds()

	record := types.AuditRecord{
		// ID will be generated by SendLog
		Timestamp:  requestStartTime.UTC().Format(time.RFC3339Nano),
		McpMethod:  method,
		ToolName:   toolNameExtract, // Use extracted if available
		DurationMs: &durationMs,
		Status:     status,
		// ProxyVersion will be set by SendLog
		TargetServerAlias: alias,
		RequestPreview:    requestParams,
		ResponsePreview:   responsePreview,
		ErrorDetails:      errorDetails,
	}

	SendLog(record, observeUrl)
}

// CreateAuditRecordForError is a utility to create an AuditRecord when an error occurs
// even before a full MCP interaction might have completed (e.g., connection error).
func CreateAuditRecordForError(errMsg string, alias *string, method *string, correlationID *string) types.AuditRecord {
	now := time.Now().UTC()
	status := "failure"

	// If a correlationID is provided (e.g., from an incoming request that failed early),
	// use it. Otherwise, generate a new UUID.
	entryID := uuid.New().String()
	if correlationID != nil && *correlationID != "" {
		entryID = *correlationID
	}

	dummyDuration := int64(0) // Error occurred, duration might be minimal or unknown

	return types.AuditRecord{
		ID:        entryID,
		Timestamp: now.Format(time.RFC3339Nano),
		McpMethod: method, // May be nil if error is very early
		// ToolName: // Usually not known for such early errors
		DurationMs: &dummyDuration,
		Status:     status,
		// ProxyVersion: will be set by SendLog
		TargetServerAlias: alias, // May be nil
		// RequestPreview: // Usually not available or relevant for early errors
		// ResponsePreview: // Not applicable
		ErrorDetails: map[string]string{"error": errMsg, "message": "Failed during CLI operation"},
	}
}
