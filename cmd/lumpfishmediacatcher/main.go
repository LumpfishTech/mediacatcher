package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"

	"github.com/lumpfishtech/mediacatcher/internal/config"
	"github.com/lumpfishtech/mediacatcher/internal/logger"
	"github.com/lumpfishtech/mediacatcher/internal/ui"
	"github.com/lumpfishtech/mediacatcher/internal/ytdlp"
)

// Application version - overridden by -ldflags during build
var version = "dev"

// Command-line flags
var showVersion = flag.Bool("version", false, "Show application version and exit")

func main() {
	// Parse command-line flags
	flag.Parse()

	// Handle --version flag
	if *showVersion {
		fmt.Printf("LumpfishMediaCatcher %s\n", version)
		os.Exit(0)
	}

	// Load configuration first (before logger, so we know where to log)
	cfg, err := config.Load()
	if err != nil {
		cfg = config.Default()
	}

	// Initialize logger with resolved path from config
	logPath := config.GetLogFilePath(cfg)
	if err := logger.Init(logPath); err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	defer logger.Close()

	logger.Info("LumpfishMediaCatcher version: %s", version)
	logger.Info("Application starting...")
	logger.Info("Configuration loaded: FirstRunCompleted=%v, YtDlpSource=%s, DownloadPath=%s",
		cfg.FirstRunCompleted, cfg.YtDlpSource, cfg.DownloadPath)

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		logger.Info("Received signal %v, shutting down gracefully...", sig)
		logger.Info("Saving configuration...")
		if err := config.Save(cfg); err != nil {
			logger.Error("Failed to save config on shutdown: %v", err)
		}
		logger.Close()
		os.Exit(0)
	}()

	// Create Fyne application
	logger.Info("Creating Fyne application...")
	a := app.New()
	w := a.NewWindow("Lumpfish Media Catcher")
	logger.Info("Window created")

	// Check if this is the first run
	if !cfg.FirstRunCompleted {
		logger.Info("First run detected, showing setup wizard...")
		// Set a reasonable window size before showing wizard
		w.Resize(fyne.NewSize(600, 500))
		w.CenterOnScreen()
		runFirstTimeSetup(w, cfg)
		return
	}

	// Try to create service with configured preference
	logger.Info("Creating ytdlp service with source: %s", cfg.YtDlpSource)
	service, err := ytdlp.NewService(cfg.YtDlpSource)
	if err != nil {
		logger.Error("Failed to create ytdlp service: %v", err)
		handleMissingYtDlp(w, cfg, err)
		return
	}
	logger.Info("Service created successfully")

	// Create and setup UI
	setupMainUI(w, service, cfg)
	logger.Info("Main UI setup complete, showing window...")

	// Show and run
	w.ShowAndRun()
	logger.Info("Application exited normally")
}

// setupMainUI creates the main UI and configures the window close handler
func setupMainUI(w fyne.Window, service *ytdlp.Service, cfg *config.Config) {
	logger.Info("Creating main UI...")
	app := ui.NewApp(w, service, cfg)
	logger.Info("UI created")

	// Save config when window closes
	w.SetOnClosed(func() {
		logger.Info("Window closing, cleaning up...")
		app.Close()
		logger.Info("Saving configuration...")
		if err := config.Save(cfg); err != nil {
			logger.Error("Failed to save config: %v", err)
		} else {
			logger.Info("Configuration saved successfully")
		}
	})
}

// runFirstTimeSetup shows the wizard and handles first-run configuration
func runFirstTimeSetup(w fyne.Window, cfg *config.Config) {
	// Detect if system yt-dlp exists
	logger.Info("Detecting system yt-dlp...")
	_, err := exec.LookPath("yt-dlp")
	systemDetected := (err == nil)
	logger.Info("System yt-dlp detected: %v", systemDetected)

	logger.Info("Creating first-run wizard...")
	wizard := ui.NewWizard(
		w,
		systemDetected,
		func(source string) {
			// User completed wizard
			logger.Info("Wizard completed, user selected: %s", source)
			cfg.YtDlpSource = source
			cfg.FirstRunCompleted = true

			logger.Info("Saving configuration...")
			if err := config.Save(cfg); err != nil {
				logger.Error("Failed to save config: %v", err)
				dialog.ShowInformation(
					"Warning",
					"Could not save preferences. Settings will not persist.",
					w,
				)
			} else {
				logger.Info("Configuration saved successfully")
			}

			// Create service and continue
			logger.Info("Creating ytdlp service...")
			service, err := ytdlp.NewService(cfg.YtDlpSource)
			if err != nil {
				logger.Error("Failed to create service: %v", err)
				dialog.ShowError(
					fmt.Errorf("Failed to initialize yt-dlp: %v", err),
					w,
				)
				w.Close()
				return
			}
			logger.Info("Service created successfully")

			// Create UI
			setupMainUI(w, service, cfg)
		},
		func() {
			// User cancelled wizard
			logger.Info("User cancelled wizard")
			w.Close()
		},
	)

	logger.Info("Showing wizard...")
	wizard.Show()
	logger.Info("Running window...")
	w.ShowAndRun()
	logger.Info("First-run setup completed")
}

// handleMissingYtDlp handles the case where yt-dlp is not found after initial setup
func handleMissingYtDlp(w fyne.Window, cfg *config.Config, err error) {
	message := fmt.Sprintf("yt-dlp not found: %v", err)
	logger.Error("yt-dlp not found: %v", err)

	if cfg.YtDlpSource == "local" {
		logger.Info("Offering to re-download local copy...")
		// Offer to re-download local copy
		dialog.ShowConfirm(
			"yt-dlp Not Found",
			message+"\n\nWould you like to download a local copy now?",
			func(download bool) {
				if download {
					logger.Info("User chose to re-download")
					// Re-run first-time setup
					cfg.FirstRunCompleted = false
					runFirstTimeSetup(w, cfg)
				} else {
					logger.Info("User declined re-download")
					w.Close()
				}
			},
			w,
		)
	} else {
		logger.Info("System yt-dlp expected but not found, showing error")
		// System yt-dlp expected but not found
		dialog.ShowError(
			fmt.Errorf("%s\n\nPlease install yt-dlp on your system:\n- macOS: brew install yt-dlp\n- Linux: pip install yt-dlp or use your package manager\n- Windows: Download from https://github.com/yt-dlp/yt-dlp/releases", message),
			w,
		)
		w.Close()
	}

	logger.Info("Showing window for missing yt-dlp handling...")
	w.ShowAndRun()
}
