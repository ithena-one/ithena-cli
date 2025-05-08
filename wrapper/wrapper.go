package wrapper

import (
	"bufio"
	// "bytes" // Unused
	"encoding/json"
	"fmt"
	// "github.com/google/uuid" // Unused
	"github.com/ithena-one/Ithena/packages/cli/jsonrpc"
	"github.com/ithena-one/Ithena/packages/cli/observability"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// verbose is a package-level variable to control logging within the wrapper
// It needs to be set from main.go
var verbose bool

// SetVerbose enables or disables verbose logging for the wrapper package.
func SetVerbose(v bool) {
	verbose = v
}

// Run executes the wrapper logic based on resolved profile config.
func Run(command string, args []string, resolvedEnv map[string]string, alias string, observeUrl string) {
	// Use profile alias if provided, otherwise default logging
	var aliasPtr *string
	if alias != "" {
		aliasPtr = &alias
	} else {
		aliasPtr = nil // Or set a default alias?
	}

	if verbose { log.Printf("Wrapper: Starting for command: %s %v (Alias: %s, ObserveURL: %s)", command, args, alias, observeUrl) }

	cmd := exec.Command(command, args...)

	// Set environment variables: start with current process env,
	// then override/add with resolvedEnv from profile.
	currentEnv := os.Environ()
	envMap := make(map[string]string)
	for _, envVar := range currentEnv {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	if verbose { log.Printf("Wrapper: Initial environment contains %d variables.", len(envMap)) }
	// Apply resolved environment variables from profile, overriding existing ones
	for key, value := range resolvedEnv {
		envMap[key] = value
	}
	// Convert back to slice format required by exec.Command
	var finalEnv []string
	for key, value := range envMap {
		finalEnv = append(finalEnv, key+"="+value)
	}
	cmd.Env = finalEnv
	if verbose { log.Printf("Wrapper: Final environment for backend has %d variables (profile overrides applied).", len(finalEnv)) }

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		logErrorAndExit(fmt.Sprintf("Failed to create stdin pipe for '%s'", command), aliasPtr, nil, observeUrl, nil, err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		logErrorAndExit(fmt.Sprintf("Failed to create stdout pipe for '%s'", command), aliasPtr, nil, observeUrl, nil, err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		logErrorAndExit(fmt.Sprintf("Failed to create stderr pipe for '%s'", command), aliasPtr, nil, observeUrl, nil, err)
	}

	// Start the command
	if verbose { log.Printf("Wrapper: Starting backend command '%s'...", command) }
	if err := cmd.Start(); err != nil {
		logErrorAndExit(fmt.Sprintf("Failed to start command '%s'", command), aliasPtr, nil, observeUrl, nil, err)
	}
	if verbose { log.Printf("Wrapper: Backend command started (PID: %d)", cmd.Process.Pid) }

	var wg sync.WaitGroup
	requestStore := newRequestStore()
	if verbose { log.Printf("Wrapper: Initialized request store and wait group.") }

	// Goroutine 1: Proxy ithena-cli stdin -> backend stdin & Store Request Info
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if verbose { log.Println("Wrapper: Goroutine 1 (stdin proxy) closing backend stdin pipe.") }
			stdinPipe.Close() // Close stdin when copying finishes
		}()
		if verbose { log.Println("Wrapper: Goroutine 1 (stdin proxy) started.") }
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			lineBytes := scanner.Bytes()
			startTime := time.Now() // Record start time BEFORE writing/parsing

			// Write to backend stdin FIRST
			if _, err := stdinPipe.Write(append(lineBytes, '\n')); err != nil {
				log.Printf("Error writing to backend stdin: %v", err)
				return // Stop proxying if write fails
			}

			// Attempt to parse for logging/correlation
			var req jsonrpc.Request
			if err := json.Unmarshal(lineBytes, &req); err == nil {
				if req.ID != nil {
					// Store request info for later correlation in the response handler
					requestStore.Store(req.ID, req.Method, startTime, req.Params)
					if verbose { log.Printf("Wrapper: Stored request ID %v (Method: %s)", req.ID, req.Method) }
					// DO NOT send request log here anymore
				} else {
					if verbose { log.Printf("Wrapper: Received notification on stdin: Method=%s", req.Method) }
				}
			} else {
				if verbose { log.Printf("Wrapper: Received non-JSON line on stdin: %s", string(lineBytes)) }
			}
		}
		if scanner.Err() != nil {
			log.Printf("Wrapper: Error reading from wrapper stdin: %v", scanner.Err())
		}
		if verbose { log.Println("Wrapper: Goroutine 1 (stdin proxy) finished reading.") }
	}()

	// Goroutine 2: Proxy backend stdout -> ithena-cli stdout & Log Completion
	wg.Add(1)
	go func() {
		defer wg.Done()
		if verbose { log.Println("Wrapper: Goroutine 2 (stdout proxy) started.") }
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			lineBytes := scanner.Bytes()
			// Write to wrapper stdout FIRST
			if _, err := os.Stdout.Write(append(lineBytes, '\n')); err != nil {
				log.Printf("Error writing to wrapper stdout: %v", err)
			}

			// Attempt to parse for logging
			var resp jsonrpc.Response
			if err := json.Unmarshal(lineBytes, &resp); err == nil {
				if resp.ID != nil {
					methodPtr, startTime, requestParams, found := requestStore.Retrieve(resp.ID)
					var duration time.Duration = 0

					if found {
						duration = time.Since(startTime)
						// Call the new function to handle consolidated logging
						observability.RecordRpcCompletion(resp, duration, aliasPtr, methodPtr, requestParams, startTime, observeUrl)
						if verbose { log.Printf("Wrapper: Recorded completion for ID %v (Method: %s, Duration: %s)", resp.ID, *methodPtr, duration) }
						// DO NOT send response log here anymore
					} else {
						log.Printf("Wrapper: Received RPC response with unknown/duplicate ID: %v. Cannot correlate.", resp.ID)
						// Optionally log an error record if correlation fails?
						// observability.SendLog(observability.CreateAuditRecordForError(...), observeUrl)
					}
				} else {
					if verbose { log.Printf("Wrapper: Received notification on backend stdout: %s", string(lineBytes)) }
				}
			} else {
				if verbose { log.Printf("Wrapper: Received non-JSON line on backend stdout: %s", string(lineBytes)) }
			}
		}
		if scanner.Err() != nil {
			log.Printf("Wrapper: Error reading from backend stdout: %v", scanner.Err())
		}
		if verbose { log.Println("Wrapper: Goroutine 2 (stdout proxy) finished reading.") }
	}()

	// Goroutine 3: Proxy backend stderr -> ithena-cli stderr
	wg.Add(1)
	go func() {
		defer wg.Done()
		if verbose { log.Println("Wrapper: Goroutine 3 (stderr proxy) started.") }
		if _, err := io.Copy(os.Stderr, stderrPipe); err != nil {
			log.Printf("Wrapper: Error copying backend stderr: %v", err)
		}
		if verbose { log.Println("Wrapper: Goroutine 3 (stderr proxy) finished copying.") }
	}()

	// Wait for all proxying goroutines to finish (indicates streams closed)
	if verbose { log.Println("Wrapper: Waiting for IO goroutines to complete...") }
	wg.Wait()
	if verbose { log.Println("Wrapper: IO goroutines finished.") }

	// Wait for the command to exit and capture exit code
	if verbose { log.Println("Wrapper: Waiting for backend command to exit...") }
	err = cmd.Wait()
	status := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			status = exitErr.ExitCode()
			errMsg := fmt.Sprintf("Backend command '%s' exited with non-zero status %d", command, status)
			log.Printf("Wrapper Error: %s", errMsg)
			// Log observability for non-zero exit (async)
			observability.SendLog(observability.CreateAuditRecordForError(errMsg, aliasPtr, nil, nil), observeUrl)
			observability.ShutdownObservability() // Ensure logs are flushed before exit
			os.Exit(status) // Exit wrapper with same code
		} else {
			// Error not related to exit code (e.g., Wait failed, command not found)
			logErrorAndExit(fmt.Sprintf("Error waiting for backend command '%s'", command), aliasPtr, nil, observeUrl, nil, err)
		}
	} else {
		if verbose { log.Printf("Wrapper: Backend command '%s' finished successfully (status 0).", command) }
	}
	// Exit with backend's status code (0 if successful)
	if verbose { log.Println("Wrapper: Shutting down observability and exiting with status", status) }
	observability.ShutdownObservability()
	os.Exit(status)
}

