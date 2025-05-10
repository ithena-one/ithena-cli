// packages/cli/main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter" 

	"github.com/fatih/color"

	"github.com/ithena-one/Ithena/packages/cli/auth"
	"github.com/ithena-one/Ithena/packages/cli/config"
	"github.com/ithena-one/Ithena/packages/cli/observability"
	"github.com/ithena-one/Ithena/packages/cli/placeholder"
	"github.com/ithena-one/Ithena/packages/cli/wrapper"
	"github.com/ithena-one/Ithena/packages/cli/cmd/logs" 
)


var (
	version string
	commit  string
	date    string

	// Old flags removed
	observeUrl string

	// New Wrapper mode flags (Profile-based)
	wrapperProfile    string
	wrapperConfigFile string

	// Default values
	defaultObserveUrl        = "https://ithena.one/api/v1/observe"
	defaultWrapperConfigFile = "./.ithena-wrappers.yaml" // Default config file name

	// Verbosity flag
	verbose bool

	// Version flag
	showVersion bool

	// New logs command flags
	logsShowPort int // Flag for 'logs show --port'
)

// Command-level flag sets, accessible globally within the main package for printUsage
var authCmd *flag.FlagSet
var logsCmd *flag.FlagSet

// --- main function ---
func main() {
	log.SetFlags(0) // Remove date, time, and file/line number prefixes

	// Initialize observability system (starts worker goroutine)
	observability.InitObservability()
	// Ensure observability worker is shut down gracefully on exit
	defer observability.ShutdownObservability()

	// === Subcommand definitions ===
	authCmd = flag.NewFlagSet("auth", flag.ExitOnError)
	authCmd.Usage = func() { printCommandUsage(authCmd, "auth", "Manage authentication. Available subcommands: login, status, deauth (logout)") }

	logsCmd = flag.NewFlagSet("logs", flag.ExitOnError)
	logsCmd.IntVar(&logsShowPort, "port", 8675, "Port for the local logs web UI (only for 'show' subcommand)")
	logsCmd.Usage = func() { printCommandUsage(logsCmd, "logs", "Interact with local logs. Available subcommands: show, clear") }

	// Global flags
	flag.StringVar(&wrapperProfile, "wrapper-profile", "", "Name of the wrapper profile to use from the config file")
	flag.StringVar(&wrapperConfigFile, "wrapper-config-file", defaultWrapperConfigFile, "Path to the wrapper configuration file (YAML)")
	flag.StringVar(&observeUrl, "observe-url", defaultObserveUrl, "URL for the observability API endpoint")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging output")
	flag.BoolVar(&showVersion, "version", false, "Print version information and exit")
	flag.Usage = printMainUsage

	flag.Parse()

	if showVersion {
		// Note: The 'version', 'commit', and 'date' variables are expected to be set by ldflags during build
		fmt.Printf("Ithena CLI version: %s\n", version)
		if commit != "" {
			fmt.Printf("Commit: %s\n", commit)
		}
		if date != "" {
			fmt.Printf("Build Date: %s\n", date)
		}
		os.Exit(0)
	}

	observability.SetVerbose(verbose)
	wrapper.SetVerbose(verbose)
	// localstore.SetVerbose(verbose) // Will be set if localstore is initialized

	args := flag.Args() // Get all non-flag arguments

	if len(args) > 0 {
		command := args[0]
		switch command {
		case "auth":
			authCmd.Parse(args[1:]) // Pass remaining args to subcommand
			if authCmd.NArg() > 0 {
				authSubCommand := authCmd.Arg(0)
				switch authSubCommand {
				case "login": // Assuming 'login' is the default auth action if a subcommand is needed
					if verbose { log.Println("Handling 'auth login' subcommand...") }
					auth.HandleAuth() // This is the original behavior
				case "status":
					if verbose { log.Println("Handling 'auth status' subcommand...") }
					auth.HandleAuthStatusCommand()
				case "deauth", "logout": // Allow 'logout' as an alias for 'deauth'
					if verbose { log.Println("Handling 'auth deauth/logout' subcommand...") }
					auth.HandleDeauthCommand()
				default:
					fmt.Fprintf(os.Stderr, "Error: Unknown subcommand for 'auth': %s\n", authSubCommand)
					authCmd.Usage()
					exitWithError(1)
				}
			} else {
				// Default action for 'auth' (no subcommand given) is to initiate login
				if verbose { log.Println("Handling 'auth' subcommand (defaulting to login)...") }
				auth.HandleAuth()
			}
			return
		case "logs":
			logsCmd.Parse(args[1:]) // Pass remaining args to subcommand
			if logsCmd.NArg() > 0 {
				logsSubCommand := logsCmd.Arg(0)
				switch logsSubCommand {
				case "show":
					if verbose { log.Printf("Handling 'logs show' subcommand with port: %d", logsShowPort) }
					// Pass the version, commit, and date variables to the logs show command
					// Note: 'version' variable is populated by ldflags during build.
					logs.HandleLogsShowCommand(verbose, logsShowPort, version)
					return
				case "clear":
					if verbose { log.Println("Handling 'logs clear' subcommand...") }
					logs.HandleLogsClearCommand(verbose)
					return
				default:
					fmt.Fprintf(os.Stderr, "Error: Unknown subcommand for 'logs': %s\n", logsSubCommand)
					logsCmd.Usage()
					exitWithError(1)
				}
			} else {
				logsCmd.Usage() // Show help for 'logs' if no subcommand given
				return
			}
		default:
			// Not 'auth' or 'logs'. This is a command to wrap directly.
			if wrapperProfile != "" {
				fmt.Fprintf(os.Stderr,
					"Error: Cannot specify a direct command ('%s') when --wrapper-profile ('%s') is also provided.\n"+
						"Please either provide a direct command to wrap, or use a wrapper profile, but not both.\n",
					command, wrapperProfile)
				printMainUsage()
				exitWithError(1)
			}

			commandToWrap := command
			commandArgs := []string{}
			if len(args) > 1 {
				commandArgs = args[1:]
			}
			if verbose {
				log.Printf("Wrapper mode: Wrapping direct command. Command: '%s', Args: '%v'", commandToWrap, commandArgs)
			}
			// For direct wrapping, use empty env map and command itself as alias.
			// This means the wrapped command won't inherit the parent environment directly through this map.
			// If os.Environ() inheritance is desired, this part needs to be adjusted.
			wrapper.Run(commandToWrap, commandArgs, make(map[string]string), commandToWrap, observeUrl)
			return
		}
	} else {
		// No positional arguments were given (e.g., `ithena-cli --wrapper-profile foo` or just `ithena-cli`)
		if wrapperProfile == "" {
			fmt.Fprintln(os.Stderr, "Error: No command or --wrapper-profile specified. Run 'ithena-cli --help' for usage.")
			printMainUsage()
			exitWithError(1)
		}

		// Wrapper mode with profile
		if verbose { log.Printf("Wrapper mode: Using profile '%s' from config '%s'", wrapperProfile, wrapperConfigFile) }
		wrapperConf, err := config.LoadWrapperConfig(wrapperConfigFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading wrapper config '%s': %v\n", wrapperConfigFile, err)
			exitWithError(1)
		}
		profile, found := wrapperConf.Wrappers[wrapperProfile]
		if !found {
			fmt.Fprintf(os.Stderr, "Error: Wrapper profile '%s' not found in config file '%s'\n", wrapperProfile, wrapperConfigFile)
			exitWithError(1)
		}
		resolvedEnv, err := placeholder.ResolvePlaceholders(profile.Env)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving environment variable placeholders for profile '%s': %v\n", wrapperProfile, err)
			exitWithError(1)
		}
		wrapper.Run(profile.Command, profile.Args, resolvedEnv, profile.Alias, observeUrl)
		return
	}
}

