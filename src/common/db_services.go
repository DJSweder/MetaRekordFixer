// common/db_services.go

// Package common provides shared functionality and constants for the MetaRekordFixer application.
// This file contains:
// 		- functions for working with metadata read from music files
// 		- functions for working with metadata in the database

package common

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"MetaRekordFixer/locales"

	"github.com/dhowden/tag"
)

// ErrCancelled is a sentinel error used for programmatic cancellation flow control.
//   - It indicates that a long-running operation was cancelled by the user (via context).
//   - It MUST NOT be shown directly to the user; UI should render localized status
//     messages (e.g., common.status.stopping/stopped) instead.
//   - Keep this as a non-localized, stable identifier so errors.Is(err, ErrCancelled)
//     works reliably even when the error is wrapped.
var ErrCancelled = errors.New("operation cancelled")

// ReadMetadataFromFile reads metadata from an audio file using the github.com/dhowden/tag library.
// It supports reading metadata from different audio formats, currently focusing on FLAC.
//
// Parameters:
//   - filePath: The path to the audio file
//   - format: The format of the audio file (e.g., "FLAC", "MP3")
//
// Returns:
//   - A map of metadata key-value pairs
//   - An error if the file cannot be read or parsed
func ReadMetadataFromFile(filePath string, format string) (map[string]string, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read metadata using github.com/dhowden/tag
	metadata, err := tag.ReadFrom(file)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", locales.Translate("common.err.metadataread"), err)
	}

	// Extract metadata into a map
	metadataMap := make(map[string]string)

	// Get all fields from Raw() map for consistency
	rawData := metadata.Raw()
	if rawData != nil {
		if album, ok := rawData["album"]; ok {
			if str, ok := album.(string); ok {
				metadataMap["ALBUM"] = str
			}
		}
		if albumArtist, ok := rawData["albumartist"]; ok {
			if str, ok := albumArtist.(string); ok {
				metadataMap["ALBUMARTIST"] = str
			}
		}
		if origArtist, ok := rawData["origartist"]; ok {
			if str, ok := origArtist.(string); ok {
				metadataMap["ORIGARTIST"] = str
			}
		}
		if releaseDate, ok := rawData["releasedate"]; ok {
			if str, ok := releaseDate.(string); ok {
				metadataMap["RELEASEDATE"] = str
			}
		}
		if subtitle, ok := rawData["subtitle"]; ok {
			if str, ok := subtitle.(string); ok {
				metadataMap["SUBTITLE"] = str
			}
		}
	}

	return metadataMap, nil
}

// GetNextID retrieves the next available ID for a specified table in the database.
// It queries the maximum existing ID and increments it by 1.
//
// Parameters:
//   - dbMgr: The database manager instance
//   - tableName: The name of the table to get the next ID for
//
// Returns:
//   - The next available ID as a string
//   - An error if the database query fails
func GetNextID(dbMgr *DBManager, tableName string) (string, error) {
	var maxID int64

	query := fmt.Sprintf("SELECT COALESCE(MAX(CAST(ID AS INTEGER)), 0) FROM %s", tableName)
	row := dbMgr.QueryRow(query)
	if row == nil {
		return "", fmt.Errorf(locales.Translate("common.err.dbnotconnected"), dbMgr.GetDatabasePath())
	}
	err := row.Scan(&maxID)
	if err != nil {
		return "", fmt.Errorf("%s: %w", locales.Translate("common.err.dbmaxidcheck"), err)
	}

	maxID++
	return fmt.Sprintf("%d", maxID), nil
}

