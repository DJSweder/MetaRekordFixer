// common/config_types.go

// Package common provides shared types, functions, and utilities used across
// the MetaRekordFixer application.
// This file includes base module functionality, Configuration management, error handling, and other common components.

package common

// Cfg is the main structure that maps the entire Configuration file settings.conf.
// Contains global settings and specific Configurations for all modules.
type Cfg struct {
	Global  GlobalCfg  `json:"global"`
	Modules ModuleCfgs `json:"modules"`
}

// GlobalCfg contains global application settings, such as database path or preferred language.
type GlobalCfg struct {
	DatabasePath string `json:"DatabasePath"`
	Language     string `json:"Language"`
}

// FieldCfg defines the properties and value of a single Configuration field.
// This information is used for validation.
type FieldCfg struct {
	FieldType         string   `json:"FieldType"`
	Required          bool     `json:"Required"`
	DependsOn         string   `json:"DependsOn"`
	ActiveWhen        string   `json:"ActiveWhen"`
	ValidationType    string   `json:"ValidationType"`
	Value             string   `json:"Value"` // Always a string, conversion only occurs when used.
	ValidateOnActions []string `json:"ValidateOnActions"`
}

// ModuleCfgs groups the Configurations for all available modules.
// Each field corresponds to one module and its key in `settings.conf`.
type ModuleCfgs struct {
	FormatConverter FormatConverterCfg `json:"formatconverter"`
	DatesMaster     DatesMasterCfg     `json:"datesmaster"`
	FlacFixer       FlacFixerCfg       `json:"flacfixer"`
	DataDuplicator  DataDuplicatorCfg  `json:"dataduplicator"`
	FormatUpdater   FormatUpdaterCfg   `json:"formatupdater"`
}


// FormatConverterCfg defines all fields for the "Format Converter" module.
type FormatConverterCfg struct {
	SourceFolder     FieldCfg `json:"sourceFolder"`
	TargetFolder     FieldCfg `json:"targetFolder"`
	SourceFormat     FieldCfg `json:"sourceFormat"`
	TargetFormat     FieldCfg `json:"targetFormat"`
	MakeTargetFolder FieldCfg `json:"makeTargetFolder"`
	RewriteExisting  FieldCfg `json:"rewriteExisting"`
	MP3Bitrate       FieldCfg `json:"MP3Bitrate"`
	MP3Samplerate    FieldCfg `json:"MP3Samplerate"`
	FLACBitdepth     FieldCfg `json:"FLACBitdepth"`
	FLACSamplerate   FieldCfg `json:"FLACSamplerate"`
	FLACCompression  FieldCfg `json:"FLACCompression"`
	WAVBitdepth      FieldCfg `json:"WAVBitdepth"`
	WAVSamplerate    FieldCfg `json:"WAVSamplerate"`
}

// DatesMasterCfg defines all fields for the "Dates Master" module.
type DatesMasterCfg struct {
	CustomDate            FieldCfg `json:"customDate"`
	CustomDateFolders     FieldCfg `json:"customDateFolders"`
	ExcludeFoldersEnabled FieldCfg `json:"excludeFoldersEnabled"`
	ExcludedFolders       FieldCfg `json:"excludedFolders"`
}

// FlacFixerCfg defines all fields for the "Flac Fixer" module.
type FlacFixerCfg struct {
	SourceFolder FieldCfg `json:"sourceFolder"`
	Recursive    FieldCfg `json:"recursive"`
}

// DataDuplicatorCfg defines all fields for the "Data Duplicator" module.
type DataDuplicatorCfg struct {
	SourceType     FieldCfg `json:"sourceType"`
	SourceFolder   FieldCfg `json:"sourceFolder"`
	SourcePlaylist FieldCfg `json:"sourcePlaylist"`
	TargetType     FieldCfg `json:"targetType"`
	TargetFolder   FieldCfg `json:"targetFolder"`
	TargetPlaylist FieldCfg `json:"targetPlaylist"`
}

// FormatUpdaterCfg defines all fields for the "Format Updater" module.
type FormatUpdaterCfg struct {
	Folder     FieldCfg `json:"folder"`
	PlaylistID FieldCfg `json:"playlistID"`
}
