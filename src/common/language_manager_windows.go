//go:build windows

// common/language_manager_windows.go

package common

import (
	"strings"
	"syscall"
	"unsafe"
)

// getSystemLanguage retrieves the system language on Windows via kernel32.dll calls.
func getSystemLanguage() string {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getUserDefaultLocaleName := kernel32.NewProc("GetUserDefaultLocaleName")

	// Buffer to store the locale name
	localeName := make([]uint16, 85) // LOCALE_NAME_MAX_LENGTH is 85
	getUserDefaultLocaleName.Call(uintptr(unsafe.Pointer(&localeName[0])), uintptr(len(localeName)))
	return strings.ToLower(syscall.UTF16ToString(localeName))
}
