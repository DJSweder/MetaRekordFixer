//go:build darwin

// common/language_manager_darwin.go
// Package common implements shared functionality used across the MetaRekordFixer application.
// This file contains macOS-specific language detection functionality.

package common

import (
	"os"
	"strings"
)

// getSystemLanguage retrieves the system language on macOS by checking environment variables.
func getSystemLanguage() string {
	// On Unix-like systems, locale is often defined in environment variables.
	// The order of precedence is generally LC_ALL > LC_MESSAGES > LANG.
	for _, env := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if locale := os.Getenv(env); locale != "" {
			// Typically, the format is like 'en_US.UTF-8'. We want the 'en' part.
			return strings.Split(strings.ToLower(locale), "_")[0]
		}
	}
	return ""
}
