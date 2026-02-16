package ui

import (
	"context"
	"fmt"
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/lumpfishtech/mediacatcher/internal/logger"
	"github.com/lumpfishtech/mediacatcher/internal/ytdlp"
)

// Wizard handles the first-run setup
type Wizard struct {
	window         fyne.Window
	onComplete     func(source string)
	onCancel       func()
	systemDetected bool
	selectedSource string
	ctx            context.Context
	cancelFunc     context.CancelFunc
}

// NewWizard creates a new first-run wizard
func NewWizard(w fyne.Window, systemDetected bool, onComplete func(string), onCancel func()) *Wizard {
	defaultSource := "local"
	ctx, cancel := context.WithCancel(context.Background())
	return &Wizard{
		window:         w,
		onComplete:     onComplete,
		onCancel:       onCancel,
		systemDetected: systemDetected,
		selectedSource: defaultSource,
		ctx:            ctx,
		cancelFunc:     cancel,
	}
}

// Show displays the wizard dialog
func (wz *Wizard) Show() {
	logger.Info("Wizard.Show() called, systemDetected=%v", wz.systemDetected)

	// Create title
	title := widget.NewLabelWithStyle(
		"Welcome to Lumpfish Media Catcher",
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	// Create explanation text
	explanation := widget.NewLabel(
		"Lumpfish Media Catcher is a frontend for yt-dlp, a third-party tool for downloading media from the web.",
	)
	explanation.Wrapping = fyne.TextWrapWord

	// Create hyperlink
	ytdlpURL, _ := url.Parse("https://github.com/yt-dlp/yt-dlp")
	link := widget.NewHyperlink("Learn more about yt-dlp", ytdlpURL)

	// Create radio buttons if system yt-dlp detected
	var radioGroup *widget.RadioGroup
	if wz.systemDetected {
		detectedLabel := widget.NewLabel("yt-dlp detected on your system")
		detectedLabel.TextStyle = fyne.TextStyle{Bold: true}

		radioGroup = widget.NewRadioGroup(
			[]string{
				"Use system yt-dlp (no auto-update)",
				"Download local copy (recommended)",
			},
			func(selected string) {
				if selected == "Use system yt-dlp (no auto-update)" {
					wz.selectedSource = "system"
				} else {
					wz.selectedSource = "local"
				}
			},
		)
		radioGroup.SetSelected("Download local copy (recommended)")
		wz.selectedSource = "local"
	}

	// Create disclaimer
	disclaimer := widget.NewLabel(
		"Important: You are responsible for using this software in compliance with applicable laws and terms of service.",
	)
	disclaimer.Wrapping = fyne.TextWrapWord
	disclaimer.TextStyle = fyne.TextStyle{Italic: true}

	// Create content container
	content := container.NewVBox(
		title,
		widget.NewSeparator(),
		explanation,
		link,
	)

	if radioGroup != nil {
		content.Add(widget.NewSeparator())
		content.Add(radioGroup)
	}

	content.Add(widget.NewSeparator())
	content.Add(disclaimer)

	// Wrap content in a scrollable container for proper resizing
	scrollContent := container.NewVScroll(content)
	scrollContent.SetMinSize(fyne.NewSize(500, 300))

	// Create buttons
	cancelBtn := widget.NewButton("Cancel", func() {
		logger.Info("Wizard: Cancel button clicked")
		if wz.cancelFunc != nil {
			wz.cancelFunc()
		}
		if wz.onCancel != nil {
			wz.onCancel()
		}
	})

	continueBtn := widget.NewButton("Continue", func() {
		logger.Info("Wizard: Continue button clicked, selectedSource=%s", wz.selectedSource)
		// If local copy selected or no system yt-dlp, download it
		if wz.selectedSource == "local" {
			logger.Info("Wizard: Starting yt-dlp download...")
			wz.downloadYtDlp()
		} else {
			logger.Info("Wizard: Using system yt-dlp, completing immediately")
			// Using system yt-dlp, complete immediately
			if wz.onComplete != nil {
				wz.onComplete(wz.selectedSource)
			}
		}
	})

	// Create buttons container with proper spacing
	buttons := container.NewGridWithColumns(2, cancelBtn, continueBtn)

	// Use border layout to keep buttons at bottom and content scrollable
	finalContent := container.NewBorder(
		nil,      // top
		buttons,  // bottom (always visible)
		nil, nil, // left, right
		scrollContent, // center (scrollable)
	)

	// Show dialog - using "First Run Setup" as dismiss label to avoid extra empty button
	logger.Info("Wizard: Creating and showing dialog")
	wz.window.SetContent(finalContent)
	wz.window.Resize(fyne.NewSize(550, 450))
	wz.window.CenterOnScreen()
	logger.Info("Wizard: Content set with size 550x450")
}

// downloadYtDlp downloads yt-dlp and shows progress
func (wz *Wizard) downloadYtDlp() {
	logger.Info("downloadYtDlp: Starting download process")

	// Create progress bar and label
	progressBar := widget.NewProgressBar()
	progressLabel := widget.NewLabel("Preparing download...")

	// Create title label
	titleLabel := widget.NewLabelWithStyle("Downloading yt-dlp", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	content := container.NewVBox(
		titleLabel,
		widget.NewSeparator(),
		progressBar,
		progressLabel,
	)

	// Add padding around the content
	paddedContent := container.NewPadded(content)

	logger.Info("downloadYtDlp: Creating progress modal")
	progressModal := widget.NewModalPopUp(paddedContent, wz.window.Canvas())
	progressModal.Resize(fyne.NewSize(400, 150))
	progressModal.Show()
	logger.Info("downloadYtDlp: Progress modal shown with size 400x150")

	// Download in background
	go func() {
		logger.Info("downloadYtDlp: Download goroutine started")

		err := ytdlp.DownloadYtDlp(wz.ctx, func(msg string) {
			logger.Debug("downloadYtDlp: Progress - %s", msg)
			progressLabel.SetText(msg)
			progressLabel.Refresh()

			// Try to parse percentage from message
			var percentage float64
			if n, _ := fmt.Sscanf(msg, "Downloaded %f%%", &percentage); n == 1 {
				progressBar.SetValue(percentage / 100.0)
			}
		})

		logger.Info("downloadYtDlp: Download completed, err=%v", err)
		progressModal.Hide()

		if err != nil {
			logger.Error("downloadYtDlp: Download failed - %v", err)
			// Show error dialog
			errorDialog := dialog.NewError(
				fmt.Errorf("Failed to download yt-dlp: %v", err),
				wz.window,
			)

			// Add retry button
			errorDialog.SetOnClosed(func() {
				retryDialog := dialog.NewConfirm(
					"Download Failed",
					"Would you like to retry downloading yt-dlp?",
					func(retry bool) {
						if retry {
							logger.Info("downloadYtDlp: User chose to retry")
							wz.downloadYtDlp()
						} else {
							logger.Info("downloadYtDlp: User cancelled after error")
							if wz.onCancel != nil {
								wz.onCancel()
							}
						}
					},
					wz.window,
				)
				retryDialog.Show()
			})

			errorDialog.Show()
			return
		}

		// Success
		logger.Info("downloadYtDlp: Download successful, calling onComplete callback")
		if wz.onComplete != nil {
			wz.onComplete(wz.selectedSource)
		}
	}()
}