// GetNextUSN increments and retrieves the next USN (Update Sequence Number) from the agentRegistry table.
// This function is used to maintain consistency in database updates.
//
// Parameters:
//   - dbMgr: The database manager instance
//
// Returns:
//   - The next USN value as an integer
//   - An error if the database operation fails
func GetNextUSN(dbMgr *DBManager) (int64, error) {
	// Increment the USN counter
	updateQuery := `
		UPDATE agentRegistry
		SET int_1 = int_1 + 1
		WHERE registry_id = 'localUpdateCount'
	`

	err := dbMgr.Execute(updateQuery)
	if err != nil {
		return 0, err
	}

	// Get the new USN value
	var usn int64
	selectQuery := `
		SELECT int_1
		FROM agentRegistry
		WHERE registry_id = 'localUpdateCount'
	`

	row := dbMgr.QueryRow(selectQuery)
	if row == nil {
		return 0, fmt.Errorf(locales.Translate("common.err.dbnotconnected"), dbMgr.GetDatabasePath())
	}
	err = row.Scan(&usn)
	if err != nil {
		return 0, err
	}

	return usn, nil
}

// AddOrGetArtist adds a new artist to the djmdArtist table if it doesn't exist,
// or returns the ID of an existing artist with the same name.
//
// Parameters:
//   - dbMgr: The database manager instance
//   - artistName: The name of the artist to add or find
//   - usn: The Update Sequence Number to use for the new record
//
// Returns:
//   - The ID of the artist (new or existing)
//   - An error if the database operation fails
func AddOrGetArtist(dbMgr *DBManager, artistName string, usn int64) (string, error) {
	if artistName == "" {
		return "", nil
	}

	// Check if artist already exists
	var artistID string
	checkQuery := "SELECT ID FROM djmdArtist WHERE Name = ? COLLATE NOCASE"
	row := dbMgr.QueryRow(checkQuery, artistName)
	if row == nil {
		return "", fmt.Errorf(locales.Translate("common.err.dbnotconnected"), dbMgr.GetDatabasePath())
	}
	err := row.Scan(&artistID)

	// If artist exists, return its ID
	if err == nil {
		// Artist found - no log message needed here as it's not an action, just a check.
		return artistID, nil
	}

	// If error is not "no rows", return the error
	if err != sql.ErrNoRows {
		return "", err
	}

	// Artist doesn't exist, create new
	dbMgr.logger.Info("%s %s",
		fmt.Sprintf(locales.Translate("common.log.artist"), artistName),
		locales.Translate("common.log.dbinserted"))

	newID, err := GetNextID(dbMgr, "djmdArtist")
	if err != nil {
		return "", err
	}

	// Get current timestamp
	currentTime := time.Now().UTC().Format("2006-01-02 15:04:05.000 +00:00")

	// Insert new artist
	insertQuery := `
		INSERT INTO djmdArtist (
			ID, Name, rb_local_usn, created_at, updated_at
		) VALUES (
			?, ?, ?, ?, ?
		)
	`

	err = dbMgr.Execute(insertQuery, newID, artistName, usn, currentTime, currentTime)
	if err != nil {
		dbMgr.logger.Error(locales.Translate("common.log.dberrorat"), "djmdArtist", err)
		return "", err
	}

	return newID, nil
}

// GetAlbumIDFromTrack retrieves the AlbumID from djmdContent table for a specific track.
// This function is used to identify which album should be updated with AlbumArtistID.
//
// Parameters:
//   - dbMgr: The database manager instance
//   - trackID: The ID of the track in djmdContent table
//
// Returns:
//   - The AlbumID as a string (empty if not found or NULL)
//   - An error if the database operation fails
func GetAlbumIDFromTrack(dbMgr *DBManager, trackID string) (string, error) {
	var albumID sql.NullString

	query := "SELECT AlbumID FROM djmdContent WHERE ID = ?"
	row := dbMgr.QueryRow(query, trackID)
	if row == nil {
		return "", fmt.Errorf(locales.Translate("common.err.dbnotconnected"), dbMgr.GetDatabasePath())
	}
	err := row.Scan(&albumID)
	if err != nil {
		dbMgr.logger.Error(locales.Translate("common.log.dberrorat"), "djmdContent", err)
		return "", err
	}

	if albumID.Valid {
		return albumID.String, nil
	}

	return "", nil
}

