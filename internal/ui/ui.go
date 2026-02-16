// Package ui implements the graphical user interface for Lumpfish Media Catcher
// using the Fyne toolkit. It provides a cross-platform desktop application for
// downloading media using yt-dlp.
//
// The package includes:
//   - Main application window with URL input and format selection
//   - First-run setup wizard
//   - Progress reporting for downloads
//   - Context-based cancellation for long-running operations
package ui

import (
	"context"
	"fmt"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/lumpfishtech/mediacatcher/internal/config"
	"github.com/lumpfishtech/mediacatcher/internal/ytdlp"
)

// App represents the GUI application
type App struct {
	window       fyne.Window
	service      *ytdlp.Service
	config       *config.Config
	urlEntry     *widget.Entry
	formatSelect *widget.Select
	pathEntry    *widget.Entry
	statusLabel  *widget.Label
	versionLabel *widget.Label
	listBtn      *widget.Button
	downloadBtn  *widget.Button
	formats      []ytdlp.Format
	formatsMu    sync.Mutex
	ctx          context.Context
	cancelFunc   context.CancelFunc
}

// NewApp creates a new GUI application
func NewApp(w fyne.Window, service *ytdlp.Service, cfg *config.Config) *App {
	ctx, cancel := context.WithCancel(context.Background())
	app := &App{
		window:     w,
		service:    service,
		config:     cfg,
		ctx:        ctx,
		cancelFunc: cancel,
	}
	app.setupUI()

	// Fetch and display yt-dlp version in background
	go app.updateVersionLabel()

	return app
}

// Close cancels the app context and cleans up resources
func (a *App) Close() {
	if a.cancelFunc != nil {
		a.cancelFunc()
	}
}

// updateVersionLabel fetches and displays the yt-dlp version
func (a *App) updateVersionLabel() {
	version, err := a.service.GetVersion(a.ctx)
	if err == nil {
		versionText := fmt.Sprintf("yt-dlp v%s", version)
		if a.service.GetSource() == "system" {
			versionText += " (system)"
		}
		a.versionLabel.SetText(versionText)
	} else {
		a.versionLabel.SetText("yt-dlp: version unknown")
	}
}

func (a *App) setupUI() {
	// URL entry
	a.urlEntry = widget.NewEntry()
	a.urlEntry.SetPlaceHolder("Enter video URL (e.g., https://youtube.com/watch?v=...)")

	// List formats button
	a.listBtn = widget.NewButton("List Formats", a.handleListFormats)

	// Format selector
	a.formatSelect = widget.NewSelect([]string{}, nil)

	// Download path entry (use path from config)
	a.pathEntry = widget.NewEntry()
	a.pathEntry.SetText(a.config.DownloadPath)

	// Browse button
	browseBtn := widget.NewButton("Browse...", a.handleBrowse)

	// Download button
	a.downloadBtn = widget.NewButton("Download", a.handleDownload)

	// Set initial state for format selection and download button
	a.resetFormatSelection()

	// Status label
	a.statusLabel = widget.NewLabel("Ready")
	a.statusLabel.Truncation = fyne.TextTruncateClip

	// Version label
	a.versionLabel = widget.NewLabel("yt-dlp: checking...")

	// Layout
	urlBox := container.NewBorder(nil, nil, nil, a.listBtn, a.urlEntry)
	pathBox := container.NewBorder(nil, nil, widget.NewLabel("Save to:"), browseBtn, a.pathEntry)

	mainContent := container.NewVBox(
		widget.NewLabel("Video URL:"),
		urlBox,
		widget.NewLabel("Format:"),
		a.formatSelect,
		pathBox,
		a.downloadBtn,
		widget.NewSeparator(),
		a.statusLabel,
	)

	// Bottom bar with version info
	bottomBar := container.NewBorder(
		nil, nil,
		a.versionLabel, // left
		nil,            // right
		nil,            // center
	)

	// Final layout with bottom bar
	content := container.NewBorder(
		nil,        // top
		bottomBar,  // bottom
		nil, nil,   // left, right
		mainContent, // center
	)

	a.window.SetContent(container.NewPadded(content))
	a.window.Resize(fyne.NewSize(600, 400))
	// a.window.SetFixedSize(true)
}

