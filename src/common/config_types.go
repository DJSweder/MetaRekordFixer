//go:build ignore

// common/config_types.go

// Package common provides shared types, functions, and utilities used across
// the MetaRekordFixer application.
// This file includes base module functionality, configuration management, error handling, and other common components.

package common

// Config is the main structure that maps the entire configuration file settings.conf.
// Contains global settings and specific configurations for all modules.
type Config struct {
	Global  GlobalConfig  `json:"global"`
	Modules ModuleConfigs `json:"modules"`
}

// GlobalConfig contains global application settings, such as database path or preferred language.
type GlobalConfig struct {
	DatabasePath string `json:"DatabasePath"`
	Language     string `json:"Language"`
}

// FieldConfig defines the properties and value of a single configuration field.
// This information is used for validation.
type FieldConfig struct {
	FieldType         string   `json:"FieldType"`
	Required          bool     `json:"Required"`
	DependsOn         string   `json:"DependsOn"`
	ActiveWhen        string   `json:"ActiveWhen"`
	ValidationType    string   `json:"ValidationType"`
	Value             string   `json:"Value"` // Always a string, conversion only occurs when used.
	ValidateOnActions []string `json:"ValidateOnActions"`
}

// ModuleConfigs groups the configurations for all available modules.
// Each field corresponds to one module and its key in `settings.conf`.
type ModuleConfigs struct {
	FormatConverter FormatConverterConfig `json:"formatconverter"`
	DatesMaster     DatesMasterConfig     `json:"datesmaster"`
	FlacFixer       FlacFixerConfig       `json:"flacfixer"`
	DataDuplicator  DataDuplicatorConfig  `json:"dataduplicator"`
	FormatUpdater   FormatUpdaterConfig   `json:"formatupdater"`
}

// FormatConverterConfig defines all fields for the "Format Converter" module.
type FormatConverterConfig struct {
	SourceFolder     FieldConfig `json:"source_folder"`
	TargetFolder     FieldConfig `json:"target_folder"`
	SourceFormat     FieldConfig `json:"source_format"`
	TargetFormat     FieldConfig `json:"target_format"`
	MakeTargetFolder FieldConfig `json:"make_target_folder"`
	RewriteExisting  FieldConfig `json:"rewrite_existing"`
	MP3Bitrate       FieldConfig `json:"mp3_bitrate"`
	MP3Samplerate    FieldConfig `json:"mp3_samplerate"`
	FLACBitdepth     FieldConfig `json:"flac_bitdepth"`
	FLACSamplerate   FieldConfig `json:"flac_samplerate"`
	FLACCompression  FieldConfig `json:"flac_compression"`
	WAVBitdepth      FieldConfig `json:"wav_bitdepth"`
	WAVSamplerate    FieldConfig `json:"wav_samplerate"`
}

// DatesMasterConfig defines all fields for the "Dates Master" module.
type DatesMasterConfig struct {
	CustomDate            FieldConfig `json:"custom_date"`
	CustomDateFolders     FieldConfig `json:"custom_date_folders"`
	ExcludeFoldersEnabled FieldConfig `json:"exclude_folders_enabled"`
	ExcludedFolders       FieldConfig `json:"excluded_folders"`
}

// FlacFixerConfig defines all fields for the "Flac Fixer" module.
type FlacFixerConfig struct {
	SourceFolder FieldConfig `json:"source_folder"`
	Recursive    FieldConfig `json:"recursive"`
}

// DataDuplicatorConfig defines all fields for the "Data Duplicator" module.
type DataDuplicatorConfig struct {
	SourceType     FieldConfig `json:"source_type"`
	SourceFolder   FieldConfig `json:"source_folder"`
	SourcePlaylist FieldConfig `json:"source_playlist"`
	TargetType     FieldConfig `json:"target_type"`
	TargetFolder   FieldConfig `json:"target_folder"`
	TargetPlaylist FieldConfig `json:"target_playlist"`
}

// FormatUpdaterConfig defines all fields for the "Format Updater" module.
type FormatUpdaterConfig struct {
	Folder     FieldConfig `json:"folder"`
	PlaylistID FieldConfig `json:"playlist_id"`
}
