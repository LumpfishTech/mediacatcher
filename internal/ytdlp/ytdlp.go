// Package ytdlp provides a Go wrapper for the yt-dlp command-line tool.
// It handles automatic detection, downloading, and management of yt-dlp,
// supporting both system-installed and locally downloaded binaries.
//
// The package provides functionality for:
//   - Listing available video/audio formats from URLs
//   - Downloading media in specified formats
//   - Automatic yt-dlp installation and updates
//   - Version management and caching
//   - Progress reporting for long-running operations
package ytdlp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

var (
	// ErrYtDlpNotFound is returned when yt-dlp is not installed
	ErrYtDlpNotFound = errors.New("yt-dlp not found in PATH")
	// ErrInvalidURL is returned when the URL is invalid
	ErrInvalidURL = errors.New("invalid URL")
	// ErrDownloadFailed is returned when downloading yt-dlp fails
	ErrDownloadFailed = errors.New("failed to download yt-dlp")
	// ErrUpdateNotSupported is returned when trying to update system yt-dlp
	ErrUpdateNotSupported = errors.New("cannot update system yt-dlp")
	// ErrUpdateFailed is returned when yt-dlp update fails
	ErrUpdateFailed = errors.New("yt-dlp update failed")
)

// Format represents a video format option
type Format struct {
	ID          string
	Extension   string
	Resolution  string
	Description string
}

// Service handles yt-dlp operations
type Service struct {
	ytdlpPath  string
	source     string // "system" or "local"
	version    string // cached version string
	versionMu  sync.RWMutex
	execDir    string // directory containing the app executable
}

// NewService creates a new yt-dlp service
// preferredSource can be "system" or "local"
func NewService(preferredSource string) (*Service, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}
	execDir := filepath.Dir(execPath)

	var ytdlpPath string
	var source string

	// Priority 1: If user prefers system, try PATH first
	if preferredSource == "system" {
		path, err := exec.LookPath("yt-dlp")
		if err == nil {
			ytdlpPath = path
			source = "system"
		} else {
			return nil, fmt.Errorf("system yt-dlp not found in PATH")
		}
	} else {
		// Priority 2: Check for local copy
		localPath := getLocalYtDlpPath(execDir)
		if _, err := os.Stat(localPath); err == nil {
			ytdlpPath = localPath
			source = "local"
		} else {
			// Priority 3: Try system PATH as fallback
			path, err := exec.LookPath("yt-dlp")
			if err == nil {
				ytdlpPath = path
				source = "system"
			} else {
				return nil, ErrYtDlpNotFound
			}
		}
	}

	return &Service{
		ytdlpPath: ytdlpPath,
		source:    source,
		execDir:   execDir,
	}, nil
}

// getLocalYtDlpPath returns the path to the local yt-dlp binary
func getLocalYtDlpPath(execDir string) string {
	filename := "yt-dlp"
	if runtime.GOOS == "windows" {
		filename = "yt-dlp.exe"
	}
	return filepath.Join(execDir, filename)
}

// IsLocal returns true if using a local copy of yt-dlp
func (s *Service) IsLocal() bool {
	return s.source == "local"
}

// GetVersion retrieves the yt-dlp version
func (s *Service) GetVersion(ctx context.Context) (string, error) {
	// Check cache with read lock
	s.versionMu.RLock()
	if s.version != "" {
		cached := s.version
		s.versionMu.RUnlock()
		return cached, nil
	}
	s.versionMu.RUnlock()

	// Fetch version
	cmd := exec.CommandContext(ctx, s.ytdlpPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get version: %w", err)
	}

	version := strings.TrimSpace(string(output))

	// Update cache with write lock
	s.versionMu.Lock()
	s.version = version
	s.versionMu.Unlock()

	return version, nil
}

// GetSource returns the source of yt-dlp ("system" or "local")
func (s *Service) GetSource() string {
	return s.source
}

