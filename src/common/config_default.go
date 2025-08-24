// common/config_default.go

// Package common implements shared functionality used across the MetaRekordFixer application.
// This file contains default configuration values for all modules.

package common

// GetDefaultFormatConverterCfg returns default configuration for FormatConverter module
func GetDefaultFormatConverterCfg() FormatConverterCfg {
	return FormatConverterCfg{
		SourceFolder: FieldCfg{
			FieldType:         "folder",
			Required:          true,
			ValidationType:    "exists",
			Value:             "",
			ValidateOnActions: []string{ValidatorActionStart},
		},
		TargetFolder: FieldCfg{
			FieldType:         "folder",
			Required:          true,
			ValidationType:    "exists | write",
			Value:             "",
			ValidateOnActions: []string{ValidatorActionStart},
		},
		SourceFormat: FieldCfg{
			FieldType:         "select",
			Required:          true,
			ValidationType:    "none",
			Value:             "All",
			ValidateOnActions: []string{ValidatorActionStart},
		},
		TargetFormat: FieldCfg{
			FieldType:         "select",
			Required:          true,
			ValidationType:    "none",
			Value:             "MP3",
			ValidateOnActions: []string{ValidatorActionStart},
		},
		MakeTargetFolder: FieldCfg{
			FieldType:      "checkbox",
			Required:       false,
			ValidationType: "none",
			Value:          "false",
		},
		RewriteExisting: FieldCfg{
			FieldType:      "checkbox",
			Required:       false,
			ValidationType: "none",
			Value:          "false",
		},
		MP3Bitrate: FieldCfg{
			FieldType:         "select",
			Required:          true,
			DependsOn:         "targetFormat",
			ActiveWhen:        "MP3",
			ValidationType:    "none",
			Value:             "320k",
			ValidateOnActions: []string{ValidatorActionStart},
		},
		MP3Samplerate: FieldCfg{
			FieldType:         "select",
			Required:          true,
			DependsOn:         "targetFormat",
			ActiveWhen:        "MP3",
			ValidationType:    "none",
			Value:             "copy",
			ValidateOnActions: []string{ValidatorActionStart},
		},
		FLACBitdepth: FieldCfg{
			FieldType:         "select",
			Required:          true,
			DependsOn:         "targetFormat",
			ActiveWhen:        "FLAC",
			ValidationType:    "none",
			Value:             "copy",
			ValidateOnActions: []string{ValidatorActionStart},
		},
		FLACSamplerate: FieldCfg{
			FieldType:         "select",
			Required:          true,
			DependsOn:         "targetFormat",
			ActiveWhen:        "FLAC",
			ValidationType:    "none",
			Value:             "copy",
			ValidateOnActions: []string{ValidatorActionStart},
		},
		FLACCompression: FieldCfg{
			FieldType:         "select",
			Required:          true,
			DependsOn:         "targetFormat",
			ActiveWhen:        "FLAC",
			ValidationType:    "none",
			Value:             "12",
			ValidateOnActions: []string{ValidatorActionStart},
		},
		WAVBitdepth: FieldCfg{
			FieldType:         "select",
			Required:          true,
			DependsOn:         "targetFormat",
			ActiveWhen:        "WAV",
			ValidationType:    "none",
			Value:             "copy",
			ValidateOnActions: []string{ValidatorActionStart},
		},
		WAVSamplerate: FieldCfg{
			FieldType:         "select",
			Required:          true,
			DependsOn:         "targetFormat",
			ActiveWhen:        "WAV",
			ValidationType:    "none",
			Value:             "copy",
			ValidateOnActions: []string{ValidatorActionStart},
		},
	}
}

// GetDefaultDatesMasterCfg returns default configuration for DatesMaster module
func GetDefaultDatesMasterCfg() DatesMasterCfg {
	return DatesMasterCfg{
		CustomDate: FieldCfg{
			FieldType:         "date",
			Required:          true,
			ValidationType:    "valid_date",
			Value:             "",
			ValidateOnActions: []string{ValidatorActionStart},
		},
		CustomDateFolders: FieldCfg{
			FieldType:         "folder",
			Required:          true,
			ValidationType:    "exists",
			Value:             "",
			ValidateOnActions: []string{ValidatorActionStart},
		},
		ExcludeFoldersEnabled: FieldCfg{
			FieldType:      "checkbox",
			Required:       false,
			ValidationType: "none",
			Value:          "false",
		},
		ExcludedFolders: FieldCfg{
			FieldType:         "folder",
			Required:          false,
			DependsOn:         "excludeFoldersEnabled",
			ActiveWhen:        "true",
			ValidationType:    "exists",
			Value:             "",
			ValidateOnActions: []string{ValidatorActionStart},
		},
	}
}

