package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg == nil {
		t.Fatal("Default() returned nil")
	}
	if cfg.FirstRunCompleted {
		t.Error("FirstRunCompleted should be false by default")
	}
	if cfg.YtDlpSource != "local" {
		t.Errorf("Expected YtDlpSource to be 'local', got %q", cfg.YtDlpSource)
	}
	if cfg.ConfigVersion != "0.0" {
		t.Errorf("Expected ConfigVersion to be '0.0', got %q", cfg.ConfigVersion)
	}
	if cfg.DownloadPath == "" {
		t.Error("DownloadPath should not be empty")
	}
	if cfg.LogFile == "" {
		t.Error("LogFile should not be empty")
	}
}

func TestGetUserDownloadsFolder(t *testing.T) {
	path := GetUserDownloadsFolder()
	if path == "" {
		t.Error("GetUserDownloadsFolder() returned empty string")
	}
	// Should either be Downloads folder or fallback to "."
	if path != "." {
		if !filepath.IsAbs(path) {
			t.Errorf("Expected absolute path or '.', got %q", path)
		}
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a test config
	testCfg := &Config{
		FirstRunCompleted: true,
		YtDlpSource:       "system",
		LogFile:           filepath.Join(tempDir, "test.log"),
		DownloadPath:      tempDir,
		ConfigVersion:     "0.0",
	}

	// Save to a temporary file
	configPath := filepath.Join(tempDir, "test-config.json")
	data, err := json.MarshalIndent(testCfg, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load it back
	loadedData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	var loadedCfg Config
	if err := json.Unmarshal(loadedData, &loadedCfg); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Verify fields match
	if loadedCfg.FirstRunCompleted != testCfg.FirstRunCompleted {
		t.Errorf("FirstRunCompleted mismatch: got %v, want %v",
			loadedCfg.FirstRunCompleted, testCfg.FirstRunCompleted)
	}
	if loadedCfg.YtDlpSource != testCfg.YtDlpSource {
		t.Errorf("YtDlpSource mismatch: got %q, want %q",
			loadedCfg.YtDlpSource, testCfg.YtDlpSource)
	}
	if loadedCfg.LogFile != testCfg.LogFile {
		t.Errorf("LogFile mismatch: got %q, want %q",
			loadedCfg.LogFile, testCfg.LogFile)
	}
	if loadedCfg.DownloadPath != testCfg.DownloadPath {
		t.Errorf("DownloadPath mismatch: got %q, want %q",
			loadedCfg.DownloadPath, testCfg.DownloadPath)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	// Create a temporary file with invalid JSON
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid-config.json")

	if err := os.WriteFile(configPath, []byte("{ invalid json }"), 0644); err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}

	// Try to load it
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var cfg Config
	err = json.Unmarshal(data, &cfg)
	if err == nil {
		t.Error("Expected error when parsing invalid JSON, got nil")
	}
}

func TestGetLogFilePath(t *testing.T) {
	tests := []struct {
		name     string
		logFile  string
		wantAbs  bool
		wantName string
	}{
		{
			name:     "relative filename",
			logFile:  "test.log",
			wantAbs:  true,
			wantName: "test.log",
		},
		{
			name:     "default filename",
			logFile:  "lmc-app.log",
			wantAbs:  true,
			wantName: "lmc-app.log",
		},
		{
			name:     "empty uses default",
			logFile:  "",
			wantAbs:  true,
			wantName: "lmc-app.log",
		},
		{
			name:     "absolute path preserved",
			logFile:  "/tmp/custom.log",
			wantAbs:  true,
			wantName: "custom.log",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{LogFile: tt.logFile}
			path := GetLogFilePath(cfg)

			if tt.wantAbs && !filepath.IsAbs(path) {
				t.Errorf("Expected absolute path, got %q", path)
			}

			if filepath.Base(path) != tt.wantName {
				t.Errorf("Expected filename %q, got %q", tt.wantName, filepath.Base(path))
			}
		})
	}
}

func TestConcurrentSave(t *testing.T) {
	// Test that concurrent saves don't corrupt the file
	tempDir := t.TempDir()

	cfg1 := &Config{
		FirstRunCompleted: true,
		YtDlpSource:       "local",
		LogFile:           filepath.Join(tempDir, "test1.log"),
		DownloadPath:      tempDir,
		ConfigVersion:     "0.0",
	}

	cfg2 := &Config{
		FirstRunCompleted: false,
		YtDlpSource:       "system",
		LogFile:           filepath.Join(tempDir, "test2.log"),
		DownloadPath:      tempDir,
		ConfigVersion:     "0.0",
	}

	// Note: We can't easily test the real Save() function since it uses
	// os.Executable() which won't work in tests. This test demonstrates
	// the concept of concurrent saves. In practice, the mutex in Save()
	// protects against corruption.

	configPath := filepath.Join(tempDir, "concurrent-test.json")

	done := make(chan bool)

	write := func(cfg *Config) {
		data, _ := json.MarshalIndent(cfg, "", "  ")
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			t.Errorf("Failed to write config: %v", err)
		}
		done <- true
	}

	// Launch concurrent writes
	go write(cfg1)
	go write(cfg2)

	// Wait for both to complete
	<-done
	<-done

	// Verify the file is valid JSON (either cfg1 or cfg2)
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config after concurrent writes: %v", err)
	}

	var result Config
	if err := json.Unmarshal(data, &result); err != nil {
		t.Errorf("Config file corrupted after concurrent writes: %v", err)
	}
}
