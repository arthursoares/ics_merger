package app

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/arthur/ical_merger/internal/config"
	"github.com/arthur/ical_merger/internal/ical"
	"github.com/arran4/golang-ical"
)

// Merger handles the merging of multiple calendars
type Merger struct {
	cfg *config.Config
}

// NewMerger creates a new Merger instance
func NewMerger(cfg *config.Config) *Merger {
	return &Merger{
		cfg: cfg,
	}
}

// Merge fetches all calendars and combines them into a single iCalendar file
func (m *Merger) Merge() error {
	calendars := make(map[string]*ics.Calendar)

	// Fetch each calendar
	for _, cal := range m.cfg.Calendars {
		log.Printf("Fetching calendar %s from %s", cal.Name, cal.URL)
		calendar, err := ical.FetchCalendar(cal.URL)
		if err != nil {
			log.Printf("Error fetching calendar %s: %v", cal.Name, err)
			continue
		}
		calendars[cal.Name] = calendar
	}

	if len(calendars) == 0 {
		log.Println("No calendars were successfully fetched. No output generated.")
		return nil
	}

	// Merge the calendars
	log.Println("Merging calendars")
	merged := ical.MergeCalendars(calendars)

	// Ensure we have at least one event in the merged calendar
	if len(merged.Events()) == 0 {
		log.Println("No events found in any calendar, creating dummy event")
		// Add a dummy event if the calendar is empty
		dummyEvent := ics.NewEvent("dummy-event-" + time.Now().Format("20060102150405"))
		dummyEvent.SetProperty(ics.ComponentPropertySummary, "Calendar Merger Info")
		dummyEvent.SetProperty(ics.ComponentPropertyDescription, "No valid events were found in any of the source calendars")
		now := time.Now()
		dummyEvent.SetProperty(ics.ComponentPropertyDtStart, now.Format("20060102T150405Z"))
		dummyEvent.SetProperty(ics.ComponentPropertyDtEnd, now.Add(time.Hour).Format("20060102T150405Z"))
		merged.AddVEvent(dummyEvent)
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(m.cfg.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	// Write the merged calendar to file
	file, err := os.Create(m.cfg.OutputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	log.Printf("Writing merged calendar to %s (%d events)", m.cfg.OutputPath, len(merged.Events()))
	
	// Serialize the calendar 
	output := merged.Serialize()
	
	// Apply Ruby compatibility fixes if needed
	fixedOutput := ical.RubyCompatibilityFixer(output, m.cfg.OutputTimezone)
	
	if _, err := file.WriteString(fixedOutput); err != nil {
		return err
	}
	
	return nil
}