// DownloadYtDlp downloads yt-dlp binary to the local directory
func DownloadYtDlp(ctx context.Context, progress func(string)) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execDir := filepath.Dir(execPath)

	// Determine download URL based on OS
	const baseURL = "https://github.com/yt-dlp/yt-dlp/releases/latest/download/"
	var filename string
	switch runtime.GOOS {
	case "windows":
		filename = "yt-dlp.exe"
	case "darwin":
		filename = "yt-dlp_macos"
	default:
		filename = "yt-dlp"
	}

	downloadURL := baseURL + filename
	if progress != nil {
		progress(fmt.Sprintf("Downloading from %s...", downloadURL))
	}

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDownloadFailed, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDownloadFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: HTTP %d", ErrDownloadFailed, resp.StatusCode)
	}

	// Create temporary file
	outputPath := getLocalYtDlpPath(execDir)
	tempPath := outputPath + ".tmp"
	outFile, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDownloadFailed, err)
	}

	// Download with progress
	totalBytes := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := outFile.Write(buf[:n]); writeErr != nil {
				outFile.Close()
				os.Remove(tempPath)
				return fmt.Errorf("%w: %v", ErrDownloadFailed, writeErr)
			}
			downloaded += int64(n)
			if progress != nil && totalBytes > 0 {
				percentage := float64(downloaded) / float64(totalBytes) * 100
				progress(fmt.Sprintf("Downloaded %.1f%%", percentage))
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			outFile.Close()
			os.Remove(tempPath)
			return fmt.Errorf("%w: %v", ErrDownloadFailed, err)
		}
	}

	// Close file before chmod/rename (defer will also close, but this ensures it's closed now)
	if err := outFile.Close(); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("%w: failed to close file: %v", ErrDownloadFailed, err)
	}

	// Make executable on Unix-like systems
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tempPath, 0755); err != nil {
			os.Remove(tempPath)
			return fmt.Errorf("%w: failed to make executable: %v", ErrDownloadFailed, err)
		}
	}

	// Move to final location
	if err := os.Rename(tempPath, outputPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("%w: %v", ErrDownloadFailed, err)
	}

	// Notify user of verification step
	if progress != nil {
		progress("Verifying download... please wait")
	}

	// Verify the download by checking version
	cmd := exec.CommandContext(ctx, outputPath, "--version")
	if err := cmd.Run(); err != nil {
		os.Remove(outputPath)
		return fmt.Errorf("%w: downloaded binary verification failed", ErrDownloadFailed)
	}

	if progress != nil {
		progress("Download completed successfully!")
	}

	return nil
}

// UpdateYtDlp updates the local yt-dlp binary
func (s *Service) UpdateYtDlp(ctx context.Context, progress func(string)) error {
	if s.source != "local" {
		return ErrUpdateNotSupported
	}

	if progress != nil {
		progress("Checking for updates...")
	}

	cmd := exec.CommandContext(ctx, s.ytdlpPath, "-U")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %v (output: %s)", ErrUpdateFailed, err, string(output))
	}

	outputStr := string(output)
	if progress != nil {
		progress(outputStr)
	}

	// Clear cached version so it's re-fetched
	s.versionMu.Lock()
	s.version = ""
	s.versionMu.Unlock()

	return nil
}

// ListFormats retrieves available formats for a video URL
// Automatically attempts to update yt-dlp and retry on failure if using local copy
func (s *Service) ListFormats(ctx context.Context, url string) ([]Format, error) {
	if url == "" {
		return nil, ErrInvalidURL
	}

	formats, err := s.listFormats(ctx, url)
	if err != nil && s.IsLocal() {
		// Try updating and retry once
		updateErr := s.UpdateYtDlp(ctx, nil)
		if updateErr == nil {
			// Retry after update
			formats, err = s.listFormats(ctx, url)
		}
	}

	return formats, err
}

