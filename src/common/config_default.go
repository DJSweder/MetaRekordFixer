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
			ValidateOnActions: []string{"start"},
		},
		TargetFolder: FieldCfg{
			FieldType:         "folder",
			Required:          true,
			ValidationType:    "exists | write",
			Value:             "",
			ValidateOnActions: []string{"start"},
		},
		SourceFormat: FieldCfg{
			FieldType:         "select",
			Required:          true,
			ValidationType:    "none",
			Value:             "All",
			ValidateOnActions: []string{"start"},
		},
		TargetFormat: FieldCfg{
			FieldType:         "select",
			Required:          true,
			ValidationType:    "none",
			Value:             "MP3",
			ValidateOnActions: []string{"start"},
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
			ValidateOnActions: []string{"start"},
		},
		MP3Samplerate: FieldCfg{
			FieldType:         "select",
			Required:          true,
			DependsOn:         "targetFormat",
			ActiveWhen:        "MP3",
			ValidationType:    "none",
			Value:             "copy",
			ValidateOnActions: []string{"start"},
		},
		FLACBitdepth: FieldCfg{
			FieldType:         "select",
			Required:          true,
			DependsOn:         "targetFormat",
			ActiveWhen:        "FLAC",
			ValidationType:    "none",
			Value:             "copy",
			ValidateOnActions: []string{"start"},
		},
		FLACSamplerate: FieldCfg{
			FieldType:         "select",
			Required:          true,
			DependsOn:         "targetFormat",
			ActiveWhen:        "FLAC",
			ValidationType:    "none",
			Value:             "copy",
			ValidateOnActions: []string{"start"},
		},
		FLACCompression: FieldCfg{
			FieldType:         "select",
			Required:          true,
			DependsOn:         "targetFormat",
			ActiveWhen:        "FLAC",
			ValidationType:    "none",
			Value:             "12",
			ValidateOnActions: []string{"start"},
		},
		WAVBitdepth: FieldCfg{
			FieldType:         "select",
			Required:          true,
			DependsOn:         "targetFormat",
			ActiveWhen:        "WAV",
			ValidationType:    "none",
			Value:             "copy",
			ValidateOnActions: []string{"start"},
		},
		WAVSamplerate: FieldCfg{
			FieldType:         "select",
			Required:          true,
			DependsOn:         "targetFormat",
			ActiveWhen:        "WAV",
			ValidationType:    "none",
			Value:             "copy",
			ValidateOnActions: []string{"start"},
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
			ValidateOnActions: []string{"start"},
		},
		CustomDateFolders: FieldCfg{
			FieldType:         "folder",
			Required:          true,
			ValidationType:    "exists",
			Value:             "",
			ValidateOnActions: []string{"start"},
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
			ValidateOnActions: []string{"start"},
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
			ValidateOnActions: []string{"start"},
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
			ValidateOnActions: []string{"start"},
		},
		SourceFolder: FieldCfg{
			FieldType:         "folder",
			Required:          true,
			DependsOn:         "sourceType",
			ActiveWhen:        "folder",
			ValidationType:    "exists",
			Value:             "",
			ValidateOnActions: []string{"start"},
		},
		SourcePlaylist: FieldCfg{
			FieldType:         "playlist",
			Required:          true,
			DependsOn:         "sourceType",
			ActiveWhen:        "playlist",
			ValidationType:    "filled",
			Value:             "",
			ValidateOnActions: []string{"start"},
		},
		TargetType: FieldCfg{
			FieldType:         "select",
			Required:          true,
			ValidationType:    "none",
			Value:             "folder",
			ValidateOnActions: []string{"start"},
		},
		TargetFolder: FieldCfg{
			FieldType:         "folder",
			Required:          true,
			DependsOn:         "targetType",
			ActiveWhen:        "folder",
			ValidationType:    "exists | write",
			Value:             "",
			ValidateOnActions: []string{"start"},
		},
		TargetPlaylist: FieldCfg{
			FieldType:         "playlist",
			Required:          true,
			DependsOn:         "targetType",
			ActiveWhen:        "playlist",
			ValidationType:    "filled",
			Value:             "",
			ValidateOnActions: []string{"start"},
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
			ValidateOnActions: []string{"start"},
		},
		PlaylistID: FieldCfg{
			FieldType:         "playlist",
			Required:          true,
			ValidationType:    "filled",
			Value:             "",
			ValidateOnActions: []string{"start"},
		},
	}
}

// GetDefaultModuleCfg returns default configuration for any module by type
func GetDefaultModuleCfg(moduleType string) interface{} {
	switch moduleType {
	case "formatconverter":
		return GetDefaultFormatConverterCfg()
	case "datesmaster":
		return GetDefaultDatesMasterCfg()
	case "flacfixer":
		return GetDefaultFlacFixerCfg()
	case "dataduplicator":
		return GetDefaultDataDuplicatorCfg()
	case "formatupdater":
		return GetDefaultFormatUpdaterCfg()
	default:
		return nil
	}
}
