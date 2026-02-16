package logger

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestInit(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test.log")

	err := Init(logPath)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer Close()

	if Logger == nil {
		t.Error("Logger should not be nil after Init")
	}

	// Verify log file was created
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("Log file was not created at %s", logPath)
	}

	// Write a test message
	Info("Test message")

	// Read log file to verify content was written
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	if !strings.Contains(logContent, "Test message") {
		t.Error("Log file should contain test message")
	}
	if !strings.Contains(logContent, "[INFO]") {
		t.Error("Log file should contain [INFO] prefix")
	}
}

func TestLogLevels(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "levels.log")

	if err := Init(logPath); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer Close()

	Info("Info message")
	Error("Error message")
	Debug("Debug message")

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	tests := []struct {
		level   string
		message string
	}{
		{"[INFO]", "Info message"},
		{"[ERROR]", "Error message"},
		{"[DEBUG]", "Debug message"},
	}

	for _, tt := range tests {
		if !strings.Contains(logContent, tt.level) {
			t.Errorf("Log should contain %s level", tt.level)
		}
		if !strings.Contains(logContent, tt.message) {
			t.Errorf("Log should contain message: %s", tt.message)
		}
	}
}

func TestLoggerBeforeInit(t *testing.T) {
	// Reset logger state
	Logger = nil
	logFile = nil

	// These should not panic when Logger is nil
	Info("This should not panic")
	Error("This should not panic")
	Debug("This should not panic")
}

func TestClose(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "close.log")

	if err := Init(logPath); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	Info("Before close")
	Close()

	// Verify close message was written
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "Stopped") {
		t.Error("Log should contain stopped message")
	}

	// logFile should be nil after Close
	if logFile != nil {
		t.Error("logFile should be nil after Close")
	}
}

func TestConcurrentWrites(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "concurrent.log")

	if err := Init(logPath); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer Close()

	var wg sync.WaitGroup
	numGoroutines := 10
	messagesPerGoroutine := 10

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				Info("Goroutine %d message %d", id, j)
			}
		}(i)
	}

	wg.Wait()

	// Verify log file exists and has content
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Count INFO messages
	count := strings.Count(string(content), "[INFO]")
	expected := numGoroutines * messagesPerGoroutine

	// There are also startup messages, so count should be at least expected
	if count < expected {
		t.Errorf("Expected at least %d log messages, got %d", expected, count)
	}
}

func TestInitWithInvalidPath(t *testing.T) {
	// Try to init with a path that can't be created
	err := Init("/invalid/path/that/does/not/exist/test.log")
	if err == nil {
		t.Error("Expected error when initializing with invalid path")
		Close()
	}
}

func TestMultipleClose(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "multiclose.log")

	if err := Init(logPath); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Multiple Close() calls should not panic
	Close()
	Close()
	Close()
}

func TestFormatting(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "format.log")

	if err := Init(logPath); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer Close()

	// Test formatted output
	Info("Number: %d, String: %s", 42, "test")

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	if !strings.Contains(logContent, "Number: 42") {
		t.Error("Log should contain formatted number")
	}
	if !strings.Contains(logContent, "String: test") {
		t.Error("Log should contain formatted string")
	}
}