// UpdateAlbumArtistID updates the AlbumArtistID in djmdAlbum table for a specific album.
// This function is used to assign the correct artist to an existing album.
//
// Parameters:
//   - dbMgr: The database manager instance
//   - albumID: The ID of the album in djmdAlbum table
//   - artistID: The ID of the artist to assign to the album
//   - usn: The Update Sequence Number to use for the update
//
// Returns:
//   - An error if the database operation fails
func UpdateAlbumArtistID(dbMgr *DBManager, albumID string, artistID string, usn int64) error {
	// No separate log message needed here; the action is logged by the caller if necessary.
	// Get current timestamp
	var artistName string
	artistNameQuery := "SELECT Name FROM djmdArtist WHERE ID = ?"
	row := dbMgr.QueryRow(artistNameQuery, artistID)
	if row != nil {
		row.Scan(&artistName)
	}

	dbMgr.logger.Info("%s %s",
		fmt.Sprintf(locales.Translate("common.log.artist"), artistName),
		fmt.Sprintf(locales.Translate("common.log.assignedalbum"), albumID))

	currentTime := time.Now().UTC().Format("2006-01-02 15:04:05.000 +00:00")

	updateQuery := `
		UPDATE djmdAlbum
		SET AlbumArtistID = ?, rb_local_usn = ?, updated_at = ?
		WHERE ID = ?
	`

	err := dbMgr.Execute(updateQuery, artistID, usn, currentTime, albumID)
	if err != nil {
		dbMgr.logger.Error(locales.Translate("common.log.dberrorat"), fmt.Sprintf("djmdAlbum/%s", albumID), err)
		return fmt.Errorf("%s: %w", locales.Translate("common.err.albumupdate"), err)
	}

	return nil
}

// ProcessSummary holds aggregated metrics for folder metadata processing.
type ProcessSummary struct {
	Total        int
	Updated      int
	NoChange     int
	SkippedZero  int
	MetadataErrs int
	DbMisses     int
	DbUpdateErrs int
	SkippedDirs  int
}

