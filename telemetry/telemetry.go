package telemetry

import (
	"log" // Import the log package
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time" // Import time for batching interval

	"github.com/google/uuid"
	"github.com/posthog/posthog-go"
	"github.com/ithena-one/Ithena/packages/cli/version"
)

const (
	telemetryIDFileName        = "telemetry_id.txt"
	defaultTelemetryBatchSize  = 5                // More aggressive batching for CLI
	defaultTelemetryInterval   = 5 * time.Second  // More aggressive interval for CLI
)

var (
	anonymousID   string
	posthogClient posthog.Client
	once          sync.Once
	mu            sync.Mutex
	optOut        bool
	isInitialized bool
	verbose       bool
)

// SetVerbose enables or disables verbose logging for the telemetry package.
func SetVerbose(v bool) {
	mu.Lock()
	defer mu.Unlock()
	verbose = v
}

// Init initializes the telemetry module.
// It loads or generates an anonymous machine ID and initializes the PostHog client.
// Telemetry will be disabled if the ITHENA_TELEMETRY_OPTOUT environment variable is set to "true"
// or if the ITHENA_POSTHOG_KEY is not provided.
func Init() {
	once.Do(func() {
		if os.Getenv("ITHENA_TELEMETRY_OPTOUT") == "true" {
			optOut = true
			isInitialized = true // Mark as initialized even if opted out, to prevent re-init
			return
		}

		apiKey := os.Getenv("ITHENA_POSTHOG_KEY")
		apiEndpoint := os.Getenv("ITHENA_POSTHOG_ENDPOINT")
		if apiEndpoint == "" {
			apiEndpoint = posthog.DefaultEndpoint // Use default if not set
		}

		if apiKey == "" {
			// No API key, telemetry remains disabled but considered initialized
			// This allows users building from source to not have telemetry by default
			isInitialized = true
			return
		}

		var err error
		anonymousID, err = loadOrGenerateAnonymousID()
		if err != nil {
			if verbose {
				log.Printf("Telemetry: Failed to load/generate anonymous ID: %v. Telemetry will be disabled.", err)
			}
			return
		}

		config := posthog.Config{
			Endpoint:  apiEndpoint,
			BatchSize: defaultTelemetryBatchSize,
			Interval:  defaultTelemetryInterval,
		}
		if verbose {
			config.Verbose = true // Enable PostHog client's internal verbose logging if CLI verbose is on
		}

		client, err := posthog.NewWithConfig(apiKey, config)
		if err != nil {
			if verbose {
				log.Printf("Telemetry: Failed to initialize PostHog client: %v. Telemetry will be disabled.", err)
			}
			return
		}
		posthogClient = client
		isInitialized = true // Mark as successfully initialized
	})
}

func getIthenaConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".ithena"), nil
}

func loadOrGenerateAnonymousID() (string, error) {
	configDir, err := getIthenaConfigDir()
	if err != nil {
		return "", err
	}

	idFilePath := filepath.Join(configDir, telemetryIDFileName)

	idBytes, err := os.ReadFile(idFilePath)
	if err == nil {
		return string(idBytes), nil
	}

	if !os.IsNotExist(err) {
		return "", err // Other error reading file
	}

	// File does not exist, generate new ID
	newID := uuid.New().String()

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0750); err != nil { // rwxr-x---
		// Check if the error is because the directory already exists (race condition)
		if _, statErr := os.Stat(configDir); !os.IsNotExist(statErr) && statErr != nil && !os.IsExist(statErr) {
			// If it's not a "NotExist" or "IsExist" error, then it's a real problem.
			// However, if it's IsExist, it means another process created it, which is fine.
			// If it's NotExist, it means MkdirAll failed for some other reason.
			if !os.IsExist(statErr) {
				return "", err
			}
		} else if os.IsNotExist(statErr) { // MkdirAll failed and dir still doesn't exist
             return "", err
        }
	}


	err = os.WriteFile(idFilePath, []byte(newID), 0600) // rw-------
	if err != nil {
		return "", err
	}
	return newID, nil
}

// TrackEvent sends an event to PostHog.
// It ensures that Init() has been called.
func TrackEvent(eventName string, properties map[string]interface{}) {
	mu.Lock()
	defer mu.Unlock()

	if !isInitialized {
		Init() // Ensure initialization if called directly before explicit Init
	}

	if optOut || posthogClient == nil || anonymousID == "" {
		return // Telemetry is opted out, not configured, or ID is missing
	}

	// Add common properties
	if properties == nil {
		properties = make(map[string]interface{})
	}
	properties["anonymous_machine_id"] = anonymousID
	properties["cli_version"] = version.Version
	properties["os_type"] = runtime.GOOS
	properties["arch_type"] = runtime.GOARCH

	err := posthogClient.Enqueue(posthog.Capture{
		DistinctId: anonymousID,
		Event:      eventName,
		Properties: properties,
	})
	if err != nil && verbose {
		log.Printf("Telemetry: Error enqueuing event '%s': %v", eventName, err)
	}
}

// Shutdown flushes any queued events to PostHog.
// This should be called before the CLI exits.
func Shutdown() {
	mu.Lock()
	defer mu.Unlock()
	if posthogClient != nil && !optOut {
		err := posthogClient.Close()
		if err != nil && verbose {
			log.Printf("Telemetry: Error closing PostHog client: %v", err)
		}
	}
}

// GetAnonymousID returns the anonymous machine ID.
// It ensures that Init() has been called.
func GetAnonymousID() string {
	if !isInitialized {
		Init()
	}
	return anonymousID
}

// IsOptOut returns true if telemetry is opted out.
func IsOptOut() bool {
	if !isInitialized {
		Init()
	}
	return optOut
}

// IsEnabled returns true if telemetry is configured and not opted out.
func IsEnabled() bool {
	mu.Lock()
	defer mu.Unlock()
	if !isInitialized {
		Init()
	}
	return !optOut && posthogClient != nil && anonymousID != ""
}

// For testing purposes or if properties need to be dynamically set on the client
func GetPosthogClient() posthog.Client {
	return posthogClient
}
