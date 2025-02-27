package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/arthur/ical_merger/internal/app"
	"github.com/arthur/ical_merger/internal/config"
	"github.com/arthur/ical_merger/internal/ical"
)

func main() {
	// Early log to confirm process started
	log.Printf("iCal Merger starting up...")
	
	// Set up panic handling
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC RECOVERED: %v", r)
			// Print stack trace
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			log.Printf("STACK TRACE: %s", buf[:n])
		}
	}()
	
	// Command line flags
	var (
		serveMode      = flag.Bool("serve", false, "Run as HTTP server")
		httpAddr       = flag.String("addr", ":8080", "HTTP server address")
		configPath     = flag.String("config", "", "Path to config file (overrides CONFIG_PATH env var)")
		outputPath     = flag.String("output", "", "Path to output file (overrides config)")
		localMode      = flag.Bool("local", false, "Run with local files instead of URLs")
		calendarDir    = flag.String("calendar-dir", "./calendars", "Directory containing calendar files when using local mode")
	)
	
	// Log command-line arguments
	log.Printf("Parsing command line flags: %v", os.Args)
	flag.Parse()
	log.Printf("serveMode=%v, httpAddr=%s", *serveMode, *httpAddr)

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
		log.Printf("Entering serve mode setup")
		
		// Add root handler for easy testing
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Received request for: %s", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("iCal Merger is running. Use /calendar to access the merged calendar."))
		})
		
		log.Printf("Root handler registered")
		
		// HTTP handler for health check - keep this simple to test basic functionality
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Health check request received from %s", r.RemoteAddr)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		
		log.Printf("Health check handler registered")
		
		// HTTP handler to serve the merged calendar
		http.HandleFunc("/calendar", func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Calendar request received from %s", r.RemoteAddr)
			
			// Only refresh cache if the nocache parameter is set
			if r.URL.Query().Get("nocache") != "" {
				log.Printf("Nocache parameter set, refreshing calendar data")
				if err := merger.Merge(); err != nil {
					log.Printf("Error merging calendars: %v", err)
					http.Error(w, fmt.Sprintf("Error merging calendars: %v", err), http.StatusInternalServerError)
					return
				}
			}

			log.Printf("Opening calendar file from: %s", cfg.OutputPath)
			// Open the merged calendar file
			file, err := os.Open(cfg.OutputPath)
			if err != nil {
				log.Printf("Error opening calendar file: %v", err)
				http.Error(w, fmt.Sprintf("Error opening calendar file: %v", err), http.StatusInternalServerError)
				return
			}
			defer file.Close()

			// Set content type and headers with charset explicitly specified
			w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
			w.Header().Set("Content-Disposition", "attachment; filename=\"merged.ics\"")
			
			// Set X-WR headers that some clients expect
			w.Header().Set("X-WR-CALNAME", "Merged Calendar")
			
			// Set caching headers based on the calendar sync interval
			maxAge := cfg.SyncIntervalMinutes * 60 // Convert minutes to seconds
			w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, public", maxAge))
			
			// Signal that the content is complete (not chunked)
			w.Header().Set("Transfer-Encoding", "identity")
			
			// Copy the file to the response
			if _, err := io.Copy(w, file); err != nil {
				log.Printf("Error sending calendar: %v", err)
				http.Error(w, fmt.Sprintf("Error sending calendar: %v", err), http.StatusInternalServerError)
				return
			}
			log.Printf("Successfully served calendar to %s", r.RemoteAddr)
		})
		
		log.Printf("Calendar handler registered")
		
		// HTTP handler to serve a summary calendar (±30 days from current date)
		http.HandleFunc("/summary", func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Summary calendar request received from %s", r.RemoteAddr)
			
			// Only refresh cache if the nocache parameter is set
			if r.URL.Query().Get("nocache") != "" {
				log.Printf("Nocache parameter set, refreshing calendar data")
				if err := merger.Merge(); err != nil {
					log.Printf("Error merging calendars: %v", err)
					http.Error(w, fmt.Sprintf("Error merging calendars: %v", err), http.StatusInternalServerError)
					return
				}
			}

			// Open the merged calendar file
			file, err := os.Open(cfg.OutputPath)
			if err != nil {
				log.Printf("Error opening calendar file: %v", err)
				http.Error(w, fmt.Sprintf("Error opening calendar file: %v", err), http.StatusInternalServerError)
				return
			}
			defer file.Close()
			
			// Parse the calendar
			calData, err := io.ReadAll(file)
			if err != nil {
				log.Printf("Error reading calendar file: %v", err)
				http.Error(w, fmt.Sprintf("Error reading calendar file: %v", err), http.StatusInternalServerError)
				return
			}
			
			calendar, err := ical.ParseCalendar(strings.NewReader(string(calData)))
			if err != nil {
				log.Printf("Error parsing calendar: %v", err)
				http.Error(w, fmt.Sprintf("Error parsing calendar: %v", err), http.StatusInternalServerError)
				return
			}
			
			// Filter events to ±30 days
			filteredCalendar := ical.FilterCalendarByDateRange(calendar, 30, 30)
			
			// Set content type and headers
			w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
			w.Header().Set("Content-Disposition", "attachment; filename=\"summary.ics\"")
			w.Header().Set("X-WR-CALNAME", "Summary Calendar")
			
			// Set caching headers
			maxAge := cfg.SyncIntervalMinutes * 60 // Convert minutes to seconds
			w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, public", maxAge))
			w.Header().Set("Transfer-Encoding", "identity")
			
			// Serialize and return the filtered calendar
			output := filteredCalendar.Serialize()
			if _, err := w.Write([]byte(output)); err != nil {
				log.Printf("Error sending summary calendar: %v", err)
				http.Error(w, fmt.Sprintf("Error sending summary calendar: %v", err), http.StatusInternalServerError)
				return
			}
			
			log.Printf("Successfully served summary calendar to %s", r.RemoteAddr)
		})
		
		log.Printf("Summary calendar handler registered")

		// Start HTTP server - correctly in a goroutine
		log.Printf("Starting HTTP server on %s", *httpAddr)
		go func() {
			err := http.ListenAndServe(*httpAddr, nil)
			log.Fatalf("*** HTTP server failed: %v ***", err)
		}()
		
		log.Printf("HTTP server is now running")
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