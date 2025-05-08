package localstore

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	// SQLite driver
	_ "modernc.org/sqlite" // Pure Go SQLite driver (no CGO)

	"github.com/ithena-one/Ithena/packages/cli/types" // Import the new types package
)

var verbose bool // Package-level verbosity, can be set by a setter if needed

// SetVerbose enables or disables verbose logging for the localstore package.
func SetVerbose(v bool) {
	verbose = v
}

// DB is a package-level variable to hold the database connection.
var DB *sql.DB

const currentSchemaVersion = 1
const logsTableName = "logs"

// InitDB initializes the SQLite database for local log storage.
// It ensures the database file and necessary tables are created and migrated if needed.
func InitDB(explicitDBPath string) error {
	dbPath := explicitDBPath
	var err error
	if dbPath == "" {
		dbPath, err = getDefaultLogStorePath()
		if err != nil {
			return fmt.Errorf("failed to get default log store path: %w", err)
		}
	}

	if verbose {
		log.Printf("LocalStore: Initializing database at %s", dbPath)
	}

	// Ensure the directory for the database file exists
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory %s: %w", dbDir, err)
	}

	// Open the SQLite database file. It will be created if it doesn't exist.
	// The DSN for modernc.org/sqlite is simply the path to the file.
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database at %s: %w", dbPath, err)
	}

	// Check if the database connection is actually working.
	if err = DB.Ping(); err != nil {
		DB.Close() // Close DB if ping fails
		DB = nil    // Reset global DB variable
		return fmt.Errorf("failed to ping database at %s: %w", dbPath, err)
	}

	if verbose {
		log.Println("LocalStore: Database opened successfully.")
	}

	// Create schema (logs table and schema_version table)
	err = createSchema()
	if err != nil {
		DB.Close()
		DB = nil
		return fmt.Errorf("failed to create/migrate schema: %w", err)
	}

	if verbose {
		log.Println("LocalStore: Schema initialized successfully.")
	}

	return nil
}

// createSchema handles the creation and migration of database schema.
func createSchema() error {
	// 1. Create schema_version table if it doesn't exist
	_, err := DB.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL PRIMARY KEY);`)
	if err != nil {
		return fmt.Errorf("failed to create schema_version table: %w", err)
	}

	// 2. Check current version
	var dbVersion int
	err = DB.QueryRow(`SELECT version FROM schema_version ORDER BY version DESC LIMIT 1;`).Scan(&dbVersion)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// No version found, assume new database, insert current version
			_, err = DB.Exec(`INSERT INTO schema_version (version) VALUES (?);`, currentSchemaVersion)
			if err != nil {
				return fmt.Errorf("failed to insert initial schema version: %w", err)
			}
			dbVersion = currentSchemaVersion
		} else {
			return fmt.Errorf("failed to query schema version: %w", err)
		}
	}

	// 3. Perform migrations if dbVersion < currentSchemaVersion
	if dbVersion < currentSchemaVersion {
		// Placeholder for migration logic if schema evolves in the future
		if verbose {
			log.Printf("LocalStore: Database schema version %d is older than current version %d. Migrating...", dbVersion, currentSchemaVersion)
		}
		// Example: if dbVersion == 1 && currentSchemaVersion == 2 { migrateToV2() }
		// For now, we just ensure the logs table for V1 exists.
		// Update the version after successful migration
		// _, err = DB.Exec(`UPDATE schema_version SET version = ? WHERE version = ?;`, currentSchemaVersion, dbVersion) // This is wrong, should insert new or be atomic
		// A better way for versioning is to have an upgrade path and insert new version record or update a single row.
		// For now, we'll just ensure the latest tables are there and update to currentSchemaVersion if it was a new DB.
	}

	// 4. Create logs table (version 1 schema)
	// Ensure this matches types.AuditRecord fields that need to be columnized vs JSON.
	createLogsTableSQL := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
		id TEXT NOT NULL PRIMARY KEY,
		timestamp TEXT NOT NULL,
		mcp_method TEXT,
		tool_name TEXT,
		duration_ms INTEGER,
		status TEXT NOT NULL,
		proxy_version TEXT,
		target_server_alias TEXT,
		request_preview TEXT, -- Stored as JSON
		response_preview TEXT, -- Stored as JSON
		error_details TEXT -- Stored as JSON
	);
	`, logsTableName)

	_, err = DB.Exec(createLogsTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create %s table: %w", logsTableName, err)
	}

	// Create indexes for common query patterns
	indexes := []string{
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON %s (timestamp DESC);", logsTableName),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_logs_status ON %s (status);", logsTableName),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_logs_tool_name ON %s (tool_name);", logsTableName),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_logs_mcp_method ON %s (mcp_method);", logsTableName),
	}

	for _, indexSQL := range indexes {
		_, err = DB.Exec(indexSQL)
		if err != nil {
			// Non-fatal, but log it
			log.Printf("LocalStore Warning: Failed to create index (%s): %v", indexSQL, err)
		}
	}

	// If we reached here and the initial dbVersion was less than current (e.g. new DB)
	// ensure the schema_version table reflects the current version.
	if dbVersion < currentSchemaVersion {
		// This is a simplified way, for a real app you might have version-specific migrations.
		// We are basically saying that by creating all tables up to currentSchemaVersion, we are at currentSchemaVersion.
		// If schema_version was empty, we already inserted currentSchemaVersion.
		// If it was an older version, we would run migrations then update.
		// For now, if it was old, we assume applying the latest CREATE TABLE IF NOT EXISTS is enough for V1.
		// A more robust migration system would track each version and apply incremental changes.
		// For current simple case, if we just created the table, and old version was less, we ensure it is set.
		// We already inserted `currentSchemaVersion` if `sql.ErrNoRows` was met.
		// If `dbVersion` was genuinely an older, existing version, we'd need proper migration steps here.
		// For now, this will ensure that if the DB was old and we just applied V1 schema, version is updated.
		// This part needs more robust handling if actual schema migrations are introduced.
		// _, err = DB.Exec(`INSERT OR REPLACE INTO schema_version (version) VALUES (?);`, currentSchemaVersion)
		// For a single row version table: 
		_, err = DB.Exec(`INSERT INTO schema_version (version) VALUES (?) ON CONFLICT(version) DO UPDATE SET version = excluded.version WHERE excluded.version > (SELECT MAX(version) FROM schema_version);`, currentSchemaVersion)
		// Actually, simpler for a single schema version table, just update if it exists, or insert if not (which we did earlier).
		// The earlier check for sql.ErrNoRows already handles inserting the first version.
		// If there was an older version, and we had migrations, this is where we'd update it AFTER migrations.
		// For now, we consider the schema up-to-date if all CREATE TABLE IF NOT EXISTS passes.
		// Let's assume for V1, if dbVersion was < currentSchemaVersion, and we are at V1, this is the first run for V1.
		if verbose && dbVersion < currentSchemaVersion {
			log.Printf("LocalStore: Schema potentially updated to version %d", currentSchemaVersion)
		}
	}

	return nil
}