// exitWithError ensures observability shutdown before exiting with an error code.
func exitWithError(code int) {
	observability.ShutdownObservability() // Call shutdown explicitly
	os.Exit(code)
}

// printMainUsage prints the main help message for the CLI.
func printMainUsage() {
	header := color.New(color.FgYellow, color.Bold)
	commandStyle := color.New(color.FgGreen)
	executableName := os.Args[0]

	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  %s [command] [flags]\n", executableName)
	fmt.Fprintf(os.Stderr, "  %s <your_command_to_wrap> [args...] [global_flags]\n", executableName)
	fmt.Fprintf(os.Stderr, "  %s --wrapper-profile <profile_name> [global_flags]\n\n", executableName)

	w := tabwriter.NewWriter(os.Stderr, 0, 0, 2, ' ', 0)

	header.Fprintln(w, "Description:")
	fmt.Fprintln(w, "  ithena-cli can operate in several modes:")
	fmt.Fprintln(w, "  1. Manage authentication ('auth').")
	fmt.Fprintln(w, "  2. Manage and view local logs ('logs show', 'logs clear').")
	fmt.Fprintln(w, "  3. Wrap a pre-configured command using a profile (via '--wrapper-profile').")
	fmt.Fprintln(w, "  4. Directly wrap and observe an arbitrary command by specifying it directly.")
	fmt.Fprintln(w)

	header.Fprintln(w, "Available Commands:")
	fmt.Fprintf(w, "  %s\t\tManage authentication. Use 'ithena-cli auth <subcommand> --help' for details.\n", commandStyle.Sprint("auth"))
	fmt.Fprintf(w, "  %s\t\tInteract with local logs. Use 'ithena-cli logs <subcommand> --help' for details.\n", commandStyle.Sprint("logs"))
	fmt.Fprintln(w)

	header.Fprintln(w, "Global Flags (applicable to wrapper modes and some commands):")
	globalFlags := flag.NewFlagSet("global", flag.ContinueOnError) // Temporary set to iterate
	// Re-declare global flags here for iteration purposes ONLY, do not assign to the actual variables.
	// Their actual values are parsed from flag.CommandLine.
	var tempWrapperProfile, tempWrapperConfigFile, tempObserveUrl string
	var tempVerbose, tempShowVersion bool
	globalFlags.StringVar(&tempWrapperProfile, "wrapper-profile", "", "Name of the wrapper profile to use from the config file")
	globalFlags.StringVar(&tempWrapperConfigFile, "wrapper-config-file", defaultWrapperConfigFile, "Path to the wrapper configuration file (YAML)")
	globalFlags.StringVar(&tempObserveUrl, "observe-url", defaultObserveUrl, "URL for the observability API endpoint")
	globalFlags.BoolVar(&tempVerbose, "verbose", false, "Enable verbose logging output")
	globalFlags.BoolVar(&tempShowVersion, "version", false, "Print version information and exit") // Added for help text
	
	globalFlags.VisitAll(func(f *flag.Flag) {
		// Fetch the actual global flag from the main flag set to get its properties
		actualFlag := flag.Lookup(f.Name)
		if actualFlag != nil {
			printFlag(w, actualFlag)
		}
	})
	fmt.Fprintln(w)

	header.Fprintln(w, "Use 'ithena-cli [command] --help' for more information about a command.")
	w.Flush()
}

