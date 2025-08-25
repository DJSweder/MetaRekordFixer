// constants.go

// Package common provides shared functionality and constants for the MetaRekordFixer application.
// This file contains constants used across the application to replace hardcoded strings.
package common

// ModuleKeys - Constants for module identification in configuration
const (
	// ModuleKeyFlacFixer is the key for FlacFixer module
	ModuleKeyFlacFixer = "FlacFixer"

	// ModuleKeyDataDuplicator is the key for DataDuplicator module
	ModuleKeyDataDuplicator = "DataDuplicator"

	// ModuleKeyDatesMaster is the key for DatesMaster module
	ModuleKeyDatesMaster = "DatesMaster"

	// ModuleKeyFormatUpdater is the key for FormatUpdater module
	ModuleKeyFormatUpdater = "FormatUpdater"

	// ModuleKeyFormatConverter is the key for FormatConverter module
	ModuleKeyFormatConverter = "FormatConverter"
)

// SourceTypes - Constants for data source types
const (
	// SourceTypeFolder indicates a folder as a data source
	ContentTypeFolder = "folder"

	// SourceTypePlaylist indicates a playlist as a data source
	ContentTypePlaylist = "playlist"
)

// OperationNames - Constants for operation names used in ErrorContext
const (
	// OperationDatabaseQuery indicates a database query operation
	OperationDbQuery = "DatabaseQuery"

	// OperationDatabaseValidation indicates a database validation operation
	OperationDbValidation = "DatabaseValidation"

	// OperationLoadDataFromDatabase indicates a data loading operation from database
	OperationLoadDataFromDb = "LoadDataFromDatabase"

	// OperationMetadataSync indicates a metadata synchronization operation
	OperationMetadataCopy = "MetadataCopy"

	// OperationUpdateFlacMetadata indicates a FLAC metadata update operation
	OperationUpdateFlacMetadata = "UpdateFLACMetadata"

	// OperationReadDatabaseRecords indicates a database records reading operation
	OperationReadDatabaseRecords = "ReadDatabaseRecords"
)

// FileExtensions - Constants for file extensions
const (
	ExtensionFLAC = ".flac"

	ExtensionMP3 = ".mp3"

	ExtensionWAV = ".wav"

	ExtensionAIFF = ".aiff"

	ExtensionM4A = ".m4a"
)

// FileNames - Constants for file names
const (
	// FileNameSettings is the name of the configuration file
	FileNameSettings = "settings.conf"

	// FileNameLog is the name of the application log file
	FileNameLog = "metarekordfixer_app.log"

	//FileNameFFmpegLog is the name of the ffmpeg log file
	FileNameFFmpegLog = "metarekordfixer_ffmpeg.log"

	//FolderNameLog is the name of the log folder
	FolderNameLog = "log"
)

// ValidatorActions - Constants for validator actions
const (
	// ValidatorActionStart indicates the start validation action
	ValidatorActionStart = "start"
)

// AppIdentifiers - Constants for application identification
const (
	// AppID is the application identifier
	AppID = "com.metarekordfixer.app"

	//AppName is the application name
	AppName = "MetaRekordFixer"
)

// SQLFragments - Constants for frequently used SQL query fragments
const (
	// SQLSelectMaxID is an SQL query fragment for getting the maximum ID
	SQLSelectMaxID = "SELECT COALESCE(MAX(CAST(ID AS INTEGER)), 0) FROM"

	// SQLTableDJMDContent is the name of the djmdContent table in the database
	SQLTableDJMDContent = "djmdContent"

	// SQLTableDJMDCue is the name of the djmdCue table in the database
	SQLTableDJMDCue = "djmdCue"

	// SQLTableDJPlaylist is the name of the djmdPlaylist table in the database
	SQLTableDJMDPlaylist = "djmdPlaylist"

	// SQLTableDJMDArtist is the name of the djmdArtist table in the database
	SQLTableDJMDArtist = "djmdArtist"

	// SQLTableDJMDAlbum is the name of the djmdAlbum table in the database
	SQLTableDJMDAlbum = "djmdAlbum"
)