// SaveBatch saves a batch of audit records to the local SQLite database.
func SaveBatch(records []types.AuditRecord) error {
	if DB == nil {
		return errors.New("localstore: database not initialized, call InitDB first")
	}

	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("localstore: failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if commit is not called or if any error occurs

	stmtSQL := fmt.Sprintf(`
	INSERT INTO %s (id, timestamp, mcp_method, tool_name, duration_ms, status, proxy_version, target_server_alias, request_preview, response_preview, error_details)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`, logsTableName)

	stmt, err := tx.Prepare(stmtSQL)
	if err != nil {
		return fmt.Errorf("localstore: failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, record := range records {
		// Serialize JSON fields
		reqPreviewBytes, err := json.Marshal(record.RequestPreview)
		if err != nil {
			log.Printf("LocalStore Warning: Failed to marshal RequestPreview for record %s: %v. Storing as NULL.", record.ID, err)
			reqPreviewBytes = []byte("null") // Store as SQL NULL or JSON null
		}
		respPreviewBytes, err := json.Marshal(record.ResponsePreview)
		if err != nil {
			log.Printf("LocalStore Warning: Failed to marshal ResponsePreview for record %s: %v. Storing as NULL.", record.ID, err)
			respPreviewBytes = []byte("null")
		}
		errDetailsBytes, err := json.Marshal(record.ErrorDetails)
		if err != nil {
			log.Printf("LocalStore Warning: Failed to marshal ErrorDetails for record %s: %v. Storing as NULL.", record.ID, err)
			errDetailsBytes = []byte("null")
		}

		// Handle potentially nil pointers for string/int fields by converting to sql.NullString, sql.NullInt64
		var mcpMethod sql.NullString
		if record.McpMethod != nil {
			mcpMethod = sql.NullString{String: *record.McpMethod, Valid: true}
		}
		var toolName sql.NullString
		if record.ToolName != nil {
			toolName = sql.NullString{String: *record.ToolName, Valid: true}
		}
		var durationMs sql.NullInt64
		if record.DurationMs != nil {
			durationMs = sql.NullInt64{Int64: *record.DurationMs, Valid: true}
		}
		var proxyVersion sql.NullString
		if record.ProxyVersion != nil {
			proxyVersion = sql.NullString{String: *record.ProxyVersion, Valid: true}
		}
		var targetServerAlias sql.NullString
		if record.TargetServerAlias != nil {
			targetServerAlias = sql.NullString{String: *record.TargetServerAlias, Valid: true}
		}

		_, err = stmt.Exec(
			record.ID,
			record.Timestamp, // Assuming this is already a string in ISO 8601 format
			mcpMethod,
			toolName,
			durationMs,
			record.Status,
			proxyVersion,
			targetServerAlias,
			string(reqPreviewBytes),
			string(respPreviewBytes),
			string(errDetailsBytes),
		)
		if err != nil {
			// Log the error and continue to try other records in the batch, but the transaction will be rolled back.
			log.Printf("LocalStore Error: Failed to execute statement for record %s: %v. Batch will be rolled back.", record.ID, err)
			return fmt.Errorf("localstore: failed to execute statement for record %s: %w", record.ID, err) // Ensure rollback
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("localstore: failed to commit transaction: %w", err)
	}

	if verbose {
		log.Printf("LocalStore: Successfully saved batch of %d records.", len(records))
	}
	return nil
}

// getDefaultLogStorePath helper function to get the default database path.
func getDefaultLogStorePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}
	ithenaConfigDir := filepath.Join(configDir, "ithena-cli") 
	if err := os.MkdirAll(ithenaConfigDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", ithenaConfigDir, err)
	}
	return filepath.Join(ithenaConfigDir, "local_logs.v1.db"), nil 
}

// GetDefaultLogStorePathForInfo returns the default path where local logs are stored.
// This is primarily for informational display to the user.
func GetDefaultLogStorePathForInfo() (string, error) {
	return getDefaultLogStorePath()
}

// LogQueryFilters defines available filters for querying logs.
// All filters are ANDed together if multiple are provided.
type LogQueryFilters struct {
	Status   string // e.g., "success", "failure"
	ToolName string // Exact match for tool_name
	McpMethod string // Exact match for mcp_method
	SearchTerm string // Simple text search across ID, and JSON previews (requires LIKE clause)
}

// QueryLogsResult holds the result of a log query, including total count for pagination.
type QueryLogsResult struct {
	Logs       []types.AuditRecord `json:"logs"`
	TotalCount int                 `json:"total_count"`
	Page       int                 `json:"page"`
	Limit      int                 `json:"limit"`
}

// QueryLogs retrieves a paginated and filtered list of logs from the database.
func QueryLogs(filters LogQueryFilters, page int, limit int) (*QueryLogsResult, error) {
	if DB == nil {
		return nil, errors.New("localstore: database not initialized")
	}

	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20 // Default limit
	}
	offset := (page - 1) * limit

	var queryArgs []interface{}
	whereClauses := []string{"1 = 1"} // Start with a true condition to simplify appending ANDs

	if filters.Status != "" {
		whereClauses = append(whereClauses, "status = ?")
		queryArgs = append(queryArgs, filters.Status)
	}
	if filters.ToolName != "" {
		whereClauses = append(whereClauses, "tool_name = ?")
		queryArgs = append(queryArgs, filters.ToolName)
	}
	if filters.McpMethod != "" {
		whereClauses = append(whereClauses, "mcp_method = ?")
		queryArgs = append(queryArgs, filters.McpMethod)
	}
	if filters.SearchTerm != "" {
		// Basic search: check ID and LIKE against JSON previews
		// This is not super efficient for JSON but okay for a local tool with moderate data.
		// For SQLite, JSON fields are just text, so LIKE works.
		searchTermPattern := "%" + filters.SearchTerm + "%"
		whereClauses = append(whereClauses, "(id LIKE ? OR request_preview LIKE ? OR response_preview LIKE ? OR error_details LIKE ?)")
		queryArgs = append(queryArgs, searchTermPattern, searchTermPattern, searchTermPattern, searchTermPattern)
	}

	baseQuery := fmt.Sprintf("SELECT id, timestamp, mcp_method, tool_name, duration_ms, status, proxy_version, target_server_alias, request_preview, response_preview, error_details FROM %s", logsTableName)
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", logsTableName)

	whereStr := strings.Join(whereClauses, " AND ")

	fullQuery := fmt.Sprintf("%s WHERE %s ORDER BY timestamp DESC LIMIT ? OFFSET ?", baseQuery, whereStr)
	fullCountQuery := fmt.Sprintf("%s WHERE %s", countQuery, whereStr)

	// Arguments for the main query (filters + limit + offset)
	finalQueryArgs := make([]interface{}, len(queryArgs))
	copy(finalQueryArgs, queryArgs)
	finalQueryArgs = append(finalQueryArgs, limit, offset)

	// Arguments for the count query (only filters)
	finalCountQueryArgs := make([]interface{}, len(queryArgs))
	copy(finalCountQueryArgs, queryArgs)

	rows, err := DB.Query(fullQuery, finalQueryArgs...)
	if err != nil {
		return nil, fmt.Errorf("localstore: failed to execute query logs: %w (Query: %s, Args: %v)", err, fullQuery, finalQueryArgs)
	}
	defer rows.Close()

	logs := []types.AuditRecord{}
	for rows.Next() {
		var r types.AuditRecord
		var reqPreviewJSON, respPreviewJSON, errDetailsJSON sql.NullString // For raw JSON strings
		// Need to use sql.NullString etc. for potentially NULL DB columns when scanning
		var mcpMethod, toolName, proxyVersion, targetServerAlias sql.NullString
		var durationMs sql.NullInt64

		err = rows.Scan(
			&r.ID, &r.Timestamp, &mcpMethod, &toolName, &durationMs,
			&r.Status, &proxyVersion, &targetServerAlias,
			&reqPreviewJSON, &respPreviewJSON, &errDetailsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("localstore: failed to scan log row: %w", err)
		}

		// Assign to pointers in AuditRecord if valid
		if mcpMethod.Valid { r.McpMethod = &mcpMethod.String }
		if toolName.Valid { r.ToolName = &toolName.String }
		if durationMs.Valid { r.DurationMs = &durationMs.Int64 }
		if proxyVersion.Valid { r.ProxyVersion = &proxyVersion.String }
		if targetServerAlias.Valid { r.TargetServerAlias = &targetServerAlias.String }

		// Deserialize JSON strings back into interface{}
		if reqPreviewJSON.Valid { json.Unmarshal([]byte(reqPreviewJSON.String), &r.RequestPreview) }
		if respPreviewJSON.Valid { json.Unmarshal([]byte(respPreviewJSON.String), &r.ResponsePreview) }
		if errDetailsJSON.Valid { json.Unmarshal([]byte(errDetailsJSON.String), &r.ErrorDetails) }

		logs = append(logs, r)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("localstore: error iterating log rows: %w", err)
	}

	var totalCount int
	err = DB.QueryRow(fullCountQuery, finalCountQueryArgs...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("localstore: failed to count logs: %w (Query: %s, Args: %v)", err, fullCountQuery, finalCountQueryArgs)
	}

	return &QueryLogsResult{Logs: logs, TotalCount: totalCount, Page: page, Limit: limit}, nil
}

