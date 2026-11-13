package common

import (
	"os"
	"path/filepath"
	"runtime"
)

// Platform holds OS detection information.
type PlatformInfo struct {
	OS   string // "windows", "linux", "darwin"
	Arch string // "amd64", "arm64", "386"
}

// DetectPlatform returns the current OS and architecture.
func DetectPlatform() PlatformInfo {
	return PlatformInfo{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}
}

// IsWindows returns true if running on Windows.
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// IsAdminRequired returns true if the current operation typically needs
// elevated privileges on Windows.
func IsAdminRequired() bool {
	return true
}

// IsAdmin returns true if the process has administrative privileges.
func IsAdmin() bool {
	// Windows check via shell command
	cmd := HiddenCommand("net", "session")
	err := cmd.Run()
	return err == nil
}

// GetBaseDir returns the application base directory. It prefers a local 'data'
// directory for portable mode, but falls back to the OS-standard UserConfigDir.
func GetBaseDir() string {
	// Check for portable mode first
	if _, err := os.Stat("data"); err == nil {
		return "."
	}

	// Fallback to standard OS directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "." // Last resort
	}
	return filepath.Join(configDir, "winops")
}

// ConfigDir returns the application data directory path.
func ConfigDir() (string, error) {
	base := GetBaseDir()
	if base == "." {
		return "data", nil
	}
	return filepath.Join(base, "data"), nil
}

// LogsDir returns the application logs directory path.
// Logs are stored alongside the database under the app data directory.
func LogsDir() (string, error) {
	base := GetBaseDir()
	if base == "." {
		return "data", nil
	}
	return filepath.Join(base, "data"), nil
}

// IsOnboarded checks if the onboarding marker file exists in the data dir.
func IsOnboarded() bool {
	dir, _ := ConfigDir()
	_, err := os.Stat(filepath.Join(dir, ".onboarded"))
	return err == nil
}

// MarkOnboarded creates the onboarding marker file.
func MarkOnboarded() error {
	dir, _ := ConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, ".onboarded"), []byte{}, 0600)
}

// ClearOnboarded removes the onboarding marker.
func ClearOnboarded() error {
	dir, _ := ConfigDir()
	return os.Remove(filepath.Join(dir, ".onboarded"))
}