// logErrorAndExit logs a fatal wrapper error and exits.
// It attempts to send an observability log and ensures shutdown before exiting.
// The original error `origErr` is included for more context.
func logErrorAndExit(baseMsg string, alias *string, method *string, observeUrl string, correlationID *string, origErr error) {
	errMsg := baseMsg
	if origErr != nil {
		errMsg = fmt.Sprintf("%s: %v", baseMsg, origErr)
	}
	log.Printf("Fatal Wrapper Error: %s", errMsg) // Log the detailed error
	// Attempt to log observability using the base message for brevity in observability system
	observability.SendLog(observability.CreateAuditRecordForError(baseMsg, alias, method, correlationID), observeUrl)
	// Ensure logs are flushed before exiting
	observability.ShutdownObservability()
	os.Exit(1) // Exit with status 1 for fatal wrapper errors
}

// --- Request Store for correlating requests/responses ---

type requestInfo struct {
	method    string
	startTime time.Time
	params    interface{} // Store the request params
}

type requestStore struct {
	mu    sync.Mutex
	store map[interface{}]requestInfo // Key is the JSON-RPC request ID
}

func newRequestStore() *requestStore {
	return &requestStore{
		store: make(map[interface{}]requestInfo),
	}
}

// Store saves the request details needed for response correlation.
func (rs *requestStore) Store(id interface{}, method string, startTime time.Time, params interface{}) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	// Convert ID to string for reliable map key if it's a number
	key := idToString(id)
	rs.store[key] = requestInfo{
		method:    method,
		startTime: startTime,
		params:    params,
	}
}

// Retrieve fetches and removes the request info using the JSON-RPC request ID.
func (rs *requestStore) Retrieve(id interface{}) (method *string, startTime time.Time, params interface{}, found bool) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	// Convert ID to string for lookup
	key := idToString(id)
	info, found := rs.store[key]
	if found {
		delete(rs.store, key) // Remove after retrieval
		// Return a pointer to the method string
		methodCopy := info.method
		return &methodCopy, info.startTime, info.params, true
	}
	// Return zero values if not found
	return nil, time.Time{}, nil, false
}

// idToString converts JSON-RPC ID (number or string) to a string for map keys.
func idToString(id interface{}) string {
	switch v := id.(type) {
	case string:
		return v
	case float64: // encoding/json uses float64 for numbers
		// Format float64 to string without unnecessary decimals
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		// Fallback for unexpected types
		return fmt.Sprintf("%v", v)
	}
} 