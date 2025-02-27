package app

import (
	"log"
	"os"
	"path/filepath"

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
		return nil
	}

	// Merge the calendars
	log.Println("Merging calendars")
	merged := ical.MergeCalendars(calendars)

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

	log.Printf("Writing merged calendar to %s", m.cfg.OutputPath)
	// Serialize returns a string, not an error
	merged.Serialize(file)
	return nil
}