// listFormats is the internal implementation without retry logic
func (s *Service) listFormats(ctx context.Context, url string) ([]Format, error) {
	cmd := exec.CommandContext(ctx, s.ytdlpPath, "-F", url)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list formats: %w (output: %s)", err, string(output))
	}

	return parseFormats(string(output)), nil
}

// Download downloads a video with the specified format
// Automatically attempts to update yt-dlp and retry on failure if using local copy
func (s *Service) Download(ctx context.Context, url, formatID, outputPath string, progress func(string)) error {
	if url == "" {
		return ErrInvalidURL
	}

	err := s.download(ctx, url, formatID, outputPath, progress)
	if err != nil && s.IsLocal() {
		// Try updating and retry once
		if progress != nil {
			progress("Updating yt-dlp...")
		}
		updateErr := s.UpdateYtDlp(ctx, progress)
		if updateErr == nil {
			// Retry after update
			err = s.download(ctx, url, formatID, outputPath, progress)
		}
	}

	return err
}

// download is the internal implementation without retry logic
func (s *Service) download(ctx context.Context, url, formatID, outputPath string, progress func(string)) error {
	args := []string{"-f", formatID}
	if outputPath != "" {
		args = append(args, "-P", outputPath)
	}
	args = append(args, url)

	cmd := exec.CommandContext(ctx, s.ytdlpPath, args...)

	// Capture stdout and stderr for progress updates
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start download: %w", err)
	}

	// Read progress from both stdout and stderr
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if progress != nil {
				progress(scanner.Text())
			}
		}
		// Check for scanner errors
		if err := scanner.Err(); err != nil && progress != nil {
			progress(fmt.Sprintf("Error reading stdout: %v", err))
		}
	}()

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			if progress != nil {
				progress(scanner.Text())
			}
		}
		// Check for scanner errors
		if err := scanner.Err(); err != nil && progress != nil {
			progress(fmt.Sprintf("Error reading stderr: %v", err))
		}
	}()

	// Wait for command to finish
	cmdErr := cmd.Wait()

	// Wait for all output to be processed
	wg.Wait()

	if cmdErr != nil {
		return fmt.Errorf("download failed: %w", cmdErr)
	}

	return nil
}

// parseFormats parses the output of yt-dlp -F command
func parseFormats(output string) []Format {
	var formats []Format
	lines := strings.Split(output, "\n")

	// Regex to match format lines (ID ext resolution note)
	// Example: "22              mp4   1280x720    720p  1463k , avc1.64001F, 30fps, mp4a.40.2"
	formatRegex := regexp.MustCompile(`^(\S+)\s+(\S+)\s+(\S+)?\s*(.*)$`)

	foundHeader := false
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip until we find the format list header
		if strings.HasPrefix(line, "ID") && strings.Contains(line, "EXT") {
			foundHeader = true
			continue
		}

		if !foundHeader || line == "" {
			continue
		}

		// Skip separator lines
		if strings.HasPrefix(line, "---") {
			continue
		}

		matches := formatRegex.FindStringSubmatch(line)
		if len(matches) >= 3 {
			id := matches[1]
			ext := matches[2]
			resolution := ""
			if len(matches) > 3 {
				resolution = matches[3]
			}
			description := ""
			if len(matches) > 4 {
				description = strings.TrimSpace(matches[4])
			}

			// Create readable description
			fullDesc := fmt.Sprintf("%s - %s", id, ext)
			if resolution != "" && resolution != "audio" && resolution != "~" {
				fullDesc += fmt.Sprintf(" [%s]", resolution)
			}
			if description != "" {
				// Truncate long descriptions
				if len(description) > 50 {
					description = description[:47] + "..."
				}
				fullDesc += " - " + description
			}

			formats = append(formats, Format{
				ID:          id,
				Extension:   ext,
				Resolution:  resolution,
				Description: fullDesc,
			})
		}
	}

	return formats
}
