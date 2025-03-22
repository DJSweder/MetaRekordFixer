// common/db.go

package common

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"MetaRekordFixer/locales"

	_ "github.com/mutecomm/go-sqlcipher/v4"
)

// Define parts for obsfuscation
var (
	dbPasswordPart1 = "402fd482c38817c35ffa8"
	dbPasswordPart2 = "ffb8c7d93143b749e7d3"
	dbPasswordPart3 = "15df7a81732a1ff43608497"
)

// getDbPassword creates string by concatenating parts
func getDbPassword() string {
	return dbPasswordPart1 + dbPasswordPart2 + dbPasswordPart3
}

// DBManager provides unified database access for all modules
type DBManager struct {
	db                *sql.DB
	dbPath            string
	isConnected       bool
	mutex             sync.Mutex
	logger            *log.Logger
	errorHandler      *ErrorHandler
	useTransactions   bool
	activeTransaction *sql.Tx
	finalized         bool
}

// NewDBManager creates a new database manager
func NewDBManager(dbPath string, logger *log.Logger, errorHandler *ErrorHandler) (*DBManager, error) {
	dbDir := filepath.Dir(dbPath)
	err := EnsureDirectoryExists(dbDir)
	if err != nil {
		return nil, fmt.Errorf(locales.Translate("common.db.dirensureerr"), err)
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
		manager.logger = log.New(os.Stdout, "DB: ", log.LstdFlags)
	}

	// Removed automatic connection - will connect only when needed
	// err = manager.Connect()
	// if err != nil {
	// 	return nil, err
	// }

	return manager, nil
}

// Connect establishes a connection to the database
func (m *DBManager) Connect() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.isConnected {
		return nil
	}

	// Check if database path is set
	if m.dbPath == "" {
		return fmt.Errorf("database path is not configured")
	}

	connStr := fmt.Sprintf("file:%s?_pragma_key=%s&_pragma_cipher_compatibility=3&_pragma_cipher_page_size=4096", m.dbPath, getDbPassword())
	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return fmt.Errorf(locales.Translate("common.db.dbopenerr"), err)
	}

	err = db.Ping()
	if err != nil {
		db.Close()
		return fmt.Errorf(locales.Translate("common.db.dbconnecterr"), err)
	}

	// Set pragmas to disable WAL mode and optimize performance
	_, err = db.Exec("PRAGMA journal_mode=DELETE")
	if err != nil {
		db.Close()
		return fmt.Errorf("failed to set journal mode: %w", err)
	}

	_, err = db.Exec("PRAGMA synchronous=FULL")
	if err != nil {
		db.Close()
		return fmt.Errorf("failed to set synchronous mode: %w", err)
	}

	m.db = db
	m.isConnected = true
	m.logger.Printf("Connected to database: %s", m.dbPath)

	return nil
}

// Close closes the database connection
// This method is deprecated, use Finalize() instead
func (m *DBManager) Close() error {
	// For backward compatibility, we call Finalize()
	return m.Finalize()
}

// EnsureConnected ensures the database connection is active
func (m *DBManager) EnsureConnected(skipConnect bool) error {
	if !m.isConnected && !skipConnect {
		return m.Connect()
	}
	if !m.isConnected && skipConnect {
		return fmt.Errorf(locales.Translate("common.db.dbnotconnectederr"), m.dbPath)
	}
	return nil
}

// BeginTransaction starts a new transaction
func (m *DBManager) BeginTransaction() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.isConnected {
		return fmt.Errorf(locales.Translate("common.db.dbnotconnectederr"), m.dbPath)
	}

	if m.activeTransaction != nil {
		return fmt.Errorf(locales.Translate("common.db.txactiveerr"), m.dbPath)
	}

	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf(locales.Translate("common.db.txbeginerr"), err)
	}

	m.activeTransaction = tx
	m.useTransactions = true

	return nil
}

// CommitTransaction commits the current transaction
func (m *DBManager) CommitTransaction() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.useTransactions || m.activeTransaction == nil {
		return fmt.Errorf(locales.Translate("common.db.txnoactiveerr"), m.dbPath)
	}

	err := m.activeTransaction.Commit()
	if err != nil {
		return fmt.Errorf(locales.Translate("common.db.txcommiterr"), err)
	}

	m.activeTransaction = nil
	return nil
}

// RollbackTransaction rolls back the current transaction
func (m *DBManager) RollbackTransaction() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.useTransactions || m.activeTransaction == nil {
		return fmt.Errorf(locales.Translate("common.db.txnoactiveerr"), m.dbPath)
	}

	err := m.activeTransaction.Rollback()
	if err != nil {
		return fmt.Errorf(locales.Translate("common.db.txrollbackerr"), err)
	}

	m.activeTransaction = nil
	return nil
}

