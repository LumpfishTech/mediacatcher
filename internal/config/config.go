// Package config manages application configuration persistence.
// Configuration is stored in JSON format next to the executable.
//
// The package handles:
//   - Loading and saving user preferences
//   - First-run detection
//   - Default configuration values
//   - Thread-safe configuration file access
package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
)

var (
	// saveMu protects concurrent Save operations
	saveMu sync.Mutex
)

// Config represents the application configuration
type Config struct {
	FirstRunCompleted bool   `json:"first_run_completed"`
	YtDlpSource       string `json:"ytdlp_source"`     // "system" or "local"
	LogFile           string `json:"log_file"`         // log filename (placed next to executable)
	DownloadPath      string `json:"download_path"`    // media download directory
	ConfigVersion     string `json:"config_version"`   // config schema version (e.g., "1.0", "1.1") - major for breaking changes, minor for new fields
}

// GetConfigPath returns the configuration file path (next to executable)
func GetConfigPath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}

	execDir := filepath.Dir(execPath)
	configFile := filepath.Join(execDir, "lmc-config.json")

	return configFile, nil
}

// Load loads the configuration from disk
// Returns default config if file doesn't exist
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		// Can't determine config path, return default
		return Default(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Config doesn't exist, return default
			return Default(), nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		// Invalid JSON, treat as first run
		log.Printf("Warning: Failed to parse config file, using defaults: %v", err)
		return Default(), nil
	}

	return &cfg, nil
}

// Save saves the configuration to disk
func Save(cfg *Config) error {
	saveMu.Lock()
	defer saveMu.Unlock()

	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// GetUserDownloadsFolder returns the user's Downloads folder path in a cross-platform way
func GetUserDownloadsFolder() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if we can't get home
		return "."
	}

	// Downloads folder is typically in the same location on all platforms
	return filepath.Join(homeDir, "Downloads")
}

// GetDefaultLogFileName returns the default log filename (without path)
func GetDefaultLogFileName() string {
	return "lmc-app.log"
}

// GetLogFilePath returns the full path to the log file
// If cfg.LogFile is just a filename, it's placed next to the executable
// If it's already a full path (for backward compatibility), it's used as-is
func GetLogFilePath(cfg *Config) string {
	logFile := cfg.LogFile
	if logFile == "" {
		logFile = GetDefaultLogFileName()
	}

	// Check if it's already an absolute path
	if filepath.IsAbs(logFile) {
		return logFile
	}

	// It's just a filename, place it next to the executable
	execPath, err := os.Executable()
	if err != nil {
		// Fallback to current directory if we can't get executable path
		return logFile
	}

	execDir := filepath.Dir(execPath)
	return filepath.Join(execDir, logFile)
}

// Default returns the default configuration
func Default() *Config {
	return &Config{
		FirstRunCompleted: false,
		YtDlpSource:       "local",
		LogFile:           GetDefaultLogFileName(),
		DownloadPath:      GetUserDownloadsFolder(),
		ConfigVersion:     "0.0",
	}
}
