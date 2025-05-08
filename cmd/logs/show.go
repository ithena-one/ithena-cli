package logs

import (
	"bufio" // For reading user input
	"fmt"
	"log"
	"os" // For os.Remove
	"strings" // For trimming input

	"github.com/ithena-one/Ithena/packages/cli/localstore"
	"github.com/ithena-one/Ithena/packages/cli/webui" // Import webui package
)

var verbose bool
// const defaultWebUIPort = 8675 // Port is now passed as an argument

// HandleLogsShowCommand handles the 'ithena-cli logs show' command.
func HandleLogsShowCommand(verbose bool, port int) { // Add port parameter
	if verbose {
		log.Printf("Executing 'logs show' command for port %d...", port)
	}

	localstore.SetVerbose(verbose) 
	webui.SetVerbose(verbose) // Pass verbosity to webui as well

	err := localstore.InitDB("")
	if err != nil {
		log.Fatalf("Error initializing local database for 'logs show': %v", err)
	}

	if verbose {
		log.Println("Local database initialized successfully for 'logs show'.")
	}

	// Get the actual path for informational purposes, though webui doesn't need it directly
	dbPath, pathErr := localstore.GetDefaultLogStorePathForInfo() // We'll add this helper to localstore
	if pathErr != nil {
		log.Printf("Info: Could not determine local log store path: %v", pathErr)
		dbPath = "(Could not determine path)"
	}
	fmt.Printf("Attempting to start local log viewer UI. Access it at http://localhost:%d\n", port)
	fmt.Printf("Local logs are being read from: %s\n", dbPath)
	fmt.Println("Press Ctrl+C to stop the server.")

	webui.StartServer(port) // Use the passed port
}

// HandleLogsClearCommand handles the 'ithena-cli logs clear' command.
func HandleLogsClearCommand(verbose bool) {
	if verbose {
		log.Println("Executing 'logs clear' command...")
	}

	dbPath, err := localstore.GetDefaultLogStorePathForInfo()
	if err != nil {
		log.Fatalf("Error determining local log store path: %v", err)
	}

	fmt.Printf("This will delete all locally stored logs at: %s\n", dbPath)
	fmt.Print("Are you sure you want to continue? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input != "y" && input != "yes" {
		fmt.Println("Operation cancelled.")
		return
	}

	// Close the database connection if it's open, before deleting the file.
	if localstore.DB != nil {
		err := localstore.DB.Close()
		if err != nil {
			// Log the error but proceed with attempting to delete the file.
			log.Printf("Warning: Error closing local database: %v. Attempting to delete file anyway.", err)
		}
		localstore.DB = nil // Set to nil so it gets re-initialized if needed later
	}

	err = os.Remove(dbPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No local logs file found to delete.")
		} else {
			log.Fatalf("Error deleting local logs file %s: %v", dbPath, err)
		}
	} else {
		fmt.Printf("Successfully deleted local logs file: %s\n", dbPath)
	}

	if verbose {
		log.Println("'logs clear' command finished.")
	}
} 