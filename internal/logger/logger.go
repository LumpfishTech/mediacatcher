// Package logger provides a simple logging facility that writes to both
// stdout and a log file. It uses a global logger instance with thread-safe
// initialization and cleanup.
//
// Usage:
//   logger.Init("/path/to/logfile.log")
//   defer logger.Close()
//   logger.Info("Application started")
//   logger.Error("Something went wrong: %v", err)
package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

var (
	// Logger is the global logger instance
	Logger *log.Logger
	// logFile is the log file handle
	logFile *os.File
	// mu protects logger initialization and cleanup
	mu sync.Mutex
)

// Init initializes the logger to write to both stdout and a log file
func Init(logPath string) error {
	mu.Lock()
	defer mu.Unlock()

	// Get executable path for logging
	execPath, err := os.Executable()
	if err != nil {
		execPath = "unknown"
	}

	// Create log file (overwrite if exists)
	logFile, err = os.Create(logPath)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

	// Create multi-writer for both stdout and file
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	// Initialize logger
	Logger = log.New(multiWriter, "", log.LstdFlags|log.Lshortfile)

	Logger.Printf("=== Lumpfish Media Catcher Started ===")
	Logger.Printf("Log file: %s", logPath)
	Logger.Printf("Executable: %s", execPath)

	return nil
}

// Close closes the log file
func Close() {
	mu.Lock()
	defer mu.Unlock()

	if logFile != nil {
		Logger.Printf("=== Lumpfish Media Catcher Stopped ===")
		logFile.Close()
		logFile = nil
	}
}

// Info logs an info message
func Info(format string, v ...interface{}) {
	if Logger != nil {
		Logger.Printf("[INFO] "+format, v...)
	}
}

// Error logs an error message
func Error(format string, v ...interface{}) {
	if Logger != nil {
		Logger.Printf("[ERROR] "+format, v...)
	}
}

// Debug logs a debug message
func Debug(format string, v ...interface{}) {
	if Logger != nil {
		Logger.Printf("[DEBUG] "+format, v...)
	}
}

// Fatal logs a fatal message and exits
func Fatal(format string, v ...interface{}) {
	if Logger != nil {
		Logger.Printf("[FATAL] "+format, v...)
	}
	os.Exit(1)
}
