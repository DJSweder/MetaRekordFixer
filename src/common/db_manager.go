// common/db_manager.go

// Package common implements shared functionality used across the MetaRekordFixer application.
// This file contains database management functionality for accessing and manipulating Rekordbox databases.

package common

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"MetaRekordFixer/locales"
	"strings"

	_ "github.com/mutecomm/go-sqlcipher/v4"
)

// Declaration of a variable with a password.
// The password is passed through ldflags during application compilation.
var dbPassword string

// getDbPassword is used internally by the DBManager to access encrypted Rekordbox database.
// Returns the complete database password string
func getDbPassword() string {
	return dbPassword
}

// DBManager provides unified database access for all modules in the application.
// It handles encrypted Rekordbox database connections, transactions, and query execution
// while providing error handling, logging, and thread safety through mutex locking.
type DBManager struct {
	db              *sql.DB       // database connection
	dbPath          string        // path to the database file
	isConnected     bool          // whether the connection is established
	mutex           sync.Mutex    // mutex for thread safety
	logger          *Logger       // logger for recording operations
	errorHandler    *ErrorHandler // handler for database errors
	useTransactions bool          // whether to use transactions
	finalized       bool          // whether the manager has been finalized
}

// NewDBManager creates a new database manager instance for the specified database path.
// It ensures the database directory exists and initializes the manager with the provided
// logger and error handler. If no logger is provided, an empty logger is created.
//
// Parameters:
//   - dbPath: Path to the Rekordbox database file
//   - logger: Logger instance for recording database operations
//   - errorHandler: Error handler for processing database errors
//
// Returns:
//   - A new DBManager instance and nil if successful
//   - nil and an error if the database directory cannot be created
func NewDBManager(dbPath string, logger *Logger, errorHandler *ErrorHandler) (*DBManager, error) {
	dbDir := filepath.Dir(dbPath)
	err := EnsureDirectoryExists(dbDir)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", locales.Translate("common.err.dbdirensure"), err)
	}

	manager := &DBManager{
		dbPath:          dbPath,
		isConnected:     false,
		logger:          logger,
		errorHandler:    errorHandler,
		useTransactions: false,
		finalized:       false,
	}

	if manager.logger == nil {
		manager.logger = &Logger{} // Create empty logger if none provided
	}

	return manager, nil
}

// Connect establishes a connection to the encrypted Rekordbox database.
// This method performs several validation steps:
// 1. Checks if the database path is set
// 2. Verifies the database file exists and is not empty
// 3. Attempts to open the database with the encryption key
// 4. Sets pragmas to optimize database performance
//
// The method is thread-safe through mutex locking and handles various error conditions
// with specific localized error messages.
//
// Returns:
//   - nil if the connection is successful
//   - An error with context if any step fails
func (m *DBManager) Connect() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.isConnected {
		return nil
	}

	// Check if database path is set
	if m.dbPath == "" {
		return errors.New(locales.Translate("common.err.dbpath"))
	}

	fileInfo, err := GetFileInfo(m.dbPath)
	if err != nil {
		// Error getting file info (e.g., path does not exist, permissions)
		// This could be considered a sub-type of "dbformat" or a specific file access error.
		// Let's use dbformat for now, as an inaccessible or non-existent file isn't a valid DB.
		return fmt.Errorf("%s: %w", locales.Translate("common.err.dbformat"), err)
	}

	if fileInfo.Size == 0 {
		// Explicitly return an error for zero-length files.
		// No need to wrap an error here, as GetFileInfo succeeded.
		return errors.New(locales.Translate("common.err.dbzerolength"))
	}

	connStr := fmt.Sprintf("file:%s?_pragma_key=%s&_pragma_cipher_compatibility=3&_pragma_cipher_page_size=4096", m.dbPath, getDbPassword())
	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return fmt.Errorf("%s: %w", locales.Translate("common.err.dbopen"), err)
	}

	err = db.Ping()
	if err != nil {
		db.Close()
		// Analyze type of error
		if strings.Contains(err.Error(), "file is not a database") {
			return fmt.Errorf("%s: %w", locales.Translate("common.err.dbformat"), err)
		}
		if strings.Contains(err.Error(), "no such table") {
			return fmt.Errorf("%s: %w", locales.Translate("common.err.dbtablesmissing"), err)
		}
		return fmt.Errorf("%s: %w", locales.Translate("common.err.dbconnect"), err)
	}

	// Set pragmas to disable WAL mode and optimize performance
	_, err = db.Exec("PRAGMA journal_mode=DELETE")
	if err != nil {
		db.Close()
		return fmt.Errorf("%s: %w", locales.Translate("common.err.dbpragma"), err)
	}

	_, err = db.Exec("PRAGMA synchronous=FULL")
	if err != nil {
		db.Close()
		return fmt.Errorf("%s: %w", locales.Translate("common.err.dbpragmasync"), err)
	}

	m.db = db
	m.isConnected = true
	m.logger.Info("Connected to database: %s", m.dbPath)

	return nil
}