func (a *App) handleListFormats() {
	url := a.urlEntry.Text
	if url == "" {
		a.showError("Please enter a URL")
		return
	}

	a.setLoading(true)
	a.statusLabel.SetText("Fetching formats...")

	go func() {
		formats, err := a.service.ListFormats(a.ctx, url)

		// Update UI on main thread
		a.window.Canvas().Content().Refresh()

		if err != nil {
			a.setLoading(false)
			a.showError(fmt.Sprintf("Failed to list formats: %v", err))
			a.statusLabel.SetText("Error listing formats")
			return
		}

		if len(formats) == 0 {
			a.setLoading(false)
			a.showError("No formats found")
			a.statusLabel.SetText("No formats found")
			return
		}

		a.formatsMu.Lock()
		a.formats = formats
		options := make([]string, len(formats))
		for i, f := range formats {
			options[i] = f.Description
		}
		a.formatsMu.Unlock()

		a.formatSelect.Options = options
		a.formatSelect.Enable()
		a.downloadBtn.Enable()
		a.setLoading(false)
		a.statusLabel.SetText(fmt.Sprintf("Found %d formats", len(formats)))
	}()
}

func (a *App) handleDownload() {
	if a.formatSelect.SelectedIndex() < 0 {
		a.showError("Please select a format")
		return
	}

	a.formatsMu.Lock()
	if a.formatSelect.SelectedIndex() >= len(a.formats) {
		a.formatsMu.Unlock()
		a.showError("Invalid format selection")
		return
	}
	selectedFormat := a.formats[a.formatSelect.SelectedIndex()]
	a.formatsMu.Unlock()

	url := a.urlEntry.Text
	outputPath := a.pathEntry.Text

	a.setLoading(true)
	a.statusLabel.SetText("Downloading...")

	go func() {
		progressFunc := func(msg string) {
			// Update status on main thread
			a.statusLabel.SetText(msg)
		}

		err := a.service.Download(a.ctx, url, selectedFormat.ID, outputPath, progressFunc)

		a.setLoading(false)

		if err != nil {
			a.showError(fmt.Sprintf("Download failed: %v", err))
			a.statusLabel.SetText("Download failed")
			return
		}

		a.resetFormatSelection()
		a.statusLabel.SetText("Download completed successfully!")
	}()
}

func (a *App) handleBrowse() {
	dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil || uri == nil {
			return
		}
		newPath := uri.Path()
		a.pathEntry.SetText(newPath)
		// Update config with new download path
		a.config.DownloadPath = newPath
	}, a.window)
}

func (a *App) showError(message string) {
	dialog.ShowError(fmt.Errorf("%s", message), a.window)
}

func (a *App) setLoading(loading bool) {
	if loading {
		a.listBtn.Disable()
		a.downloadBtn.Disable()
		a.urlEntry.Disable()
		a.formatSelect.Disable()
	} else {
		a.listBtn.Enable()
		a.urlEntry.Enable()
		a.formatsMu.Lock()
		hasFormats := len(a.formats) > 0
		a.formatsMu.Unlock()
		if hasFormats {
			a.formatSelect.Enable()
			a.downloadBtn.Enable()
		}
	}
}

func (a *App) resetFormatSelection() {
	a.formatsMu.Lock()
	a.formats = nil
	a.formatsMu.Unlock()
	a.formatSelect.Options = []string{}
	a.formatSelect.ClearSelected()
	a.formatSelect.PlaceHolder = "Select a format after listing"
	a.formatSelect.Disable()
	a.downloadBtn.Disable()
	a.formatSelect.Refresh()
}
