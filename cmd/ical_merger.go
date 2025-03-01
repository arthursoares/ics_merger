package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/arthur/ical_merger/internal/app"
	"github.com/arthur/ical_merger/internal/config"
	"github.com/arthur/ical_merger/internal/ical"
	"github.com/arran4/golang-ical"
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
			
			// Read the calendar file
			calData, err := io.ReadAll(file)
			if err != nil {
				log.Printf("Error reading calendar file: %v", err)
				http.Error(w, fmt.Sprintf("Error reading calendar file: %v", err), http.StatusInternalServerError)
				return
			}
			
			// Apply Ruby compatibility fixes
			fixedCalData := ical.RubyCompatibilityFixer(string(calData), cfg.OutputTimezone)

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
			
			// Write the fixed calendar data to the response
			if _, err := w.Write([]byte(fixedCalData)); err != nil {
				log.Printf("Error sending calendar: %v", err)
				http.Error(w, fmt.Sprintf("Error sending calendar: %v", err), http.StatusInternalServerError)
				return
			}
			log.Printf("Successfully served calendar to %s", r.RemoteAddr)
		})
		
		log.Printf("Calendar handler registered")
		
		// HTTP handler to serve calendar data as JSON for TRMNL plugin
		http.HandleFunc("/api/calendar", func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Calendar API request received from %s", r.RemoteAddr)
			
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
			
			// Get date range from query params or use defaults
			daysBack := 1
			daysForward := 30
			if days := r.URL.Query().Get("days_back"); days != "" {
				if val, err := strconv.Atoi(days); err == nil && val > 0 {
					daysBack = val
				}
			}
			if days := r.URL.Query().Get("days_forward"); days != "" {
				if val, err := strconv.Atoi(days); err == nil && val > 0 {
					daysForward = val
				}
			}
			
			// Filter events to the specified date range
			filteredCalendar := ical.FilterCalendarByDateRange(calendar, daysBack, daysForward)
			
			// Convert calendar events to JSON format
			type EventJSON struct {
				UID         string    `json:"uid"`
				Summary     string    `json:"summary"`
				StartTime   time.Time `json:"start_time"`
				EndTime     time.Time `json:"end_time,omitempty"`
				StartStr    string    `json:"start"`
				EndStr      string    `json:"end,omitempty"`
				Location    string    `json:"location,omitempty"`
				Description string    `json:"description,omitempty"`
				AllDay      bool      `json:"all_day"`
				Categories  []string  `json:"categories,omitempty"`
				Status      string    `json:"status,omitempty"`
			}
			
			events := []EventJSON{}
			now := time.Now()
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
			
			for _, event := range filteredCalendar.Events() {
				// Extract event properties
				uid := event.GetProperty(ics.ComponentPropertyUniqueId).Value
				summary := event.GetProperty(ics.ComponentPropertySummary).Value
				
				// Parse start time
				var startTime time.Time
				var isAllDay bool
				
				startProp := event.GetProperty(ics.ComponentPropertyDtStart)
				if startProp == nil {
					continue // Skip events without start time
				}
				
				// Try to parse start time
				dateFormats := []string{
					"20060102T150405Z",     // UTC
					"20060102T150405",      // Local
					"20060102",             // Date only (all day)
				}
				
				for _, format := range dateFormats {
					if t, err := time.Parse(format, startProp.Value); err == nil {
						startTime = t
						isAllDay = format == "20060102"
						break
					}
				}
				
				// Skip events we can't parse
				if startTime.IsZero() {
					log.Printf("Skipping event with unparseable date: %s", summary)
					continue
				}
				
				// Parse end time
				var endTime time.Time
				endProp := event.GetProperty(ics.ComponentPropertyDtEnd)
				if endProp != nil {
					for _, format := range dateFormats {
						if t, err := time.Parse(format, endProp.Value); err == nil {
							endTime = t
							break
						}
					}
				} else if isAllDay {
					// For all-day events without end date, assume same day
					endTime = startTime.AddDate(0, 0, 1)
				} else {
					// For timed events without end time, assume 1 hour
					endTime = startTime.Add(1 * time.Hour)
				}
				
				// Ensure we don't have a zero time for end_time
				if endTime.IsZero() {
					// Use fallback of 1 hour after start time
					endTime = startTime.Add(1 * time.Hour)
				}
				
				// Extract location if available
				var location string
				locProp := event.GetProperty(ics.ComponentPropertyLocation)
				if locProp != nil {
					location = locProp.Value
				}
				
				// Extract description if available
				var description string
				descProp := event.GetProperty(ics.ComponentPropertyDescription)
				if descProp != nil {
					description = descProp.Value
				}
				
				// Format times for display
				var startStr, endStr string
				if isAllDay {
					startStr = startTime.Format("Jan 2")
					if startTime.Year() != endTime.Year() || startTime.Month() != endTime.Month() || startTime.Day() != endTime.Day() {
						endStr = endTime.AddDate(0, 0, -1).Format("Jan 2") // Subtract a day because all-day end dates are exclusive
					}
				} else {
					if startTime.Year() == today.Year() && startTime.Month() == today.Month() && startTime.Day() == today.Day() {
						startStr = startTime.Format("3:04 PM") // Same day, just show time
					} else {
						startStr = startTime.Format("Jan 2 3:04 PM") // Different day, show date and time
					}
					
					if startTime.Year() == endTime.Year() && startTime.Month() == endTime.Month() && startTime.Day() == endTime.Day() {
						endStr = endTime.Format("3:04 PM") // Same day, just show time
					} else {
						endStr = endTime.Format("Jan 2 3:04 PM") // Different day, show date and time
					}
				}
				
				// Extract categories if available
				categories := []string{}
				catProps := event.GetProperties(ics.ComponentPropertyCategories)
				for _, prop := range catProps {
					categories = append(categories, strings.Split(prop.Value, ",")...)
				}
				
				// Extract status if available
				status := "confirmed" // Default status
				statusProp := event.GetProperty(ics.ComponentPropertyStatus)
				if statusProp != nil {
					status = strings.ToLower(statusProp.Value)
				}
				
				// Add event to result
				events = append(events, EventJSON{
					UID:         uid,
					Summary:     summary,
					StartTime:   startTime,
					EndTime:     endTime,
					StartStr:    startStr,
					EndStr:      endStr,
					Location:    location,
					Description: description,
					AllDay:      isAllDay,
					Categories:  categories,
					Status:      status,
				})
			}
			
			// Sort events by start time
			sort.Slice(events, func(i, j int) bool {
				return events[i].StartTime.Before(events[j].StartTime)
			})
			
			// Group events by day for easier template rendering
			eventsByDay := make(map[string][]EventJSON)
			for _, event := range events {
				dateStr := event.StartTime.Format("2006-01-02")
				eventsByDay[dateStr] = append(eventsByDay[dateStr], event)
			}
			
			// Create a slice of day keys sorted chronologically
			days := make([]string, 0, len(eventsByDay))
			for day := range eventsByDay {
				days = append(days, day)
			}
			sort.Strings(days)
			
			// Build a formatted result for template rendering
			type DayInfo struct {
				Date       string     `json:"date"`
				DateFmt    string     `json:"date_fmt"`
				Weekday    string     `json:"weekday"`
				IsToday    bool       `json:"is_today"`
				IsTomorrow bool       `json:"is_tomorrow"`
				Events     []EventJSON `json:"events"`
			}
			
			formattedDays := []DayInfo{}
			tomorrow := today.AddDate(0, 0, 1)
			
			for _, day := range days {
				date, _ := time.Parse("2006-01-02", day)
				
				isToday := date.Year() == today.Year() && date.Month() == today.Month() && date.Day() == today.Day()
				isTomorrow := date.Year() == tomorrow.Year() && date.Month() == tomorrow.Month() && date.Day() == tomorrow.Day()
				
				var dateFmt string
				if isToday {
					dateFmt = "Today"
				} else if isTomorrow {
					dateFmt = "Tomorrow"
				} else {
					dateFmt = date.Format("Monday, Jan 2")
				}
				
				formattedDays = append(formattedDays, DayInfo{
					Date:       day,
					DateFmt:    dateFmt,
					Weekday:    date.Format("Monday"),
					IsToday:    isToday,
					IsTomorrow: isTomorrow,
					Events:     eventsByDay[day],
				})
			}
			
			// Create the final response
			response := map[string]interface{}{
				"days":         formattedDays,
				"total_events": len(events),
				"date_range": map[string]string{
					"start": today.AddDate(0, 0, -daysBack).Format("2006-01-02"),
					"end":   today.AddDate(0, 0, daysForward).Format("2006-01-02"),
				},
				"generated_at": time.Now().Format(time.RFC3339),
			}
			
			// Set content type and headers
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "max-age=3600")
			
			// Serialize and return the JSON
			if err := json.NewEncoder(w).Encode(response); err != nil {
				log.Printf("Error encoding calendar JSON: %v", err)
				http.Error(w, fmt.Sprintf("Error encoding calendar JSON: %v", err), http.StatusInternalServerError)
				return
			}
			
			log.Printf("Successfully served calendar API to %s", r.RemoteAddr)
		})
		
		log.Printf("Calendar API handler registered")
		
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
			
			// Filter events to ±30 days (inclusive)
			log.Printf("Filtering calendar events for summary.ics (inclusive date range)")
			filteredCalendar := ical.FilterCalendarByDateRange(calendar, 30, 30)
			
			// Let the improved FilterCalendarByDateRange function handle all events 
			// including edge cases
			
			// Set content type and headers
			w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
			w.Header().Set("Content-Disposition", "attachment; filename=\"summary.ics\"")
			w.Header().Set("X-WR-CALNAME", "Summary Calendar")
			
			// Set caching headers
			maxAge := cfg.SyncIntervalMinutes * 60 // Convert minutes to seconds
			w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, public", maxAge))
			w.Header().Set("Transfer-Encoding", "identity")
			
			// Serialize the filtered calendar
			output := filteredCalendar.Serialize()
			
			// Add required properties for Ruby client compatibility
			// Insert the necessary calendar properties and VTIMEZONEs
			var enhancedLines []string
			
			// First add the calendar properties
			for _, line := range strings.Split(output, "\n") {
				enhancedLines = append(enhancedLines, line)
				if line == "VERSION:2.0" {
					enhancedLines = append(enhancedLines, "CALSCALE:GREGORIAN")
					enhancedLines = append(enhancedLines, "X-WR-CALNAME:Summary Calendar")
					enhancedLines = append(enhancedLines, "X-WR-TIMEZONE:"+cfg.OutputTimezone)
				}
			}
			
			// Fix any date formatting issues for Ruby icalendar library
			// Specifically all DATE-only fields need VALUE=DATE parameter
			for i, line := range enhancedLines {
				// Fix all-day events by adding VALUE=DATE
				if strings.HasPrefix(line, "DTSTART:") && len(line) == 17 { // Format: DTSTART:20250208
					// Check that we're not accidentally adding a duplicate parameter
					if !strings.Contains(line, "VALUE=DATE") {
						enhancedLines[i] = strings.Replace(line, "DTSTART:", "DTSTART;VALUE=DATE:", 1)
					}
				}
				if strings.HasPrefix(line, "DTEND:") && len(line) == 15 { // Format: DTEND:20250208
					// Check that we're not accidentally adding a duplicate parameter
					if !strings.Contains(line, "VALUE=DATE") {
						enhancedLines[i] = strings.Replace(line, "DTEND:", "DTEND;VALUE=DATE:", 1)
					}
				}
				
				// Fix any timezone info for recurring events
				if strings.HasPrefix(line, "DTSTART:") && strings.Contains(line, "T") { // timed event
					// Check that we're not accidentally adding a duplicate parameter
					if !strings.Contains(line, "TZID=") {
						enhancedLines[i] = strings.Replace(line, "DTSTART:", "DTSTART;TZID="+cfg.OutputTimezone+":", 1)
					}
				}
				if strings.HasPrefix(line, "DTEND:") && strings.Contains(line, "T") { // timed event
					// Check that we're not accidentally adding a duplicate parameter
					if !strings.Contains(line, "TZID=") {
						enhancedLines[i] = strings.Replace(line, "DTEND:", "DTEND;TZID="+cfg.OutputTimezone+":", 1)
					}
				}
			}
			
			// Add VTIMEZONE components after the METHOD line if not already present
			if !strings.Contains(output, "BEGIN:VTIMEZONE") {
				vtimezoneStr := fmt.Sprintf(`BEGIN:VTIMEZONE
TZID:%s
BEGIN:STANDARD
DTSTART:19701101T030000
TZOFFSETFROM:+0200
TZOFFSETTO:+0100
RRULE:FREQ=YEARLY;BYMONTH=10;BYDAY=-1SU
END:STANDARD
BEGIN:DAYLIGHT
DTSTART:19700329T020000
TZOFFSETFROM:+0100
TZOFFSETTO:+0200
RRULE:FREQ=YEARLY;BYMONTH=3;BYDAY=-1SU
END:DAYLIGHT
END:VTIMEZONE`, cfg.OutputTimezone)
				
				var finalLines []string
				addedTimezone := false
				
				for _, line := range enhancedLines {
					finalLines = append(finalLines, line)
					if line == "METHOD:PUBLISH" && !addedTimezone {
						finalLines = append(finalLines, strings.Split(vtimezoneStr, "\n")...)
						addedTimezone = true
					}
				}
				
				enhancedLines = finalLines
			}
			
			// Fix any malformed properties that might already have duplicate parameters
			// For example: DTEND;VALUE=DATE:;VALUE=DATE:20250219
			var fixedLines []string
			for _, line := range enhancedLines {
				// Fix malformed DTEND with duplicate parameters
				if strings.Contains(line, "DTEND;VALUE=DATE:;VALUE=DATE:") {
					line = strings.Replace(line, "DTEND;VALUE=DATE:;VALUE=DATE:", "DTEND;VALUE=DATE:", 1)
				}
				if strings.Contains(line, "DTEND;TZID="+cfg.OutputTimezone+":;TZID="+cfg.OutputTimezone+":") {
					line = strings.Replace(line, "DTEND;TZID="+cfg.OutputTimezone+":;TZID="+cfg.OutputTimezone+":", "DTEND;TZID="+cfg.OutputTimezone+":", 1)
				}
				fixedLines = append(fixedLines, line)
			}
			
			enhancedOutput := strings.Join(fixedLines, "\n")
			
			// Send the enhanced output
			if _, err := w.Write([]byte(enhancedOutput)); err != nil {
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