// ProcessFolderMetadata processes metadata from all FLAC files in a folder
// and updates the database accordingly.
//
// Parameters:
//   - dbMgr: The database manager instance
//   - folderPath: The path to the folder containing FLAC files
//   - recursive: Whether to process subfolders recursively
//   - onFilesFound: Callback invoked after counting files (can be nil)
//   - onProgress: Callback invoked during processing with progress and counts (can be nil)
//
// Returns:
//   - ProcessSummary with counters
//   - An error if the operation fails (fatal pre-processing errors only)
func ProcessFolderMetadata(
	ctx context.Context,
	dbMgr *DBManager,
	folderPath string,
	recursive bool,
	onFilesFound func(total int),
	onProgress func(progress float64, updated int, total int),
) (ProcessSummary, error) {
	// Find all FLAC files in the folder using the new safe file listing function
	flacFiles, skippedDirsFromProcessing, err := GetFilesInFolder(dbMgr.logger, folderPath, []string{".flac"}, recursive)

	if err != nil {
		return ProcessSummary{}, err
	}

	// Notify files found
	if onFilesFound != nil {
		onFilesFound(len(flacFiles))
	}

	// Return early if no files found
	if len(flacFiles) == 0 {
		return ProcessSummary{}, errors.New(locales.Translate("common.err.nofiles"))
	}

	// Early cancel check
	select {
	case <-ctx.Done():
		return ProcessSummary{}, ErrCancelled
	default:
	}

	// Get tracks from folder and create hash map for O(1) lookup
	tracks, err := dbMgr.GetTracksBasedOnFolder(folderPath)
	if err != nil {
		return ProcessSummary{}, err
	}

	// Create hash map: normalized path -> trackID
	trackMap := make(map[string]string)
	for _, track := range tracks {
		normalizedPath := NormalizePath(track.FolderPath)
		trackMap[normalizedPath] = track.ID
	}

	// Get a single USN for the entire operation
	usn, err := GetNextUSN(dbMgr)
	if err != nil {
		return ProcessSummary{}, err
	}

	// Process each FLAC file
	totalFiles := len(flacFiles)
	summary := ProcessSummary{Total: totalFiles, SkippedDirs: len(skippedDirsFromProcessing)}

	for i, flacFile := range flacFiles {
		// Cancellation check before processing each file
		select {
		case <-ctx.Done():
			return summary, ErrCancelled
		default:
		}

		// Zero-byte file skip detection
		if fi, statErr := os.Stat(flacFile); statErr != nil {
			dbMgr.logger.Error("%s %s",
				fmt.Sprintf(locales.Translate("common.log.file"), filepath.Base(flacFile)),
				locales.Translate("common.log.iswrong"))
			summary.MetadataErrs++
			continue
		} else if fi.Size() == 0 {
			dbMgr.logger.Error("%s %s",
				fmt.Sprintf(locales.Translate("common.log.file"), filepath.Base(flacFile)),
				locales.Translate("common.log.iswrong"))
			summary.SkippedZero++
			continue
		}

		// Process the file using hash map lookup
		updated, perr := updateFileMetadataInDB(dbMgr, flacFile, usn, trackMap)
		if perr != nil {
			// Classify errors for metrics and continue
			msg := perr.Error()
			switch {
			case strings.Contains(msg, locales.Translate("common.err.metadataread")):
				dbMgr.logger.Warning("%s %s",
					fmt.Sprintf(locales.Translate("common.log.file"), filepath.Base(flacFile)),
					locales.Translate("common.log.incorrmetadata"))
				summary.MetadataErrs++
			case strings.Contains(msg, locales.Translate("common.err.dbnotrackfound")):
				dbMgr.logger.Error("%s %s",
					fmt.Sprintf(locales.Translate("common.log.file"), filepath.Base(flacFile)),
					locales.Translate("common.log.dbnotfound"))
				summary.DbMisses++
			default:
				// General database error without SQL dump
				summary.DbUpdateErrs++
			}
			continue
		}

		if updated {
			summary.Updated++
		} else {
			summary.NoChange++
		}

		// Progress update after processing current file
		if onProgress != nil && totalFiles > 0 {
			onProgress(float64(i+1)/float64(totalFiles), summary.Updated, totalFiles)
		}
	}

	// Final progress update
	if onProgress != nil {
		onProgress(1.0, summary.Updated, summary.Total)
	}

	return summary, nil
}