// GetLogByID retrieves a single log entry by its ID.
func GetLogByID(id string) (*types.AuditRecord, error) {
	if DB == nil {
		return nil, errors.New("localstore: database not initialized")
	}

	query := fmt.Sprintf("SELECT id, timestamp, mcp_method, tool_name, duration_ms, status, proxy_version, target_server_alias, request_preview, response_preview, error_details FROM %s WHERE id = ?", logsTableName)
	
	row := DB.QueryRow(query, id)

	var r types.AuditRecord
	var reqPreviewJSON, respPreviewJSON, errDetailsJSON sql.NullString
	var mcpMethod, toolName, proxyVersion, targetServerAlias sql.NullString
	var durationMs sql.NullInt64

	err := row.Scan(
		&r.ID, &r.Timestamp, &mcpMethod, &toolName, &durationMs,
		&r.Status, &proxyVersion, &targetServerAlias,
		&reqPreviewJSON, &respPreviewJSON, &errDetailsJSON,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Log not found, return nil, nil (or a specific ErrNotFound error)
		}
		return nil, fmt.Errorf("localstore: failed to scan log row for ID %s: %w", id, err)
	}

	if mcpMethod.Valid { r.McpMethod = &mcpMethod.String }
	if toolName.Valid { r.ToolName = &toolName.String }
	if durationMs.Valid { r.DurationMs = &durationMs.Int64 }
	if proxyVersion.Valid { r.ProxyVersion = &proxyVersion.String }
	if targetServerAlias.Valid { r.TargetServerAlias = &targetServerAlias.String }

	if reqPreviewJSON.Valid { json.Unmarshal([]byte(reqPreviewJSON.String), &r.RequestPreview) }
	if respPreviewJSON.Valid { json.Unmarshal([]byte(respPreviewJSON.String), &r.ResponsePreview) }
	if errDetailsJSON.Valid { json.Unmarshal([]byte(errDetailsJSON.String), &r.ErrorDetails) }

	return &r, nil
}

// TODO: Implement QueryLogs and GetLogByID functions later for the 'logs show' command. 