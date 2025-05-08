package placeholder

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/zalando/go-keyring"
)

// Regular expression to find placeholders like {{type:value}}
var placeholderRegex = regexp.MustCompile(`{{\s*(env|keyring|file)\s*:\s*([^}]+)\s*}}`)

// ResolvePlaceholders takes a map representing environment variables (potentially with placeholders)
// and returns a new map with placeholders resolved.
func ResolvePlaceholders(envMap map[string]string) (map[string]string, error) {
	resolvedEnv := make(map[string]string)
	var firstError error // Variable to capture the first error encountered

	for key, value := range envMap {
		resolvedValue, err := resolveValue(value)
		if err != nil {
			// Capture the first error and add context
			if firstError == nil {
				firstError = fmt.Errorf("failed to resolve placeholder for key '%s' (value: '%s'): %w", key, value, err)
			}
			// Store the partially resolved or error-marked value anyway
			resolvedEnv[key] = resolvedValue // Store the value with error markers
		} else {
			resolvedEnv[key] = resolvedValue
		}
	}

	// Return the first error encountered during resolution, if any
	return resolvedEnv, firstError
}

// resolveValue processes a single string value, resolving any placeholders within it.
// It returns the potentially modified string and an error if resolution fails.
func resolveValue(value string) (string, error) {
	var firstResolutionError error

	resolved := placeholderRegex.ReplaceAllStringFunc(value, func(match string) string {
		// If an error already occurred in this string, don't process further placeholders
		if firstResolutionError != nil {
			return match
		}

		parts := placeholderRegex.FindStringSubmatch(match)
		if len(parts) != 3 {
			firstResolutionError = fmt.Errorf("invalid placeholder format: %s", match)
			return match // Return original match on error
		}

		placeholderType := strings.TrimSpace(parts[1])
		placeholderValue := strings.TrimSpace(parts[2])

		switch placeholderType {
		case "env":
			envVal, found := os.LookupEnv(placeholderValue)
			if !found {
				firstResolutionError = fmt.Errorf("environment variable '%s' not found", placeholderValue)
				return match
			}
			return envVal
		case "keyring":
			krParts := strings.SplitN(placeholderValue, ":", 2)
			if len(krParts) != 2 {
				firstResolutionError = fmt.Errorf("invalid keyring format '%s', expected 'service:account'", placeholderValue)
				return match
			}
			service := krParts[0]
			user := krParts[1]
			secret, err := keyring.Get(service, user)
			if err != nil {
				firstResolutionError = fmt.Errorf("keyring error for '%s:%s': %w", service, user, err)
				return match
			}
			return secret
		case "file":
			contentBytes, err := os.ReadFile(placeholderValue)
			if err != nil {
				firstResolutionError = fmt.Errorf("failed to read file '%s': %w", placeholderValue, err)
				return match
			}
			return strings.TrimSpace(string(contentBytes))
		default:
			// Should not happen with the current regex
			firstResolutionError = fmt.Errorf("unknown placeholder type '%s'", placeholderType)
			return match
		}
	})

	// Return the processed string and the first error encountered during ReplaceAllStringFunc
	return resolved, firstResolutionError
} 