// Updates a FLAC file’s metadata in the database and logs changes.
// Reads metadata via ReadMetadataFromFile.
// Looks up track ID using normalized path hash map.
// Updates ALBUMARTIST, ORIGARTIST, RELEASEDATE, SUBTITLE fields as present.
// Returns whether any field changed and any error encountered.
func updateFileMetadataInDB(dbMgr *DBManager, filePath string, usn int64, trackMap map[string]string) (bool, error) {
	// Read metadata from file
	metadata, err := ReadMetadataFromFile(filePath, "FLAC")
	if err != nil {
		dbMgr.logger.Warning("%s %s",
			fmt.Sprintf(locales.Translate("common.log.incorrmetadata"), filePath),
			locales.Translate("common.log.skipped"))
		return false, nil // Return nil to continue processing other files
	}

	// Convert path to database format and normalize for lookup
	dbPath := ToDbPath(filePath, false)
	normalizedDbPath := NormalizePath(dbPath)

	// Find track using hash map (O(1) lookup)
	trackID, exists := trackMap[normalizedDbPath]
	if !exists {
		return false, fmt.Errorf("%s: %s", locales.Translate("common.err.dbnotrackfound"), filepath.Base(filePath))
	}

	changed := false
	updatedFields := []string{}
	notUpdatedFields := []string{}

	// Process ALBUMARTIST if available
	if albumArtist, ok := metadata["ALBUMARTIST"]; ok && albumArtist != "" {
		// Get or create artist
		artistID, err := AddOrGetArtist(dbMgr, albumArtist, usn)
		if err != nil {
			dbMgr.logger.Error(locales.Translate("common.log.dberrorat"), "djmdArtist", err)
			return false, err
		}

		// Get AlbumID from the track (step 1 from scope)
		albumID, err := GetAlbumIDFromTrack(dbMgr, trackID)
		if err != nil {
			dbMgr.logger.Error(locales.Translate("common.log.dberrorat"), "djmdContent", err)
			return false, err
		}

		// Only update album if AlbumID exists (step 2-3 from scope)
		if albumID != "" {
			if err := UpdateAlbumArtistID(dbMgr, albumID, artistID, usn); err != nil {
				dbMgr.logger.Error(locales.Translate("common.log.dberrorat"), fmt.Sprintf("djmdAlbum/%s", albumID), err)
				return false, err
			}
			changed = true
			updatedFields = append(updatedFields, "ALBUMARTIST")
		} else {
			notUpdatedFields = append(notUpdatedFields, "ALBUMARTIST")
		}
	} else {
		notUpdatedFields = append(notUpdatedFields, "ALBUMARTIST")
	}

	// Process ORIGARTIST if available
	if origArtist, ok := metadata["ORIGARTIST"]; ok && origArtist != "" {
		// Get or create artist
		artistID, err := AddOrGetArtist(dbMgr, origArtist, usn)
		if err != nil {
			dbMgr.logger.Error(locales.Translate("common.log.dberrorat"), "djmdArtist", err)
			return false, err
		}

		// Update track's OrgArtistID
		updateQuery := `
			UPDATE djmdContent
			SET OrgArtistID = ?, rb_local_usn = ?
			WHERE ID = ?
		`
		if err := dbMgr.Execute(updateQuery, artistID, usn, trackID); err != nil {
			dbMgr.logger.Error(locales.Translate("common.log.dberrorat"), fmt.Sprintf("djmdContent/%s", trackID), err)
			return false, err
		}
		changed = true
		updatedFields = append(updatedFields, "ORIGARTIST")
	} else {
		notUpdatedFields = append(notUpdatedFields, "ORIGARTIST")
	}

	// Update RELEASEDATE and SUBTITLE if available
	var updateFields []string
	var updateValues []interface{}

	if releaseDate, ok := metadata["RELEASEDATE"]; ok {
		updateFields = append(updateFields, "ReleaseDate = ?")
		updateValues = append(updateValues, releaseDate)
	} else {
		notUpdatedFields = append(notUpdatedFields, "RELEASEDATE")
	}

	if subtitle, ok := metadata["SUBTITLE"]; ok {
		updateFields = append(updateFields, "Subtitle = ?")
		updateValues = append(updateValues, subtitle)
	} else {
		notUpdatedFields = append(notUpdatedFields, "SUBTITLE")
	}

	// If we have fields to update
	if len(updateFields) > 0 {
		// Add USN and ID to values
		updateValues = append(updateValues, usn, trackID)

		// Build and execute update query
		updateQuery := fmt.Sprintf(`
			UPDATE djmdContent
			SET %s, rb_local_usn = ?
			WHERE ID = ?
		`, strings.Join(updateFields, ", "))

		if err := dbMgr.Execute(updateQuery, updateValues...); err != nil {
			dbMgr.logger.Error(locales.Translate("common.log.dberrorat"), fmt.Sprintf("djmdContent/%s", trackID), err)
			return false, err
		}
		changed = true
		if _, ok := metadata["RELEASEDATE"]; ok {
			updatedFields = append(updatedFields, "RELEASEDATE")
		}
		if _, ok := metadata["SUBTITLE"]; ok {
			updatedFields = append(updatedFields, "SUBTITLE")
		}
	}

	// INFO summary of processed files
	dbMgr.logger.Info("%s, id: %s, %s %s, %s %s",
		fmt.Sprintf(locales.Translate("common.log.file"), filepath.Base(filePath)), trackID,
		locales.Translate("common.log.updated"), strings.Join(updatedFields, ", "),
		locales.Translate("common.log.notupdated"), strings.Join(notUpdatedFields, ", "))

	return changed, nil
}