// EnsureConnected ensures the database connection is active before performing operations.
// If skipConnect is false and the database is not connected, it will attempt to connect.
// If skipConnect is true and the database is not connected, it will return an error.
//
// Parameters:
//   - skipConnect: If true, don't attempt to connect if not already connected
//
// Returns:
//   - nil if the database is connected or was successfully connected
//   - An error if the database could not be connected or skipConnect is true and not connected
func (m *DBManager) EnsureConnected(skipConnect bool) error {
	if !m.isConnected && !skipConnect {
		return m.Connect()
	}
	if !m.isConnected && skipConnect {
		return fmt.Errorf(locales.Translate("common.err.dbnotconnected"), m.dbPath)
	}
	return nil
}

// Execute runs an SQL statement with parameters that doesn't return results.
// This method is typically used for INSERT, UPDATE, DELETE, and other statements
// that modify the database. It ensures the database is connected before execution
// and is thread-safe through mutex locking.
//
// Parameters:
//   - query: The SQL statement to execute
//   - args: Optional parameters for the SQL statement
//
// Returns:
//   - nil if the statement executed successfully
//   - An error if the database is not connected or the execution fails
func (m *DBManager) Execute(query string, args ...interface{}) error {
	err := m.EnsureConnected(false)
	if err != nil {
		return err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	_, execErr := m.db.Exec(query, args...)
	if execErr != nil {
		return fmt.Errorf("%s: %w", locales.Translate("common.err.dbqueryexec"), execErr)
	}

	return nil
}

// Query executes an SQL query and returns rows of results.
// This method is typically used for SELECT statements. It ensures the database
// is connected before execution and is thread-safe through mutex locking.
//
// Parameters:
//   - query: The SQL query to execute
//   - args: Optional parameters for the SQL query
//
// Returns:
//   - A pointer to sql.Rows containing the query results and nil if successful
//   - nil and an error if the database is not connected or the query fails
func (m *DBManager) Query(query string, args ...interface{}) (*sql.Rows, error) {
	err := m.EnsureConnected(false)
	if err != nil {
		return nil, err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	rows, queryErr := m.db.Query(query, args...)
	if queryErr != nil {
		return nil, fmt.Errorf("%s: %w", locales.Translate("common.err.dbquery"), queryErr)
	}

	return rows, nil
}

// QueryRow executes an SQL query and returns a single row.
// This method is typically used for SELECT statements where only one row is expected.
// It ensures the database is connected before execution and is thread-safe through mutex locking.
//
// Parameters:
//   - query: The SQL query to execute
//   - args: Optional parameters for the SQL query
//
// Returns:
//   - A pointer to sql.Row containing the query result
//   - nil if the database is not connected
func (m *DBManager) QueryRow(query string, args ...interface{}) *sql.Row {
	err := m.EnsureConnected(false)
	if err != nil {
		return nil
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.db.QueryRow(query, args...)
}

// BackupDatabase creates a backup of the database.
// This method creates a timestamped copy of the database file in the same directory.
// It performs validation checks on the database path before attempting the backup.
//
// Returns:
//   - nil if the backup was successful
//   - An error if the database path is invalid, the file doesn't exist, or the copy operation fails
func (m *DBManager) BackupDatabase() error {
	// Check if database path is empty or not set
	if m.dbPath == "" {
		return fmt.Errorf(locales.Translate("common.err.dbpath"), m.dbPath)
	}

	// Check if database file exists
	if _, err := os.Stat(m.dbPath); os.IsNotExist(err) {
		return fmt.Errorf(locales.Translate("common.err.dbnotexist"), m.dbPath)
	}

	// Finalize connection if exists
	m.logger.Info("%s", locales.Translate("common.log.dbclosing"))
	if err := m.Finalize(); err != nil {
		m.logger.Error("Failed to close database for backup: %v", err)
		return fmt.Errorf(locales.Translate("common.err.dbclose"), err)
	}

	// Generate the backup file name with the current timestamp
	backupFileName := fmt.Sprintf("master_backup_%s.db", time.Now().Format("2006-01-02@15_04_05"))
	backupPath := filepath.Join(filepath.Dir(m.dbPath), backupFileName)

	// Copy the database file to the backup location
	err := CopyFile(m.dbPath, backupPath)
	if err != nil {
		return fmt.Errorf("%s: %w", locales.Translate("common.err.dbbackup"), err)
	}

	m.logger.Info("Database backup created: %s", backupPath)
	return nil
}

// GetPlaylists loads all playlists from the database with their hierarchical structure.
// This method retrieves playlist information including ID, name, parent ID, and full path.
// The path is constructed to show the playlist hierarchy (e.g., "Parent > Child").
// Results are ordered by playlist sequence for proper hierarchical display.
//
// Returns:
//   - A slice of PlaylistItem structures and nil if successful
//   - nil and an error if the database is not connected or the query fails
func (m *DBManager) GetPlaylists() ([]PlaylistItem, error) {
	err := m.EnsureConnected(false)
	if err != nil {
		return nil, err // EnsureConnected (and thus Connect) already provides a localized error.
	}

	query := `
        SELECT p1.ID, p1.Name, p1.ParentID,
        CASE
            WHEN p2.Name IS NOT NULL THEN p2.Name || ' > ' || p1.Name
            ELSE p1.Name
        END as Path
        FROM djmdPlaylist p1
        LEFT JOIN djmdPlaylist p2 ON p1.ParentID = p2.ID
        ORDER BY
            CASE WHEN p2.ID IS NULL THEN p1.Seq ELSE p2.Seq END,
            CASE WHEN p2.ID IS NULL THEN 0 ELSE p1.Seq + 1 END
    `

	rows, err := m.Query(query)
	if err != nil {
		return nil, fmt.Errorf(locales.Translate("common.err.playlistload"), err)
	}
	defer rows.Close()

	var playlists []PlaylistItem
	for rows.Next() {
		var playlist PlaylistItem
		err := rows.Scan(&playlist.ID, &playlist.Name, &playlist.ParentID, &playlist.Path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", locales.Translate("common.err.dbplaylistscan"), err)
		}
		playlists = append(playlists, playlist)
	}

	return playlists, nil
}

// Finalize ensures the database connection is properly closed.
// This method should be called when the DBManager is no longer needed to release
// database resources and prevent connection leaks. It is thread-safe through mutex locking
// and idempotent (can be called multiple times safely).
//
// Returns:
//   - nil if the connection was successfully closed or was already closed
//   - An error if closing the connection fails
func (m *DBManager) Finalize() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.finalized {
		return nil
	}

	if !m.isConnected || m.db == nil {
		m.finalized = true
		return nil
	}

	// Force synchronization before closing - helps with removing .db-shm and .db-wal files
	_, err := m.db.Exec("PRAGMA wal_checkpoint(FULL)")
	if err != nil {
		m.logger.Info("Warning: Failed to execute WAL checkpoint: %v", err)
		// Continue despite error
	}

	// Optimize the database to clean up prepared statements
	_, err = m.db.Exec("PRAGMA optimize")
	if err != nil {
		m.logger.Info("Warning: Failed to optimize database: %v", err)
		// Continue despite error
	}

	// Close the database connection
	err = m.db.Close()
	if err != nil {
		return fmt.Errorf("%s: %w", locales.Translate("common.err.dbclosefinal"), err)
	}

	m.isConnected = false
	m.finalized = true
	m.logger.Info("Database connection finalized: %s", m.dbPath)

	return nil
}

// GetTracksBasedOnFolder retrieves all tracks from a specific folder in the Rekordbox database.
// This method converts the provided folder path to the database format and queries
// the djmdContent table for tracks with matching folder paths. Results are ordered by filename.
//
// Parameters:
//   - folderPath: The filesystem path of the folder to search for tracks
//
// Returns:
//   - A slice of TrackItem structures and nil if successful
//   - nil and an error if the database is not connected, the query fails, or no tracks are found
func (m *DBManager) GetTracksBasedOnFolder(folderPath string) ([]TrackItem, error) {
	err := m.EnsureConnected(false)
	if err != nil {
		return nil, fmt.Errorf(locales.Translate("common.err.dbconnect"), err)
	}

	// Convert path to database format
	dbPath := ToDbPath(folderPath, true)

	query := `
        SELECT 
            c.ID, 
            c.FolderPath, 
            c.FileNameL, 
            c.StockDate, 
            c.DateCreated, 
            c.ColorID, 
            c.DJPlayCount
        FROM djmdContent c
        WHERE c.FolderPath LIKE ? COLLATE BINARY  
        ORDER BY c.FileNameL
    `

	rows, err := m.Query(query, dbPath+"%")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", locales.Translate("common.err.dbqueryfolderfailed"), err)
	}
	defer rows.Close()

	var tracks []TrackItem
	for rows.Next() {
		var track TrackItem
		scanErr := rows.Scan(
			&track.ID,
			&track.FolderPath,
			&track.FileNameL,
			&track.StockDate,
			&track.DateCreated,
			&track.ColorID,
			&track.DJPlayCount,
		)
		if scanErr != nil {
			return nil, fmt.Errorf("%s: %w", locales.Translate("common.err.dbtrackscan"), scanErr)
		}
		tracks = append(tracks, track)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", locales.Translate("common.err.dbrowsiteration"), err)
	}

	if len(tracks) == 0 {
		folderName := filepath.Base(folderPath)
		return nil, fmt.Errorf(locales.Translate("common.err.dbfoldermatch"), folderName)
	}

	return tracks, nil
}

// GetTracksBasedOnPlaylist retrieves all tracks from a specific playlist in the Rekordbox database.
// This method joins the djmdContent and djmdSongPlaylist tables to find tracks associated
// with the specified playlist ID. Results are ordered by filename.
//
// Parameters:
//   - playlistID: The unique identifier of the playlist to retrieve tracks from
//
// Returns:
//   - A slice of TrackItem structures and nil if successful
//   - nil and an error if the database is not connected or the query fails
func (m *DBManager) GetTracksBasedOnPlaylist(playlistID string) ([]TrackItem, error) {
	err := m.EnsureConnected(false)
	if err != nil {
		return nil, fmt.Errorf(locales.Translate("common.err.dbconnect"), err)
	}

	query := `
        SELECT 
            c.ID, 
            c.FolderPath, 
            c.FileNameL, 
            c.StockDate, 
            c.DateCreated, 
            c.ColorID, 
            c.DJPlayCount
        FROM djmdContent c
        JOIN djmdSongPlaylist sp ON c.ID = sp.ContentID
        WHERE sp.PlaylistID = ?
        ORDER BY c.FileNameL
    `

	rows, err := m.Query(query, playlistID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tracks in playlist: %w", err)
	}
	defer rows.Close()

	var tracks []TrackItem
	for rows.Next() {
		var track TrackItem
		err := rows.Scan(
			&track.ID,
			&track.FolderPath,
			&track.FileNameL,
			&track.StockDate,
			&track.DateCreated,
			&track.ColorID,
			&track.DJPlayCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan track row: %w", err)
		}
		tracks = append(tracks, track)
	}

	return tracks, nil
}

// GetTrackHotCues retrieves all hot cues for a specific track from the Rekordbox database.
// This method queries the djmdCue table for all cue points associated with the specified track ID.
// The results are returned as a slice of maps to accommodate the dynamic nature of cue point data.
//
// Parameters:
//   - trackID: The unique identifier of the track to retrieve hot cues for
//
// Returns:
//   - A slice of maps containing hot cue data and nil if successful
//   - nil and an error if the database is not connected or the query fails
func (m *DBManager) GetTrackHotCues(trackID string) ([]map[string]interface{}, error) {
	err := m.EnsureConnected(false)
	if err != nil {
		return nil, fmt.Errorf(locales.Translate("common.err.dbconnect"), err)
	}

	query := `
        SELECT 
            ID, ContentID, InMsec, InFrame, InMpegFrame, InMpegAbs, 
            OutMsec, OutFrame, OutMpegFrame, OutMpegAbs, 
            Kind, Color, ColorTableIndex, ActiveLoop, Comment, 
            BeatLoopSize, CueMicrosec, InPointSeekInfo, OutPointSeekInfo, 
            ContentUUID, UUID, rb_data_status, rb_local_data_status, 
            rb_local_deleted, rb_local_synced
        FROM djmdCue
        WHERE ContentID = ?
    `

	rows, err := m.Query(query, trackID)
	if err != nil {
		return nil, fmt.Errorf("failed to query hot cues: %w", err)
	}
	defer rows.Close()

	var hotCues []map[string]interface{}

	// Load column names, this is needed for dynamic mapping
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get column names: %w", err)
	}

	for rows.Next() {
		// Create dynamic slice for values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))

		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		err := rows.Scan(valuePtrs...)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Create a map for the hot cue
		hotCue := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			hotCue[col] = val
		}

		hotCues = append(hotCues, hotCue)
	}

	return hotCues, nil
}