// printCommandUsage prints the help message for a specific command.
func printCommandUsage(cmd *flag.FlagSet, name string, description string) {
	header := color.New(color.FgYellow, color.Bold)
	header.Fprintf(os.Stderr, "Usage: %s %s [subcommand] [flags]\n\n", os.Args[0], name)
	fmt.Fprintf(os.Stderr, "%s\n\n", description)

	if name == "logs" { 
		fmt.Fprintln(os.Stderr, "Available subcommands for logs:")
		fmt.Fprintln(os.Stderr, "  show\tDisplays locally stored MCP logs in a web interface.")
		fmt.Fprintln(os.Stderr, "  clear\tDeletes all locally stored MCP logs.")
		fmt.Fprintln(os.Stderr)
	} else if name == "auth" {
		fmt.Fprintln(os.Stderr, "Available subcommands for auth:")
		fmt.Fprintln(os.Stderr, "  login\tInitiate the device authorization flow to log in.")
		fmt.Fprintln(os.Stderr, "  status\tCheck the current authentication status.")
		fmt.Fprintln(os.Stderr, "  deauth\tLog out and remove locally stored authentication token.")
		fmt.Fprintln(os.Stderr, "  logout\tAlias for 'deauth'.")
		fmt.Fprintln(os.Stderr)
	}

	hasFlags := false
	cmd.VisitAll(func(f *flag.Flag) { hasFlags = true })

	if hasFlags {
		header.Fprintln(os.Stderr, "Flags for this command:")
		w := tabwriter.NewWriter(os.Stderr, 0, 0, 2, ' ', 0)
		cmd.SetOutput(w) // Set output for PrintDefaults
		cmd.PrintDefaults() // Use the command's PrintDefaults for its specific flags
		w.Flush()
		fmt.Fprintln(os.Stderr)
	} else if name != "logs" && name != "auth" { // Only print if no flags AND not a command group like 'logs'
		fmt.Fprintln(os.Stderr, "This command takes no flags.")
	}
}

// printFlag is a helper to print a single flag's usage with consistent styling.
func printFlag(w *tabwriter.Writer, f *flag.Flag) {
	flagNameStyle := color.New(color.FgCyan)
	flagTypeStyle := color.New(color.FgMagenta)

	flagId := fmt.Sprintf("  -%s", f.Name)
	name, usage := flag.UnquoteUsage(f)
	flagTypeStr := ""
	if len(name) > 0 {
		flagTypeStr = flagTypeStyle.Sprint(name)
	}

	description := usage
	if f.DefValue != "" && f.DefValue != "0" && f.DefValue != "false" {
		// Attempt to get the actual value to see if it's a string for quoting
		val := f.Value.(flag.Getter).Get()
		if _, okString := val.(string); okString {
		    description += fmt.Sprintf(" (default \"%s\")", f.DefValue) // Default quoting for strings
		} else {
			description += fmt.Sprintf(" (default %s)", f.DefValue)
		}
	}
	description = strings.ReplaceAll(description, "\n", "\n    \t")
	fmt.Fprintf(w, "%s %s\t%s\n", flagNameStyle.Sprint(flagId), flagTypeStr, description)
}
