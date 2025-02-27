package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/arthur/ical_merger/internal/app"
	"github.com/arthur/ical_merger/internal/config"
)

func main() {
	// Command line flags
	var (
		serveMode      = flag.Bool("serve", false, "Run as HTTP server")
		httpAddr       = flag.String("addr", ":8080", "HTTP server address")
		configPath     = flag.String("config", "", "Path to config file (overrides CONFIG_PATH env var)")
		outputPath     = flag.String("output", "", "Path to output file (overrides config)")
		localMode      = flag.Bool("local", false, "Run with local files instead of URLs")
		calendarDir    = flag.String("calendar-dir", "./calendars", "Directory containing calendar files when using local mode")
	)
	flag.Parse()

	// Set config path from flag or env var
	if *configPath != "" {
		os.Setenv("CONFIG_PATH", *configPath)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Override output path if specified
	if *outputPath != "" {
		cfg.OutputPath = *outputPath
	}

	// Convert file paths to file:// URLs if in local mode
	if *localMode {
		for i := range cfg.Calendars {
			filename := filepath.Join(*calendarDir, cfg.Calendars[i].Name+".ics")
			cfg.Calendars[i].URL = "file://" + filename
			log.Printf("Using local file: %s", filename)
		}
	}

	merger := app.NewMerger(cfg)

	// Do initial merge
	if err := merger.Merge(); err != nil {
		log.Printf("Initial merge failed: %v", err)
	}

	// Run as HTTP server if in serve mode
	if *serveMode {
		// HTTP handler to serve the merged calendar
		http.HandleFunc("/calendar", func(w http.ResponseWriter, r *http.Request) {
			// Force merge to get the latest data
			if err := merger.Merge(); err != nil {
				http.Error(w, fmt.Sprintf("Error merging calendars: %v", err), http.StatusInternalServerError)
				return
			}

			// Open the merged calendar file
			file, err := os.Open(cfg.OutputPath)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error opening calendar file: %v", err), http.StatusInternalServerError)
				return
			}
			defer file.Close()

			// Set content type and headers
			w.Header().Set("Content-Type", "text/calendar")
			w.Header().Set("Content-Disposition", "attachment; filename=\"merged.ics\"")

			// Copy the file to the response
			if _, err := io.Copy(w, file); err != nil {
				http.Error(w, fmt.Sprintf("Error sending calendar: %v", err), http.StatusInternalServerError)
				return
			}
		})

		// HTTP handler for health check
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		// Start HTTP server in a goroutine
		go func() {
			log.Printf("Starting HTTP server on %s", *httpAddr)
			if err := http.ListenAndServe(*httpAddr, nil); err != nil {
				log.Fatalf("HTTP server failed: %v", err)
			}
		}()
	}

	// Set up periodic merges
	ticker := time.NewTicker(time.Duration(cfg.SyncIntervalMinutes) * time.Minute)
	defer ticker.Stop()

	log.Printf("iCal Merger started. Merging every %d minutes", cfg.SyncIntervalMinutes)

	for {
		select {
		case <-ticker.C:
			log.Println("Starting periodic merge")
			if err := merger.Merge(); err != nil {
				log.Printf("Periodic merge failed: %v", err)
			}
		}
	}
}