// Execute runs an SQL statement with parameters
func (m *DBManager) Execute(query string, args ...interface{}) error {
	err := m.EnsureConnected(false)
	if err != nil {
		return err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	_, execErr := m.db.Exec(query, args...)
	if execErr != nil {
		return fmt.Errorf(locales.Translate("common.db.queryexecerr"), execErr)
	}

	return nil
}

// Query executes an SQL query and returns rows
func (m *DBManager) Query(query string, args ...interface{}) (*sql.Rows, error) {
	err := m.EnsureConnected(false)
	if err != nil {
		return nil, err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	rows, queryErr := m.db.Query(query, args...)
	if queryErr != nil {
		return nil, fmt.Errorf(locales.Translate("common.db.queryfailederr"), queryErr)
	}

	return rows, nil
}

// QueryWithoutConnect executes an SQL query without ensuring a database connection
// This is useful during initialization when we want to avoid connecting to the database
func (m *DBManager) QueryWithoutConnect(query string, args ...interface{}) (*sql.Rows, error) {
	if !m.isConnected {
		return nil, fmt.Errorf("database not connected")
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	rows, queryErr := m.db.Query(query, args...)
	if queryErr != nil {
		return nil, fmt.Errorf(locales.Translate("common.db.queryfailederr"), queryErr)
	}

	return rows, nil
}

// QueryRow executes an SQL query and returns a single row
func (m *DBManager) QueryRow(query string, args ...interface{}) *sql.Row {
	err := m.EnsureConnected(false)
	if err != nil {
		return nil
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.db.QueryRow(query, args...)
}

// TableExists checks if a table exists in the database
func (m *DBManager) TableExists(tableName string) (bool, error) {
	err := m.EnsureConnected(false)
	if err != nil {
		return false, err
	}

	query := `SELECT name FROM sqlite_master WHERE type='table' AND name=?`
	row := m.QueryRow(query, tableName)

	var name string
	err = row.Scan(&name)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf(locales.Translate("common.db.tablecheckerr"), tableName, err)
	}

	return true, nil
}

// BackupDatabase creates a backup of the database
func (m *DBManager) BackupDatabase() error {
	// Check if database path is empty or not set
	if m.dbPath == "" {
		return fmt.Errorf(locales.Translate("common.db.nopath"), m.dbPath)
	}

	// Check if database file exists
	if _, err := os.Stat(m.dbPath); os.IsNotExist(err) {
		return fmt.Errorf(locales.Translate("common.db.filenotexist"), m.dbPath)
	}

	// Generate the backup file name with the current timestamp
	backupFileName := fmt.Sprintf("master_backup_%s.db", time.Now().Format("2006-01-02@15_04_05"))
	backupPath := filepath.Join(filepath.Dir(m.dbPath), backupFileName)

	// Copy the database file to the backup location
	err := CopyFile(m.dbPath, backupPath)
	if err != nil {
		return fmt.Errorf(locales.Translate("common.db.backupcopyerr"), err)
	}

	m.logger.Printf("Database backup created: %s", backupPath)
	return nil
}

// GetPlaylists loads all playlists from the database
func (m *DBManager) GetPlaylists() ([]PlaylistItem, error) {
	err := m.EnsureConnected(false)
	if err != nil {
		return nil, fmt.Errorf(locales.Translate("common.db.connecterr"), err)
	}

	query := `
		SELECT 
			djmd_content.ID,
			djmd_content.Title,
			djmd_content.ParentID,
			djmd_content.Attribute
		FROM djmd_content 
		WHERE ContentType = 1 
		ORDER BY Title`

	rows, err := m.Query(query)
	if err != nil {
		return nil, fmt.Errorf(locales.Translate("common.db.playlistsloaderr"), err)
	}
	defer rows.Close()

	var playlists []PlaylistItem
	for rows.Next() {
		var playlist PlaylistItem
		var attribute int
		err := rows.Scan(&playlist.ID, &playlist.Name, &playlist.ParentID, &attribute)
		if err != nil {
			return nil, fmt.Errorf(locales.Translate("common.db.playlistscanerr"), err)
		}
		playlists = append(playlists, playlist)
	}

	return playlists, nil
}

// Finalize ensures the database connection is properly closed.
// This should be called when the DBManager is no longer needed.
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

	// If there's an active transaction, roll it back
	if m.activeTransaction != nil {
		m.logger.Printf("Rolling back active transaction during finalization")
		m.activeTransaction.Rollback()
		m.activeTransaction = nil
	}

	// Force synchronization before closing - helps with removing .db-shm and .db-wal files
	_, err := m.db.Exec("PRAGMA wal_checkpoint(FULL)")
	if err != nil {
		m.logger.Printf("Warning: Failed to execute WAL checkpoint: %v", err)
		// Continue despite error
	}

	// Optimize the database to clean up prepared statements
	_, err = m.db.Exec("PRAGMA optimize")
	if err != nil {
		m.logger.Printf("Warning: Failed to optimize database: %v", err)
		// Continue despite error
	}

	// Close the database connection
	err = m.db.Close()
	if err != nil {
		return fmt.Errorf(locales.Translate("common.db.dbcloseerr"), err)
	}

	m.isConnected = false
	m.finalized = true
	m.logger.Printf("Database connection finalized: %s", m.dbPath)

	return nil
}