// GetDatabasePath returns the configured database path.
// This method provides read access to the internal database path configuration.
//
// Returns:
//   - The current database file path as a string
func (m *DBManager) GetDatabasePath() string {
	return m.dbPath
}

// TrackItem represents a track from the djmdContent table with basic metadata.
// This structure contains essential information about a track in the Rekordbox database,
// including its unique identifier, file location, and various metadata fields.
type TrackItem struct {
	ID          string
	FolderPath  string
	FileNameL   string
	StockDate   NullString
	DateCreated NullString
	ColorID     NullInt64
	DJPlayCount NullInt64
}

// NullString represents a string that may be NULL in the database.
// This type implements the sql.Scanner interface to properly handle NULL values
// from the database and convert them to a meaningful Go representation.
type NullString struct {
	String string
	Valid  bool // Valid is true if String is not NULL
}

// NullInt64 represents an int64 that may be NULL in the database.
// This type implements the sql.Scanner interface to properly handle NULL values
// from the database and convert them to a meaningful Go representation.
type NullInt64 struct {
	Int64 int64
	Valid bool // Valid is true if Int64 is not NULL
}

// Scan implements the sql.Scanner interface for NullString.
// This method handles conversion from various database types to a string value,
// properly managing NULL values by setting Valid to false.
//
// Parameters:
//   - value: The database value to scan into the NullString
//
// Returns:
//   - nil if successful
//   - An error if the value cannot be converted to a string
func (ns *NullString) Scan(value interface{}) error {
	if value == nil {
		ns.String, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	switch v := value.(type) {
	case string:
		ns.String = v
	case []byte:
		ns.String = string(v)
	default:
		ns.String = fmt.Sprintf("%v", v)
	}
	return nil
}

// Scan implements the sql.Scanner interface for NullInt64.
// This method handles conversion from various database types to an int64 value,
// properly managing NULL values by setting Valid to false.
//
// Parameters:
//   - value: The database value to scan into the NullInt64
//
// Returns:
//   - nil if successful
//   - An error if the value cannot be converted to an int64
func (ni *NullInt64) Scan(value interface{}) error {
	if value == nil {
		ni.Int64, ni.Valid = 0, false
		return nil
	}
	ni.Valid = true
	switch v := value.(type) {
	case int64:
		ni.Int64 = v
	case int:
		ni.Int64 = int64(v)
	case float64:
		ni.Int64 = int64(v)
	case []byte:
		// Attempt to convert bytes to int64
		i, err := strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			return fmt.Errorf("NullInt64.Scan: failed to parse []byte '%s' as int: %w", string(v), err)
		}
		ni.Int64 = i
	case string:
		// Attempt to convert string to int64
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("NullInt64.Scan: failed to parse string '%s' as int: %w", v, err)
		}
		ni.Int64 = i
	default:
		return fmt.Errorf("cannot convert %T to int64", value)
	}
	return nil
}

// ValueOrNil returns the string value if valid, or nil if not valid.
// This method is useful for serialization or when interfacing with code
// that expects either a string or nil value.
//
// Returns:
//   - The string value as interface{} if Valid is true
//   - nil as interface{} if Valid is false
func (ns NullString) ValueOrNil() interface{} {
	if ns.Valid {
		return ns.String
	}
	return nil
}

// ValueOrNil returns the int64 value if valid, or nil if not valid.
// This method is useful for serialization or when interfacing with code
// that expects either an int64 or nil value.
//
// Returns:
//   - The int64 value as interface{} if Valid is true
//   - nil as interface{} if Valid is false
func (ni NullInt64) ValueOrNil() interface{} {
	if ni.Valid {
		return ni.Int64
	}
	return nil
}
