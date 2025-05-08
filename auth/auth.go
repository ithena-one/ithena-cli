package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/zalando/go-keyring"
)

// --- Auth Structs ---
type DeviceAuthResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}
type TokenRequest struct {
	GrantType  string `json:"grant_type"`
	DeviceCode string `json:"device_code"`
}
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in,omitempty"`
}
type TokenErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// --- Keyring Config ---
const keyringServiceName = "ithena-cli"
const keyringTokenKey = "authToken"

// TODO: Make backendBaseUrl configurable if needed by auth
const backendBaseUrl = "https://ithena.one" // Production backend URL

// GetToken retrieves the stored authentication token from the system keyring.
func GetToken() (string, error) {
	token, err := keyring.Get(keyringServiceName, keyringTokenKey)
	if err != nil {
		// Handle specific errors like "not found" if needed,
		// but for now, just return the error.
		return "", fmt.Errorf("failed to retrieve token from keychain: %w", err)
	}
	return token, nil
}

// HandleAuth performs the OAuth device authorization flow.
func HandleAuth() {
	log.Println("Initiating device authorization flow...")

	deviceAuthURL := backendBaseUrl + "/api/cli/auth/device"
	resp, err := http.Post(deviceAuthURL, "application/json", nil)
	if err != nil {
		log.Fatalf("Error initiating device auth: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error response from backend (%d): %s", resp.StatusCode, string(bodyBytes))
		log.Fatalf("Failed to initiate device authorization. Status: %s", resp.Status)
	}

	var authResp DeviceAuthResponse
	err = json.Unmarshal(bodyBytes, &authResp)
	if err != nil {
		log.Fatalf("Error decoding device auth response: %v. Body: %s", err, string(bodyBytes))
	}

	// Define colors
	header := color.New(color.FgYellow, color.Bold)
	urlColor := color.New(color.FgCyan, color.Underline)
	codeColor := color.New(color.FgMagenta, color.Bold)

	header.Printf("\n=== CLI Authorization Required ===\n")
	fmt.Printf("1. Open the following URL in your browser:\n   %s\n", urlColor.Sprint("https://ithena.one/cli-auth/verify"))
	fmt.Printf("2. Enter the following code when prompted:\n   %s\n\n", codeColor.Sprint(authResp.UserCode))
	fmt.Println("Waiting for authorization...")

	// Polling Logic
	tokenURL := backendBaseUrl + "/api/cli/auth/token"
	pollInterval := time.Duration(authResp.Interval) * time.Second
	expiryTime := time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second).Add(10 * time.Second)

	for time.Now().Before(expiryTime) {
		time.Sleep(pollInterval)
		fmt.Print(".")

		tokenReqPayload := TokenRequest{
			GrantType:  "urn:ietf:params:oauth:grant-type:device_code",
			DeviceCode: authResp.DeviceCode,
		}
		jsonPayload, err := json.Marshal(tokenReqPayload)
		if err != nil {
			log.Printf("Error marshaling token request: %v", err)
			continue
		}

		pollResp, err := http.Post(tokenURL, "application/json", bytes.NewBuffer(jsonPayload))
		if err != nil {
			log.Printf("Error polling for token: %v", err)
			continue
		}

		pollBodyBytes, _ := io.ReadAll(pollResp.Body)
		pollResp.Body.Close()

		if pollResp.StatusCode == http.StatusOK {
			var tokenResp TokenResponse
			err = json.Unmarshal(pollBodyBytes, &tokenResp)
			if err != nil {
				log.Printf("Error decoding token response: %v. Body: %s", err, string(pollBodyBytes))
				log.Fatalf("Failed to decode successful token response.")
			}
			fmt.Println("\nAuthorization successful!")

			err = keyring.Set(keyringServiceName, keyringTokenKey, tokenResp.AccessToken)
			if err != nil {
				log.Printf("Warning: Failed to store token securely in keychain: %v", err)
				fmt.Println("Failed to save token to keychain. You may need to authenticate again later.")
			} else {
				log.Println("Access token securely stored.")
			}

			log.Printf("Received Access Token: [REDACTED] (Type: %s)", tokenResp.TokenType)
			fmt.Println("Authentication complete.")
			return
		}

		if pollResp.StatusCode == http.StatusBadRequest {
			var errResp TokenErrorResponse
			err = json.Unmarshal(pollBodyBytes, &errResp)
			if err != nil {
				log.Printf("Error decoding error response: %v. Body: %s", err, string(pollBodyBytes))
				continue
			}

			switch errResp.Error {
			case "authorization_pending":
				continue
			case "slow_down":
				log.Println("Server requested to slow down polling...")
				pollInterval += 5 * time.Second
				continue
			case "access_denied":
				fmt.Println("\nAuthorization request denied by user.")
				os.Exit(1)
			case "expired_token":
				fmt.Println("\nAuthorization request expired.")
				os.Exit(1)
			case "invalid_grant":
				fmt.Println("\nAuthorization failed (invalid grant/code). Please try `auth` again.")
				os.Exit(1)
			default:
				log.Printf("Received unexpected error during polling: %s (%s)", errResp.Error, errResp.ErrorDescription)
				os.Exit(1)
			}
		} else {
			log.Printf("Unexpected status code during polling (%d): %s", pollResp.StatusCode, string(pollBodyBytes))
			os.Exit(1)
		}
	}

	fmt.Println("\nAuthorization timed out.")
	os.Exit(1)
}

// HandleAuthStatusCommand checks and displays the current authentication status.
func HandleAuthStatusCommand() {
	token, err := GetToken()
	if err != nil || token == "" {
		// Consider specific error checking if GetToken can return different error types
		// For now, any error or empty token means not authenticated.
		// We can use keyring.ErrNotFound if we want to be specific about the token not existing vs other errors.
		if err == keyring.ErrNotFound {
			fmt.Println("Not authenticated. No token found in keychain.")
		} else if err != nil {
			log.Printf("Error checking authentication status: %v", err)
			fmt.Println("Not authenticated. (Error accessing token)")
		} else {
			fmt.Println("Not authenticated. Token is empty.") // Should ideally not happen if GetToken returns err on empty
		}
		return
	}
	// At this point, token is not empty and err is nil
	fmt.Println("Authenticated.")
	// Optionally: Decode JWT token here to show expiry or other non-sensitive info
	// but that would require a JWT parsing library.
}

// HandleDeauthCommand removes the stored authentication token.
func HandleDeauthCommand() {
	// First, check if a token exists to provide better user feedback
	_, err := GetToken()
	if err == keyring.ErrNotFound {
		fmt.Println("Not authenticated. No active session to log out from.")
		return
	} else if err != nil && err != keyring.ErrNotFound { // some other error trying to get the token
		log.Printf("Error checking token before deauthentication: %v", err)
		fmt.Println("Could not verify current session status, but will attempt to remove token.")
		// Proceed to attempt deletion anyway
	}

	err = keyring.Delete(keyringServiceName, keyringTokenKey)
	if err != nil {
		if err == keyring.ErrNotFound { // Should be caught by the check above, but good to be safe
			fmt.Println("Not authenticated. No active session to log out from.")
		} else {
			log.Printf("Error removing token from keychain: %v", err)
			fmt.Println("Failed to log out. Could not remove token from keychain.")
		}
		return
	}
	fmt.Println("Successfully logged out.")
	log.Println("Authentication token removed from keychain.")
} 