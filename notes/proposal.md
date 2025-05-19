# Proposal: Adding PostHog Telemetry to ithena-cli

## 1. Goal

To gain insights into the usage patterns of `ithena-cli` by integrating an anonymized telemetry system. This will help understand:
-   How many unique users (machines) are actively using the CLI.
-   Which commands and features are most popular.
-   The scale of log interactions (e.g., views, clears).
-   The number of wrappers being configured.

This data will guide future development efforts and help prioritize features.

## 2. Proposed Telemetry Service: PostHog

**Why PostHog?**
-   **Developer-Friendly:** Offers straightforward SDKs (including Go).
-   **Product Analytics Focused:** Designed for event-based tracking and user behavior analysis.
-   **Cost-Effective:** Provides a generous free tier for cloud hosting, which should be sufficient for initial needs.
-   **Self-Hosting Option:** If usage grows significantly or data sovereignty becomes a major concern, PostHog can be self-hosted, offering more control and potentially lower costs at scale.
-   **Anonymization Focus:** Aligns well with the requirement to only collect anonymized data.

## 3. Data to Collect (Anonymized)

All data will be associated with a randomly generated, anonymous machine ID. No personally identifiable information (PII) will be collected.

-   **CLI Invocation:**
    -   Event: `cli_invoked`
    -   Properties: `anonymous_machine_id`, `cli_version`, `os_type`, `arch_type`
-   **Command Execution:**
    -   Event: `command_executed`
    -   Properties:
        -   `anonymous_machine_id`
        -   `command_name` (e.g., "auth", "logs", "direct_wrap", "profile_wrap")
        -   `subcommand_name` (e.g., for `auth`: "login", "status", "deauth"; for `logs`: "show", "clear")
        -   `profile_name` (if `command_name` is "profile_wrap")
-   **Wrapper Configuration:**
    -   Event: `wrapper_config_loaded` (triggered when a wrapper config file is successfully parsed)
    -   Properties: `anonymous_machine_id`, `configured_wrapper_count`
-   **Log Interaction:**
    -   Event: `logs_action_taken`
    -   Properties:
        -   `anonymous_machine_id`
        -   `log_action` ("show_requested", "cleared")

## 4. High-Level Implementation Steps

1.  **Add PostHog Go SDK:** Include `github.com/posthog/posthog-go` as a dependency.
2.  **Anonymous Machine ID Generation:**
    -   On first run (or if ID doesn't exist), generate a UUID v4.
    -   Store this ID locally (e.g., in `~/.ithena/telemetry_id.txt` or as part of the existing CLI config).
    -   This ID will be sent with every event.
3.  **Telemetry Module/Package:**
    -   Consider creating a new package (e.g., `packages/cli/telemetry`) or integrating into the existing `packages/cli/observability` package.
    -   This module will handle:
        -   Initialization of the PostHog client (API key, endpoint).
        -   Loading/generating the anonymous machine ID.
        -   A generic `TrackEvent(eventName string, properties map[string]interface{})` function.
        -   Ensuring events are flushed on CLI exit (integrating with `observability.ShutdownObservability()`).
4.  **Configuration:**
    -   Allow configuration of PostHog API key and endpoint via environment variables (e.g., `ITHENA_POSTHOG_KEY`, `ITHENA_POSTHOG_ENDPOINT`).
    -   Default to a disabled state if no API key is provided.

    **4.1. API Key Management for Open Source:**
    -   **No API Key in Public Code:** The PostHog API key for official `ithena-cli` telemetry will **NOT** be present in the public source code.
    -   **Official Builds:** For official releases built via GitHub Actions (as per `release.yml`), the API key will be injected at build time using Go's `ldflags` and GitHub Secrets. This means the compiled binary distributed by `ithena-one` will have telemetry enabled (unless opted out by the user).
    -   **Source Builds/Forks:** Users building from source or forking the repository will not have the official API key. Telemetry will be disabled by default in these builds. If these users wish to send data to their *own* PostHog instance, they can provide their API key and endpoint via the environment variables (`ITHENA_POSTHOG_KEY`, `ITHENA_POSTHOG_ENDPOINT`).

5.  **Opt-Out Mechanism:**
    -   Implement a clear way for users to opt-out of telemetry.
    -   This could be via an environment variable (e.g., `ITHENA_TELEMETRY_OPTOUT=true`) or a CLI flag.
    -   If opted out, the telemetry module should not initialize or send any data.
6.  **Integrate Event Tracking:**
    -   In `main.go`:
        -   Track `cli_invoked` at the start.
        -   Track `command_executed` within the command switch statements.
        -   Track `wrapper_config_loaded` after successful config parsing.
    -   In `cmd/logs/logs.go`:
        -   Track `logs_action_taken` for show and clear commands.
7.  **Documentation:** Update user documentation to clearly state:
    -   That anonymized telemetry is collected.
    -   What data is collected.
    -   Why it's collected.
    -   How to opt-out.

## 5. Privacy Considerations

-   **Anonymity is Key:** Strictly no PII. The generated machine ID is the only user-specific identifier.
-   **Transparency:** Clearly document the telemetry practice.
-   **User Control:** Provide an easy and effective opt-out mechanism.
-   **Data Minimization:** Only collect data that directly helps in understanding CLI usage and improving the tool.

## 6. Next Steps

-   [ ] Discuss and refine this proposal.
-   [ ] Choose the exact location for the anonymous machine ID storage.
-   [ ] Decide on the precise structure for the telemetry module/package.
-   [ ] Begin implementation, starting with the SDK integration and anonymous ID generation. 