// GetDefaultFlacFixerCfg returns default configuration for FlacFixer module
func GetDefaultFlacFixerCfg() FlacFixerCfg {
	return FlacFixerCfg{
		SourceFolder: FieldCfg{
			FieldType:         "folder",
			Required:          true,
			ValidationType:    "exists",
			Value:             "",
			ValidateOnActions: []string{ValidatorActionStart},
		},
		Recursive: FieldCfg{
			FieldType:      "checkbox",
			Required:       false,
			ValidationType: "none",
			Value:          "false",
		},
	}
}

// GetDefaultDataDuplicatorCfg returns default configuration for DataDuplicator module
func GetDefaultDataDuplicatorCfg() DataDuplicatorCfg {
	return DataDuplicatorCfg{
		SourceType: FieldCfg{
			FieldType:         "select",
			Required:          true,
			ValidationType:    "none",
			Value:             "folder",
			ValidateOnActions: []string{ValidatorActionStart},
		},
		SourceFolder: FieldCfg{
			FieldType:         "folder",
			Required:          true,
			DependsOn:         "sourceType",
			ActiveWhen:        "folder",
			ValidationType:    "exists",
			Value:             "",
			ValidateOnActions: []string{ValidatorActionStart},
		},
		SourcePlaylist: FieldCfg{
			FieldType:         "playlist",
			Required:          true,
			DependsOn:         "sourceType",
			ActiveWhen:        "playlist",
			ValidationType:    "filled",
			Value:             "",
			ValidateOnActions: []string{ValidatorActionStart},
		},
		TargetType: FieldCfg{
			FieldType:         "select",
			Required:          true,
			ValidationType:    "none",
			Value:             "folder",
			ValidateOnActions: []string{ValidatorActionStart},
		},
		TargetFolder: FieldCfg{
			FieldType:         "folder",
			Required:          true,
			DependsOn:         "targetType",
			ActiveWhen:        "folder",
			ValidationType:    "exists | write",
			Value:             "",
			ValidateOnActions: []string{ValidatorActionStart},
		},
		TargetPlaylist: FieldCfg{
			FieldType:         "playlist",
			Required:          true,
			DependsOn:         "targetType",
			ActiveWhen:        "playlist",
			ValidationType:    "filled",
			Value:             "",
			ValidateOnActions: []string{ValidatorActionStart},
		},
	}
}

// GetDefaultFormatUpdaterCfg returns default configuration for FormatUpdater module
func GetDefaultFormatUpdaterCfg() FormatUpdaterCfg {
	return FormatUpdaterCfg{
		Folder: FieldCfg{
			FieldType:         "folder",
			Required:          true,
			ValidationType:    "exists",
			Value:             "",
			ValidateOnActions: []string{ValidatorActionStart},
		},
		PlaylistID: FieldCfg{
			FieldType:         "playlist",
			Required:          true,
			ValidationType:    "filled",
			Value:             "",
			ValidateOnActions: []string{ValidatorActionStart},
		},
	}
}

// GetDefaultModuleCfg returns default configuration for any module by type
func GetDefaultModuleCfg(moduleType string) interface{} {
	switch moduleType {
	case ModuleKeyFormatConverter:
		return GetDefaultFormatConverterCfg()
	case ModuleKeyDatesMaster:
		return GetDefaultDatesMasterCfg()
	case ModuleKeyFlacFixer:
		return GetDefaultFlacFixerCfg()
	case ModuleKeyDataDuplicator:
		return GetDefaultDataDuplicatorCfg()
	case ModuleKeyFormatUpdater:
		return GetDefaultFormatUpdaterCfg()
	default:
		return nil
	}
}
